package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// CardAction 是卡片操作记录的仓储。查询都带用户条件，只能取到该用户自己的记录。
var CardAction = new(cardActionRepository)

type cardActionRepository struct{}

// Create 追加一条操作记录。
func (*cardActionRepository) Create(ctx context.Context, action *model.CardAction) error {
	return conn(ctx).Create(action).Error
}

// DeleteByUser 删除用户的全部操作记录。
func (*cardActionRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.CardAction{}).Error
}

// ListByCards 返回这些卡片的操作记录，按时间排列。
func (*cardActionRepository) ListByCards(ctx context.Context, userID int64, cardIDs []int64) ([]*model.CardAction, error) {
	actions := make([]*model.CardAction, 0)
	if len(cardIDs) == 0 {
		return actions, nil
	}
	return actions, conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("created_at ASC").Order("id ASC").Find(&actions).Error
}
