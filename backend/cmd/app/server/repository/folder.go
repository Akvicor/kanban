package repository

import (
	"context"
	"kanban/cmd/app/server/model"
	"time"
)

// Folder 是文件夹表的仓储。查询都带用户条件，只能取到该用户自己的文件夹。
var Folder = new(folderRepository)

type folderRepository struct{}

// Create 新建文件夹。
func (*folderRepository) Create(ctx context.Context, folder *model.Folder) error {
	return conn(ctx).Create(folder).Error
}

// FindByID 按 ID 查找用户的文件夹。
func (*folderRepository) FindByID(ctx context.Context, userID, id int64) (*model.Folder, error) {
	folder := new(model.Folder)
	return folder, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(folder).Error
}

// ListByUser 返回用户的全部文件夹，包括已归档的。
func (*folderRepository) ListByUser(ctx context.Context, userID int64) ([]*model.Folder, error) {
	folders := make([]*model.Folder, 0)
	return folders, conn(ctx).Where("user_id = ?", userID).Order("position ASC").Order("id ASC").Find(&folders).Error
}

// ListActiveChildren 按顺序返回父级下未归档的文件夹，parentID 为 nil 表示根。
func (*folderRepository) ListActiveChildren(ctx context.Context, userID int64, parentID *int64) ([]*model.Folder, error) {
	folders := make([]*model.Folder, 0)
	tx := conn(ctx).Where("user_id = ? AND archived_at IS NULL", userID)
	if parentID == nil {
		tx = tx.Where("parent_id IS NULL")
	} else {
		tx = tx.Where("parent_id = ?", *parentID)
	}
	return folders, tx.Order("position ASC").Order("id ASC").Find(&folders).Error
}

// Update 按 ID 更新用户文件夹的指定列。
func (*folderRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Folder{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// Archive 把指定的文件夹标记为已归档。
func (*folderRepository) Archive(ctx context.Context, userID int64, ids []int64, at time.Time) error {
	return conn(ctx).Model(&model.Folder{}).Where("user_id = ? AND id IN ?", userID, ids).Update("archived_at", at).Error
}

// DeleteByUser 删除用户的全部文件夹。
func (*folderRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Folder{}).Error
}
