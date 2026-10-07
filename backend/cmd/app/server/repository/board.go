package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// Board 是看板表的仓储。查询都带用户条件，只能取到该用户自己的看板。
var Board = new(boardRepository)

type boardRepository struct{}

// Create 新建看板。
func (*boardRepository) Create(ctx context.Context, board *model.Board) error {
	return conn(ctx).Create(board).Error
}

// FindByID 按 ID 查找用户的看板。
func (*boardRepository) FindByID(ctx context.Context, userID, id int64) (*model.Board, error) {
	board := new(model.Board)
	return board, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(board).Error
}

// ListByUser 返回用户的全部看板，包括已归档的。
func (*boardRepository) ListByUser(ctx context.Context, userID int64) ([]*model.Board, error) {
	boards := make([]*model.Board, 0)
	return boards, conn(ctx).Where("user_id = ?", userID).Order("position ASC").Order("id ASC").Find(&boards).Error
}

// ListActiveChildren 按顺序返回父级下未归档的看板，folderID 为 nil 表示根。
func (*boardRepository) ListActiveChildren(ctx context.Context, userID int64, folderID *int64) ([]*model.Board, error) {
	boards := make([]*model.Board, 0)
	tx := conn(ctx).Where("user_id = ? AND archived_at IS NULL", userID)
	if folderID == nil {
		tx = tx.Where("folder_id IS NULL")
	} else {
		tx = tx.Where("folder_id = ?", *folderID)
	}
	return boards, tx.Order("position ASC").Order("id ASC").Find(&boards).Error
}

// ListActiveInFolders 返回位于这些文件夹中、尚未归档的看板。
func (*boardRepository) ListActiveInFolders(ctx context.Context, userID int64, folderIDs []int64) ([]*model.Board, error) {
	boards := make([]*model.Board, 0)
	return boards, conn(ctx).Where("user_id = ? AND archived_at IS NULL AND folder_id IN ?", userID, folderIDs).Find(&boards).Error
}

// Update 按 ID 更新用户看板的指定列。values 中的 nil 会写成 NULL。
func (*boardRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Board{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// DeleteByUser 删除用户的全部看板。
func (*boardRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Board{}).Error
}
