package repository

import (
	"context"
	"kanban/cmd/app/server/model"
	"time"
)

// UploadSession 是上传会话的仓储。按用户查询的函数都带用户条件；过期清理按时间查询全部用户。
var UploadSession = new(uploadSessionRepository)

type uploadSessionRepository struct{}

// Create 新建上传会话。
func (*uploadSessionRepository) Create(ctx context.Context, session *model.UploadSession) error {
	return conn(ctx).Create(session).Error
}

// FindByID 按 ID 查找用户的上传会话。
func (*uploadSessionRepository) FindByID(ctx context.Context, userID, id int64) (*model.UploadSession, error) {
	session := new(model.UploadSession)
	return session, conn(ctx).Where("user_id = ? AND id = ?", userID, id).Take(session).Error
}

// FindResumable 查找用户上传同一 sha256 和大小的未完成会话，用于断点续传。
func (*uploadSessionRepository) FindResumable(ctx context.Context, userID int64, sha string, size int64) (*model.UploadSession, error) {
	session := new(model.UploadSession)
	return session, conn(ctx).Where("user_id = ? AND sha256 = ? AND size = ? AND completed = ?", userID, sha, size, false).Order("id DESC").Take(session).Error
}

// HasCompleted 判断用户是否有该 sha256 已完成的上传会话，即持有这份内容的凭证。
func (*uploadSessionRepository) HasCompleted(ctx context.Context, userID int64, sha string) (bool, error) {
	var count int64
	err := conn(ctx).Model(&model.UploadSession{}).Where("user_id = ? AND sha256 = ? AND completed = ?", userID, sha, true).Count(&count).Error
	return count > 0, err
}

// DeleteCompleted 删除用户该 sha256 已完成的上传会话。
func (*uploadSessionRepository) DeleteCompleted(ctx context.Context, userID int64, sha string) error {
	return conn(ctx).Where("user_id = ? AND sha256 = ? AND completed = ?", userID, sha, true).Delete(&model.UploadSession{}).Error
}

// Update 更新会话的字段。
func (*uploadSessionRepository) Update(ctx context.Context, userID, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.UploadSession{}).Where("user_id = ? AND id = ?", userID, id).Updates(values))
}

// Delete 删除用户的上传会话。
func (*uploadSessionRepository) Delete(ctx context.Context, userID, id int64) error {
	return conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.UploadSession{}).Error
}

// ListIDsByUser 返回用户全部上传会话的 ID。
func (*uploadSessionRepository) ListIDsByUser(ctx context.Context, userID int64) ([]int64, error) {
	var ids []int64
	return ids, conn(ctx).Model(&model.UploadSession{}).Where("user_id = ?", userID).Pluck("id", &ids).Error
}

// DeleteByUser 删除用户的全部上传会话。
func (*uploadSessionRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.UploadSession{}).Error
}

// ListExpired 返回最后活动时间早于 before 的会话（全部用户）。
func (*uploadSessionRepository) ListExpired(ctx context.Context, before time.Time) ([]*model.UploadSession, error) {
	sessions := make([]*model.UploadSession, 0)
	return sessions, conn(ctx).Where("updated_at < ?", before).Find(&sessions).Error
}

// ExistingIDs 返回这些 ID 中仍有记录的会话（全部用户），用于清理孤立的上传临时文件。
func (*uploadSessionRepository) ExistingIDs(ctx context.Context, ids []int64) (map[int64]bool, error) {
	result := map[int64]bool{}
	if len(ids) == 0 {
		return result, nil
	}
	var found []int64
	if err := conn(ctx).Model(&model.UploadSession{}).Where("id IN ?", ids).Pluck("id", &found).Error; err != nil {
		return nil, err
	}
	for _, id := range found {
		result[id] = true
	}
	return result, nil
}
