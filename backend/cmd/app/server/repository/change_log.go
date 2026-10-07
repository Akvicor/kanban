package repository

import (
	"context"
	"kanban/cmd/app/server/model"
)

// ChangeLog 是变更记录表的仓储。
var ChangeLog = new(changeLogRepository)

type changeLogRepository struct{}

// Create 追加一条变更记录。
func (*changeLogRepository) Create(ctx context.Context, log *model.ChangeLog) error {
	return conn(ctx).Create(log).Error
}

// ListAfter 按序号升序返回用户在 after 之后的全部变更记录。
func (*changeLogRepository) ListAfter(ctx context.Context, userID, after int64) ([]*model.ChangeLog, error) {
	logs := make([]*model.ChangeLog, 0)
	return logs, conn(ctx).Where("user_id = ? AND revision > ?", userID, after).Order("revision ASC").Find(&logs).Error
}

// DeleteUpTo 删除用户序号不大于 revision 的变更记录。
func (*changeLogRepository) DeleteUpTo(ctx context.Context, userID, revision int64) error {
	return conn(ctx).Where("user_id = ? AND revision <= ?", userID, revision).Delete(&model.ChangeLog{}).Error
}

// DeleteByUser 删除用户的全部变更记录。
func (*changeLogRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.ChangeLog{}).Error
}
