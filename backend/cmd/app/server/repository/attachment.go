package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// Attachment 是附件的仓储。查询都带用户条件。
var Attachment = new(attachmentRepository)

type attachmentRepository struct{}

// Create 新建附件。
func (*attachmentRepository) Create(ctx context.Context, attachment *model.Attachment) error {
	return conn(ctx).Create(attachment).Error
}

// FindByID 按 ID 查找用户的附件。
func (*attachmentRepository) FindByID(ctx context.Context, userID, id int64) (*model.Attachment, error) {
	attachment := new(model.Attachment)
	return attachment, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(attachment).Error
}

// ListByCards 返回这些卡片的附件，按添加顺序排列。
func (*attachmentRepository) ListByCards(ctx context.Context, userID int64, cardIDs []int64) ([]*model.Attachment, error) {
	attachments := make([]*model.Attachment, 0)
	if len(cardIDs) == 0 {
		return attachments, nil
	}
	return attachments, conn(ctx).Where("user_id = ? AND card_id IN ?", userID, cardIDs).Order("id ASC").Find(&attachments).Error
}

// Update 更新附件的字段。
func (*attachmentRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.Attachment{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// Delete 删除用户的附件。
func (*attachmentRepository) Delete(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.Attachment{}))
}

// DeleteByUser 删除用户的全部附件。
func (*attachmentRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Attachment{}).Error
}
