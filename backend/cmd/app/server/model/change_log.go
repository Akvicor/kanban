package model

import "time"

// ChangeLog 是某个用户的一条变更记录，按 (user_id, revision) 唯一。
// 每次写入数据库的操作在同一事务中追加记录，提交后推送给该用户在线的设备；
// 设备断线重连时，按记录补齐断线期间的变更。只保留最近一段，更早的记录被删除。
type ChangeLog struct {
	UserID    int64     `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	Revision  int64     `gorm:"column:revision;primaryKey;autoIncrement:false"`
	Op        string    `gorm:"column:op;size:16;not null"`   // upsert 或 delete
	Type      string    `gorm:"column:type;size:32;not null"` // 实体类型，例如 settings、device
	EntityID  int64     `gorm:"column:entity_id;not null"`
	Data      string    `gorm:"column:data;type:text;not null"` // 实体变更后的完整数据（JSON），删除时为 null
	CreatedAt time.Time `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*ChangeLog) TableName() string {
	return "change_logs"
}
