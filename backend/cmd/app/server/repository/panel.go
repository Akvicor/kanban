package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// Panel 是面板表的仓储。查询都带用户条件，只能取到该用户自己的面板。
var Panel = new(panelRepository)

type panelRepository struct{}

// Create 新建面板。
func (*panelRepository) Create(ctx context.Context, panel *model.Panel) error {
	return conn(ctx).Create(panel).Error
}

// FindByID 按 ID 查找用户的面板。
func (*panelRepository) FindByID(ctx context.Context, userID, id int64) (*model.Panel, error) {
	panel := new(model.Panel)
	return panel, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(panel).Error
}

// ListByUser 返回用户的全部面板，包括已归档的。
func (*panelRepository) ListByUser(ctx context.Context, userID int64) ([]*model.Panel, error) {
	panels := make([]*model.Panel, 0)
	return panels, conn(ctx).Where("user_id = ?", userID).Order("position ASC").Order("id ASC").Find(&panels).Error
}

// ListActiveInBoard 按标签页顺序返回看板中未进入面板归档的面板。
func (*panelRepository) ListActiveInBoard(ctx context.Context, userID, boardID int64) ([]*model.Panel, error) {
	panels := make([]*model.Panel, 0)
	return panels, conn(ctx).Where("user_id = ? AND board_id = ? AND archived_at IS NULL", userID, boardID).
		Order("position ASC").Order("id ASC").Find(&panels).Error
}

// Update 按 ID 更新用户面板的指定列。values 中的 nil 会写成 NULL。
func (*panelRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Panel{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// DeleteByUser 删除用户的全部面板。
func (*panelRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Panel{}).Error
}
