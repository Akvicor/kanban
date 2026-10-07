package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"slices"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

// 同步的实体类型。
const (
	EntityFolder = "folder"
	EntityBoard  = "board"
)

const (
	// maxFolderDepth 是文件夹的最大深度。根下第一层的深度为 1，看板不增加深度。
	maxFolderDepth = 16
	// nameMaxLength 是文件夹、看板和面板名称的最大字符数。
	nameMaxLength = 100
)

// validateName 去掉首尾空白，检查名称长度为 1 到 100 个字符。用于文件夹、看板和面板。
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", badRequest(resp.NameEmpty, "名称不能为空")
	}
	if utf8.RuneCountInString(name) > nameMaxLength {
		return "", badRequest(resp.NameTooLong, "名称不能超过 100 个字符")
	}
	return name, nil
}

// sibling 是目录中同一父级下的一项：文件夹或看板，二者共用排序。
type sibling struct {
	folder *model.Folder
	board  *model.Board
}

func (s sibling) position() int64 {
	if s.folder != nil {
		return s.folder.Position
	}
	return s.board.Position
}

// is 判断该项是否为指定的文件夹或看板。
func (s sibling) is(kind string, id int64) bool {
	if s.folder != nil {
		return kind == EntityFolder && s.folder.ID == id
	}
	return kind == EntityBoard && s.board.ID == id
}

// activeSiblings 按目录顺序返回父级下未归档的文件夹和看板，排除 kind/id 指定的那一项。
// 位置相同时文件夹在前，再按 ID 排列，与前端一致。
func activeSiblings(ctx context.Context, userID int64, parentID *int64, excludeKind string, excludeID int64) ([]sibling, error) {
	folders, err := repository.Folder.ListActiveChildren(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}
	boards, err := repository.Board.ListActiveChildren(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}
	list := make([]sibling, 0, len(folders)+len(boards))
	for _, folder := range folders {
		list = append(list, sibling{folder: folder})
	}
	for _, board := range boards {
		list = append(list, sibling{board: board})
	}
	list = slices.DeleteFunc(list, func(s sibling) bool { return s.is(excludeKind, excludeID) })
	slices.SortStableFunc(list, func(a, b sibling) int {
		if a.position() != b.position() {
			return compareInt64(a.position(), b.position())
		}
		if (a.folder != nil) != (b.folder != nil) {
			if a.folder != nil {
				return -1
			}
			return 1
		}
		return compareInt64(siblingID(a), siblingID(b))
	})
	return list, nil
}

func siblingID(s sibling) int64 {
	if s.folder != nil {
		return s.folder.ID
	}
	return s.board.ID
}

func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// placeInDirectory 返回把 kind/id 放到父级下第 index 位（从 0 开始，不计它自己）时应使用的排序值。
// index 为 nil 或超出范围时放在末尾。需要重新排列同级时，修改它们的排序值并记录变更。
func placeInDirectory(ctx context.Context, userID int64, parentID *int64, kind string, id int64, index *int) (int64, error) {
	siblings, err := activeSiblings(ctx, userID, parentID, kind, id)
	if err != nil {
		return 0, err
	}
	positions := make([]int64, len(siblings))
	for i, s := range siblings {
		positions[i] = s.position()
	}
	position, renumber := slotPosition(positions, index)
	for i, s := range siblings {
		if value, ok := renumber[i]; ok {
			if err = setSiblingPosition(ctx, s, value); err != nil {
				return 0, err
			}
		}
	}
	return position, nil
}

// setSiblingPosition 修改一项的排序值并记录变更。
func setSiblingPosition(ctx context.Context, s sibling, position int64) error {
	if s.folder != nil {
		s.folder.Position = position
		if err := repository.Folder.Update(ctx, s.folder.UserID, s.folder.ID, map[string]any{"position": position}); err != nil {
			return err
		}
		return recordFolder(ctx, s.folder)
	}
	s.board.Position = position
	if err := repository.Board.Update(ctx, s.board.UserID, s.board.ID, map[string]any{"position": position}); err != nil {
		return err
	}
	return recordBoard(ctx, s.board)
}

// folderTree 是用户全部文件夹（含已归档）的父子关系，用于计算深度和子树。
type folderTree struct {
	byID     map[int64]*model.Folder
	children map[int64][]int64
}

func loadFolderTree(ctx context.Context, userID int64) (*folderTree, error) {
	folders, err := repository.Folder.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	tree := &folderTree{byID: map[int64]*model.Folder{}, children: map[int64][]int64{}}
	for _, folder := range folders {
		tree.byID[folder.ID] = folder
		if folder.ParentID != nil {
			tree.children[*folder.ParentID] = append(tree.children[*folder.ParentID], folder.ID)
		}
	}
	return tree, nil
}

// depth 返回文件夹的深度，根下第一层为 1；id 为 nil 表示根，深度为 0。
func (t *folderTree) depth(id *int64) int {
	depth := 0
	for id != nil {
		depth++
		folder, ok := t.byID[*id]
		if !ok || depth > maxFolderDepth+1 {
			break
		}
		id = folder.ParentID
	}
	return depth
}

// height 返回以 id 为根的子树层数，只有它自己时为 1。已归档的下级也计入，因为它们在归档中保留层级。
func (t *folderTree) height(id int64) int {
	height := 0
	for _, child := range t.children[id] {
		height = max(height, t.height(child))
	}
	return height + 1
}

// descendants 返回 id 及其全部下级文件夹的 ID。
func (t *folderTree) descendants(id int64) []int64 {
	list := []int64{id}
	for _, child := range t.children[id] {
		list = append(list, t.descendants(child)...)
	}
	return list
}

// activeParent 检查目标父级可用：nil 表示根；否则必须是该用户未归档的文件夹。
func activeParent(ctx context.Context, userID int64, parentID *int64) error {
	if parentID == nil {
		return nil
	}
	folder, err := repository.Folder.FindByID(ctx, userID, *parentID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound(resp.TargetFolderNotFound, "目标文件夹不存在")
	}
	if err != nil {
		return err
	}
	if folder.Archived() {
		return badRequest(resp.TargetFolderArchived, "目标文件夹已归档")
	}
	return nil
}

// recordFolder 记录文件夹新增或变化。
func recordFolder(ctx context.Context, folder *model.Folder) error {
	return recordChange(ctx, folder.UserID, hub.OpUpsert, EntityFolder, folder.ID, dro.NewFolder(folder))
}

// recordBoard 记录看板新增或变化。
func recordBoard(ctx context.Context, board *model.Board) error {
	return recordChange(ctx, board.UserID, hub.OpUpsert, EntityBoard, board.ID, dro.NewBoard(board))
}

// recordSettings 记录个人设置变化，数据取库中的最新值。
func recordSettings(ctx context.Context, userID int64) (*model.User, error) {
	user, err := User.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, recordChange(ctx, userID, hub.OpUpsert, EntitySettings, userID, dro.NewSettings(user))
}

// clearMainBoard 在主看板属于 boardIDs 时清空用户的主看板指向。
func clearMainBoard(ctx context.Context, userID int64, boardIDs []int64) error {
	user, err := User.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.MainBoardID == nil || !slices.Contains(boardIDs, *user.MainBoardID) {
		return nil
	}
	if err = repository.User.Update(ctx, userID, map[string]any{"main_board_id": nil}); err != nil {
		return err
	}
	_, err = recordSettings(ctx, userID)
	return err
}
