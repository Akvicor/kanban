package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// CardLink 是卡片关联的仓储。查询都带用户条件，只能取到该用户自己的关联。
var CardLink = new(cardLinkRepository)

type cardLinkRepository struct{}

// Create 新建关联。
func (*cardLinkRepository) Create(ctx context.Context, link *model.CardLink) error {
	return conn(ctx).Create(link).Error
}

// FindByID 按 ID 查找用户的关联。
func (*cardLinkRepository) FindByID(ctx context.Context, userID, id int64) (*model.CardLink, error) {
	link := new(model.CardLink)
	return link, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(link).Error
}

// ListByCards 返回这些卡片的关联，按添加顺序排列。
func (*cardLinkRepository) ListByCards(ctx context.Context, userID int64, cardIDs []int64) ([]*model.CardLink, error) {
	links := make([]*model.CardLink, 0)
	if len(cardIDs) == 0 {
		return links, nil
	}
	return links, conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("position ASC").Order("id ASC").Find(&links).Error
}

// DeleteByUser 删除用户的全部关联。
func (*cardLinkRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.CardLink{}).Error
}

// Delete 删除用户的关联。
func (*cardLinkRepository) Delete(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.CardLink{}))
}
