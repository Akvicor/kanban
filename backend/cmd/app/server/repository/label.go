package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// Label 是标签表的仓储。查询都带用户条件，只能取到该用户自己的标签。
var Label = new(labelRepository)

type labelRepository struct{}

// Create 新建标签。
func (*labelRepository) Create(ctx context.Context, label *model.Label) error {
	return conn(ctx).Create(label).Error
}

// FindByID 按 ID 查找用户的标签。
func (*labelRepository) FindByID(ctx context.Context, userID, id int64) (*model.Label, error) {
	label := new(model.Label)
	return label, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(label).Error
}

// ListByUser 返回用户的全部标签。
func (*labelRepository) ListByUser(ctx context.Context, userID int64) ([]*model.Label, error) {
	labels := make([]*model.Label, 0)
	return labels, conn(ctx).Where("user_id = ?", userID).Order("position ASC").Order("id ASC").Find(&labels).Error
}

// ListByPanel 按顺序返回面板的标签。
func (*labelRepository) ListByPanel(ctx context.Context, userID, panelID int64) ([]*model.Label, error) {
	labels := make([]*model.Label, 0)
	return labels, conn(ctx).Where("user_id = ? AND panel_id = ?", userID, panelID).Order("position ASC").Order("id ASC").Find(&labels).Error
}

// Update 按 ID 更新用户标签的指定列。
func (*labelRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Label{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// Delete 删除用户的标签。
func (*labelRepository) Delete(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.Label{}))
}

// DeleteByUser 删除用户的全部标签。
func (*labelRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Label{}).Error
}
