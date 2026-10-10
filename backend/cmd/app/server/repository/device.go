package repository

import (
	"context"
	"kanban/cmd/app/server/model"
	"time"
)

// Device 是设备表的仓储。
var Device = new(deviceRepository)

type deviceRepository struct{}

// Create 新建设备记录。
func (*deviceRepository) Create(ctx context.Context, device *model.Device) error {
	return conn(ctx).Create(device).Error
}

// FindByTokenHash 按令牌哈希查找设备。
func (*deviceRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.Device, error) {
	device := new(model.Device)
	return device, conn(ctx).Where("token_hash = ?", tokenHash).Take(device).Error
}

// FindByFileTokenHash 按文件令牌哈希查找设备。
func (*deviceRepository) FindByFileTokenHash(ctx context.Context, fileTokenHash string) (*model.Device, error) {
	device := new(model.Device)
	return device, conn(ctx).Where("file_token_hash = ?", fileTokenHash).Take(device).Error
}

// SetFileTokenHash 设置设备的文件令牌哈希，nil 表示注销。设备不存在时返回 gorm.ErrRecordNotFound。
func (*deviceRepository) SetFileTokenHash(ctx context.Context, id int64, fileTokenHash *string) error {
	return affected(conn(ctx).Model(&model.Device{}).Where("id = ?", id).Update("file_token_hash", fileTokenHash))
}

// ListByUser 按最后活跃时间倒序返回用户的全部设备。
func (*deviceRepository) ListByUser(ctx context.Context, userID int64) ([]*model.Device, error) {
	devices := make([]*model.Device, 0)
	return devices, conn(ctx).Where("user_id = ?", userID).Order("last_active_at DESC").Order("id DESC").Find(&devices).Error
}

// ListIdleBefore 返回所有用户中最后活跃时间早于 cutoff 的设备，供后台清理闲置设备使用。
func (*deviceRepository) ListIdleBefore(ctx context.Context, cutoff time.Time) ([]*model.Device, error) {
	devices := make([]*model.Device, 0)
	return devices, conn(ctx).Where("last_active_at < ?", cutoff).Order("id").Find(&devices).Error
}

// Touch 更新设备的最后活跃时间。
func (*deviceRepository) Touch(ctx context.Context, id int64, at time.Time) error {
	return conn(ctx).Model(&model.Device{}).Where("id = ?", id).Update("last_active_at", at).Error
}

// Delete 删除用户自己的某台设备，设备不属于该用户时返回 gorm.ErrRecordNotFound。
func (*deviceRepository) Delete(ctx context.Context, userID, id int64) error {
	return affected(conn(ctx).Where("user_id = ? AND id = ?", userID, id).Delete(&model.Device{}))
}

// DeleteByUser 删除用户的全部设备。
func (*deviceRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return conn(ctx).Where("user_id = ?", userID).Delete(&model.Device{}).Error
}
