package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// Card 是卡片和卡片标签的仓储。查询都带用户条件，只能取到该用户自己的数据。
var Card = new(cardRepository)

type cardRepository struct{}

// Create 新建卡片。
func (*cardRepository) Create(ctx context.Context, card *model.Card) error {
	return conn(ctx).Create(card).Error
}

// FindByID 按 ID 查找用户的卡片。
func (*cardRepository) FindByID(ctx context.Context, userID, id int64) (*model.Card, error) {
	card := new(model.Card)
	return card, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(card).Error
}

// ListInList 按隐藏序号返回列表中的卡片（不含卡片归档中的）。
func (*cardRepository) ListInList(ctx context.Context, userID, listID int64) ([]*model.Card, error) {
	cards := make([]*model.Card, 0)
	return cards, conn(ctx).Where("user_id = ? AND list_id = ? AND archived_at IS NULL", userID, listID).
		Order("position ASC").Order("id ASC").Find(&cards).Error
}

// ListInLists 返回这些列表中的卡片（不含卡片归档中的）。
func (*cardRepository) ListInLists(ctx context.Context, userID int64, listIDs []int64) ([]*model.Card, error) {
	cards := make([]*model.Card, 0)
	if len(listIDs) == 0 {
		return cards, nil
	}
	return cards, conn(ctx).Where("user_id = ? AND list_id IN ? AND archived_at IS NULL", userID, listIDs).
		Order("position ASC").Order("id ASC").Find(&cards).Error
}

// ListArchivedInPanel 返回面板卡片归档中的卡片，最近归档的在前。
func (*cardRepository) ListArchivedInPanel(ctx context.Context, userID, panelID int64) ([]*model.Card, error) {
	cards := make([]*model.Card, 0)
	return cards, conn(ctx).Where("user_id = ? AND panel_id = ? AND archived_at IS NOT NULL", userID, panelID).
		Order("archived_at DESC").Order("id DESC").Find(&cards).Error
}

// ListRunningTimersInPanels 返回这些面板中定时器正在计时的卡片。
func (*cardRepository) ListRunningTimersInPanels(ctx context.Context, userID int64, panelIDs []int64) ([]*model.Card, error) {
	cards := make([]*model.Card, 0)
	if len(panelIDs) == 0 {
		return cards, nil
	}
	return cards, conn(ctx).Where("user_id = ? AND panel_id IN ? AND timer_started_at IS NOT NULL", userID, panelIDs).Find(&cards).Error
}

// ListByPriorityLevel 返回使用该优先级挡位的卡片。
func (*cardRepository) ListByPriorityLevel(ctx context.Context, userID, levelID int64) ([]*model.Card, error) {
	cards := make([]*model.Card, 0)
	return cards, conn(ctx).Where("user_id = ? AND priority_level_id = ?", userID, levelID).Find(&cards).Error
}

// Update 按 ID 更新用户卡片的指定列。values 中的 nil 会写成 NULL。
func (*cardRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Card{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// LabelIDs 返回这些卡片的标签，按卡片归组。
func (*cardRepository) LabelIDs(ctx context.Context, userID int64, cardIDs []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(cardIDs))
	if len(cardIDs) == 0 {
		return result, nil
	}
	rows := make([]*model.CardLabel, 0)
	if err := conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("label_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CardID] = append(result[row.CardID], row.LabelID)
	}
	return result, nil
}

// SetLabels 用新的标签集合替换卡片原有的标签。
func (*cardRepository) SetLabels(ctx context.Context, userID, cardID int64, labelIDs []int64) error {
	if err := conn(ctx).Where("user_id = ? AND card_id = ?", userID, cardID).Delete(&model.CardLabel{}).Error; err != nil {
		return err
	}
	if len(labelIDs) == 0 {
		return nil
	}
	rows := make([]*model.CardLabel, 0, len(labelIDs))
	for _, labelID := range labelIDs {
		rows = append(rows, &model.CardLabel{UserID: userID, CardID: cardID, LabelID: labelID})
	}
	return conn(ctx).Create(&rows).Error
}

// CardIDsWithLabel 返回带有该标签的卡片 ID。
func (*cardRepository) CardIDsWithLabel(ctx context.Context, userID, labelID int64) ([]int64, error) {
	ids := make([]int64, 0)
	return ids, conn(ctx).Model(&model.CardLabel{}).Where("user_id = ? AND label_id = ?", userID, labelID).Pluck("card_id", &ids).Error
}

// DeleteLabel 从全部卡片上移除该标签。
func (*cardRepository) DeleteLabel(ctx context.Context, userID, labelID int64) error {
	return conn(ctx).Where("user_id = ? AND label_id = ?", userID, labelID).Delete(&model.CardLabel{}).Error
}

// DeleteByUser 删除用户的全部卡片及卡片标签。
func (*cardRepository) DeleteByUser(ctx context.Context, userID int64) error {
	if err := conn(ctx).Where("user_id = ?", userID).Delete(&model.CardLabel{}).Error; err != nil {
		return err
	}
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Card{}).Error
}
