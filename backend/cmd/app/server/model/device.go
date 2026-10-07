package model

import "time"

// Device 是一台已登录的设备。每台设备持有自己的令牌，库中只保存令牌哈希。
// 删除记录即吊销令牌：登出、踢掉设备、修改或重置密码、停用或删除用户都通过删除记录完成。
type Device struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	UserID       int64     `gorm:"column:user_id;not null;index"`
	Name         string    `gorm:"column:name;size:128;not null"`
	TokenHash    string    `gorm:"column:token_hash;size:64;not null;uniqueIndex"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	LastActiveAt time.Time `gorm:"column:last_active_at;not null"`
}

// TableName 返回表名。
func (*Device) TableName() string {
	return "devices"
}
