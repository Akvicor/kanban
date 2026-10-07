package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// PriorityLevel 是优先级挡位表的仓储。查询都带用户条件，只能取到该用户自己的挡位。
var PriorityLevel = new(priorityLevelRepository)

type priorityLevelRepository struct{}

// Create 新建挡位。
func (*priorityLevelRepository) Create(ctx context.Context, level *model.PriorityLevel) error {
	return conn(ctx).Create(level).Error
}

// FindByID 按 ID 查找用户的挡位。
func (*priorityLevelRepository) FindByID(ctx context.Context, userID, id int64) (*model.PriorityLevel, error) {
	level := new(model.PriorityLevel)
	return level, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(level).Error
}

// ListByUser 返回用户的全部挡位。
func (*priorityLevelRepository) ListByUser(ctx context.Context, userID int64) ([]*model.PriorityLevel, error) {
	levels := make([]*model.PriorityLevel, 0)
	return levels, conn(ctx).Where("user_id = ?", userID).Order("position ASC").Order("id ASC").Find(&levels).Error
}

// ListByPanel 按顺序返回面板的挡位。
func (*priorityLevelRepository) ListByPanel(ctx context.Context, userID, panelID int64) ([]*model.PriorityLevel, error) {
	levels := make([]*model.PriorityLevel, 0)
	return levels, conn(ctx).Where("user_id = ? AND panel_id = ?", userID, panelID).Order("position ASC").Order("id ASC").Find(&levels).Error
}

// Update 按 ID 更新用户挡位的指定列。
func (*priorityLevelRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.PriorityLevel{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// Delete 删除用户的挡位。
func (*priorityLevelRepository) Delete(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.PriorityLevel{}))
}

// DeleteByUser 删除用户的全部挡位。
func (*priorityLevelRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.PriorityLevel{}).Error
}
