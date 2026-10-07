package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"time"

	"gorm.io/gorm"
)

// Board 是看板目录中看板的服务：创建、改名、移动、归档、恢复和主看板。
var Board = new(boardService)

type boardService struct{}

// findBoard 查找用户的看板。
func findBoard(ctx context.Context, userID, id int64) (*model.Board, error) {
	board, err := repository.Board.FindByID(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.BoardNotFound, "看板不存在")
	}
	return board, err
}

// activeBoard 查找用户未归档的看板。
func activeBoard(ctx context.Context, userID, id int64) (*model.Board, error) {
	board, err := findBoard(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if board.Archived() {
		return nil, badRequest(resp.BoardArchived, "看板已归档")
	}
	return board, nil
}

// Create 在文件夹（nil 为根）的第 index 位新建看板，index 为 nil 时排在末尾。
// 同时创建一个默认名称（随用户语言）的面板；新看板不设主面板，打开时进入这个面板。
func (s *boardService) Create(ctx context.Context, userID int64, folderID *int64, name string, index *int) (*model.Board, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	user, err := User.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	board := &model.Board{UserID: userID, FolderID: folderID, Name: name, CreatedAt: time.Now().UTC()}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if err := activeParent(ctx, userID, folderID); err != nil {
			return err
		}
		var err error
		if board.Position, err = placeInDirectory(ctx, userID, folderID, EntityBoard, 0, index); err != nil {
			return err
		}
		if err = repository.Board.Create(ctx, board); err != nil {
			return err
		}
		if err = recordBoard(ctx, board); err != nil {
			return err
		}
		_, err = createPanel(ctx, board, defaultPanelName(uiLang(ctx, user)))
		return err
	})
	if err != nil {
		return nil, err
	}
	return board, nil
}

// Rename 修改看板名称。
func (s *boardService) Rename(ctx context.Context, userID, id int64, name string) (*model.Board, error) {
	name, err := validateName(name)
	if err != nil {
		return nil, err
	}
	var board *model.Board
	err = write(ctx, func(ctx context.Context) error {
		if board, err = activeBoard(ctx, userID, id); err != nil {
			return err
		}
		board.Name = name
		if err = repository.Board.Update(ctx, userID, id, map[string]any{"name": name}); err != nil {
			return err
		}
		return recordBoard(ctx, board)
	})
	return board, err
}

// Move 把看板移到文件夹（nil 为根）的第 index 位，index 为 nil 时排在末尾。
func (s *boardService) Move(ctx context.Context, userID, id int64, folderID *int64, index *int) (*model.Board, error) {
	var board *model.Board
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if board, err = activeBoard(ctx, userID, id); err != nil {
			return err
		}
		return placeBoard(ctx, board, folderID, index)
	})
	return board, err
}

// Archive 把看板放入看板归档。主看板被归档时清空主看板指向；其中正在计时的定时器停止。
func (s *boardService) Archive(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		board, err := activeBoard(ctx, userID, id)
		if err != nil {
			return err
		}
		now := nowUTC()
		if err = stopTimersInBoards(ctx, userID, []int64{id}, now); err != nil {
			return err
		}
		if err = archiveBoard(ctx, board, now); err != nil {
			return err
		}
		if err = clearMainBoard(ctx, userID, []int64{id}); err != nil {
			return err
		}
		// 归档后其中的卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}

// Restore 把看板归档中的看板恢复到现存的父级（nil 为根）的第 index 位，index 为 nil 时排在末尾。
// 看板连同仍在其中的内容回到目录；不会回到已归档的原文件夹，也不会一并恢复同批归档的其他文件夹或看板。
func (s *boardService) Restore(ctx context.Context, userID, id int64, folderID *int64, index *int) (*model.Board, error) {
	var board *model.Board
	err := write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if board, err = findBoard(ctx, userID, id); err != nil {
			return err
		}
		if !board.Archived() {
			return badRequest(resp.BoardNotArchived, "看板不在看板归档中")
		}
		board.ArchivedAt = nil
		if err := placeBoard(ctx, board, folderID, index); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return board, err
}

// SetMain 把看板设为主看板；id 为 nil 时取消主看板。主看板必须是未归档的看板。
func (s *boardService) SetMain(ctx context.Context, userID int64, id *int64) (*model.User, error) {
	var user *model.User
	err := write(ctx, func(ctx context.Context) error {
		if id != nil {
			if _, err := activeBoard(ctx, userID, *id); err != nil {
				return err
			}
		}
		if err := repository.User.Update(ctx, userID, map[string]any{"main_board_id": id}); err != nil {
			return err
		}
		var err error
		user, err = recordSettings(ctx, userID)
		return err
	})
	return user, err
}

// placeBoard 把看板放到目标父级的第 index 位，连同看板当前的归档时间一起写入，并记录变更。
// 恢复时调用方先清空 board.ArchivedAt。
func placeBoard(ctx context.Context, board *model.Board, folderID *int64, index *int) error {
	if err := activeParent(ctx, board.UserID, folderID); err != nil {
		return err
	}
	position, err := placeInDirectory(ctx, board.UserID, folderID, EntityBoard, board.ID, index)
	if err != nil {
		return err
	}
	board.FolderID, board.Position = folderID, position
	err = repository.Board.Update(ctx, board.UserID, board.ID, map[string]any{
		"folder_id": folderID, "position": position, "archived_at": board.ArchivedAt,
	})
	if err != nil {
		return err
	}
	return recordBoard(ctx, board)
}

// archiveBoard 记录看板的归档时间并记录变更。必须在 write 中调用。
func archiveBoard(ctx context.Context, board *model.Board, at time.Time) error {
	board.ArchivedAt = &at
	if err := repository.Board.Update(ctx, board.UserID, board.ID, map[string]any{"archived_at": at}); err != nil {
		return err
	}
	return recordBoard(ctx, board)
}
