package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"slices"
	"time"

	"gorm.io/gorm"
)

// Folder 是看板目录中文件夹的服务。
var Folder = new(folderService)

type folderService struct{}

// activeFolder 查找用户未归档的文件夹。
func activeFolder(ctx context.Context, userID, id int64) (*model.Folder, error) {
	folder, err := repository.Folder.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.FolderNotFound, "文件夹不存在")
	}
	if err != nil {
		return nil, err
	}
	if folder.Archived() {
		return nil, badRequest(resp.FolderArchived, "文件夹已归档")
	}
	return folder, nil
}

// Create 在父级（nil 为根）的第 index 位新建文件夹，index 为 nil 时排在末尾。
func (s *folderService) Create(ctx context.Context, userID int64, parentID *int64, name string, index *int) (*model.Folder, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	folder := &model.Folder{UserID: userID, ParentID: parentID, Name: name, CreatedAt: time.Now().UTC()}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if err := activeParent(ctx, userID, parentID); err != nil {
			return err
		}
		tree, err := loadFolderTree(ctx, userID)
		if err != nil {
			return err
		}
		if tree.depth(parentID)+1 > maxFolderDepth {
			return badRequest(resp.FolderDepthLimit, "文件夹最多 16 层")
		}
		if folder.Position, err = placeInDirectory(ctx, userID, parentID, EntityFolder, 0, index); err != nil {
			return err
		}
		if err = repository.Folder.Create(ctx, folder); err != nil {
			return err
		}
		return recordFolder(ctx, folder)
	})
	if err != nil {
		return nil, err
	}
	return folder, nil
}

// Rename 修改文件夹名称。
func (s *folderService) Rename(ctx context.Context, userID, id int64, name string) (*model.Folder, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	var folder *model.Folder
	err = write(ctx, func(ctx context.Context) error {
		if folder, err = activeFolder(ctx, userID, id); err != nil {
			return err
		}
		folder.Name = name
		if err = repository.Folder.Update(ctx, userID, id, map[string]any{"name": name}); err != nil {
			return err
		}
		return recordFolder(ctx, folder)
	})
	return folder, err
}

// Move 把文件夹连同其内容移到父级（nil 为根）的第 index 位，index 为 nil 时排在末尾。
// 不能移到自己或自己的下级中，移动后整个子树的深度不能超过 16。
func (s *folderService) Move(ctx context.Context, userID, id int64, parentID *int64, index *int) (*model.Folder, error) {
	var folder *model.Folder
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if folder, err = activeFolder(ctx, userID, id); err != nil {
			return err
		}
		if err = activeParent(ctx, userID, parentID); err != nil {
			return err
		}
		tree, err := loadFolderTree(ctx, userID)
		if err != nil {
			return err
		}
		if parentID != nil && slices.Contains(tree.descendants(id), *parentID) {
			return badRequest(resp.FolderSelfMove, "不能把文件夹移到它自己或它的下级中")
		}
		if tree.depth(parentID)+tree.height(id) > maxFolderDepth {
			return badRequest(resp.FolderDepthExceeded, "移动后文件夹会超过 16 层")
		}
		if folder.Position, err = placeInDirectory(ctx, userID, parentID, EntityFolder, id, index); err != nil {
			return err
		}
		folder.ParentID = parentID
		if err = repository.Folder.Update(ctx, userID, id, map[string]any{"parent_id": parentID, "position": folder.Position}); err != nil {
			return err
		}
		return recordFolder(ctx, folder)
	})
	return folder, err
}

// Archive 归档文件夹：该文件夹、全部下级文件夹，以及这些文件夹中尚未归档的看板，在同一时刻进入看板归档。
// 层级关系保持不变，在看板归档中按原层级展示。主看板随之归档时清空主看板指向；其中正在计时的定时器停止。
func (s *folderService) Archive(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, err := activeFolder(ctx, userID, id); err != nil {
			return err
		}
		tree, err := loadFolderTree(ctx, userID)
		if err != nil {
			return err
		}
		now := nowUTC()
		folderIDs := tree.descendants(id)
		archived := make([]int64, 0, len(folderIDs))
		for _, folderID := range folderIDs {
			folder := tree.byID[folderID]
			if folder.Archived() {
				continue
			}
			folder.ArchivedAt = &now
			archived = append(archived, folderID)
			if err = recordFolder(ctx, folder); err != nil {
				return err
			}
		}
		if err = repository.Folder.Archive(ctx, userID, archived, now); err != nil {
			return err
		}

		boards, err := repository.Board.ListActiveInFolders(ctx, userID, folderIDs)
		if err != nil {
			return err
		}
		boardIDs := make([]int64, 0, len(boards))
		for _, board := range boards {
			boardIDs = append(boardIDs, board.ID)
		}
		if err = stopTimersInBoards(ctx, userID, boardIDs, now); err != nil {
			return err
		}
		for _, board := range boards {
			if err = archiveBoard(ctx, board, now); err != nil {
				return err
			}
		}
		if err = clearMainBoard(ctx, userID, boardIDs); err != nil {
			return err
		}
		// 其中的看板已归档，卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}
