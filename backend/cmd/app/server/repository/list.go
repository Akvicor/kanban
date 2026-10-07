package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// List 是列表和列表操作配置的仓储。查询都带用户条件，只能取到该用户自己的数据。
var List = new(listRepository)

type listRepository struct{}

// Create 新建列表。
func (*listRepository) Create(ctx context.Context, list *model.List) error {
	return conn(ctx).Create(list).Error
}

// FindByID 按 ID 查找用户的列表。
func (*listRepository) FindByID(ctx context.Context, userID, id int64) (*model.List, error) {
	list := new(model.List)
	return list, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(list).Error
}

// ListByPanel 按从左到右的顺序返回面板的全部列表，包括列表归档中的。
func (*listRepository) ListByPanel(ctx context.Context, userID, panelID int64) ([]*model.List, error) {
	lists := make([]*model.List, 0)
	return lists, conn(ctx).Where("user_id = ? AND panel_id = ?", userID, panelID).Order("position ASC").Order("id ASC").Find(&lists).Error
}

// ListActiveByPanel 按从左到右的顺序返回面板中不在列表归档中的列表。
func (*listRepository) ListActiveByPanel(ctx context.Context, userID, panelID int64) ([]*model.List, error) {
	lists := make([]*model.List, 0)
	return lists, conn(ctx).Where("user_id = ? AND panel_id = ? AND archived_at IS NULL", userID, panelID).
		Order("position ASC").Order("id ASC").Find(&lists).Error
}

// Update 按 ID 更新用户列表的指定列。values 中的 nil 会写成 NULL。
func (*listRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.List{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// LabelRules 返回这些列表的标签规则。
func (*listRepository) LabelRules(ctx context.Context, userID int64, listIDs []int64) ([]*model.ListLabelRule, error) {
	rules := make([]*model.ListLabelRule, 0)
	return rules, conn(ctx).Where("user_id = ? AND list_id IN ?", userID, listIDs).Order("id ASC").Find(&rules).Error
}

// TimeRules 返回这些列表的时间规则。
func (*listRepository) TimeRules(ctx context.Context, userID int64, listIDs []int64) ([]*model.ListTimeRule, error) {
	rules := make([]*model.ListTimeRule, 0)
	return rules, conn(ctx).Where("user_id = ? AND list_id IN ?", userID, listIDs).Order("id ASC").Find(&rules).Error
}

// ReplaceRules 用新的规则替换列表的全部操作配置。
func (*listRepository) ReplaceRules(ctx context.Context, userID, listID int64, labelRules []*model.ListLabelRule, timeRules []*model.ListTimeRule) error {
	if err := conn(ctx).Where("user_id = ? AND list_id = ?", userID, listID).Delete(&model.ListLabelRule{}).Error; err != nil {
		return err
	}
	if err := conn(ctx).Where("user_id = ? AND list_id = ?", userID, listID).Delete(&model.ListTimeRule{}).Error; err != nil {
		return err
	}
	if len(labelRules) > 0 {
		if err := conn(ctx).Create(&labelRules).Error; err != nil {
			return err
		}
	}
	if len(timeRules) > 0 {
		if err := conn(ctx).Create(&timeRules).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListIDsUsingLabel 返回操作配置中引用了该标签的列表 ID。
func (*listRepository) ListIDsUsingLabel(ctx context.Context, userID, labelID int64) ([]int64, error) {
	ids := make([]int64, 0)
	return ids, conn(ctx).Model(&model.ListLabelRule{}).Where("user_id = ? AND label_id = ?", userID, labelID).
		Distinct("list_id").Pluck("list_id", &ids).Error
}

// DeleteLabelRules 删除引用该标签的全部标签规则。
func (*listRepository) DeleteLabelRules(ctx context.Context, userID, labelID int64) error {
	return conn(ctx).Where("user_id = ? AND label_id = ?", userID, labelID).Delete(&model.ListLabelRule{}).Error
}

// DeleteByUser 删除用户的全部列表和操作配置。
func (*listRepository) DeleteByUser(ctx context.Context, userID int64) error {
	if err := conn(ctx).Where("user_id = ?", userID).Delete(&model.ListLabelRule{}).Error; err != nil {
		return err
	}
	if err := conn(ctx).Where("user_id = ?", userID).Delete(&model.ListTimeRule{}).Error; err != nil {
		return err
	}
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.List{}).Error
}
