package model

import "time"

// Folder 是看板目录中的文件夹，可以包含文件夹和看板。
// ParentID 为空表示位于根。同一父级下的文件夹和看板共用 Position 排序。
// ArchivedAt 不为空表示随看板归档进入了看板归档，此后只作为归档中的层级结构，不能单独恢复。
type Folder struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	UserID     int64      `gorm:"column:user_id;not null;index"`
	ParentID   *int64     `gorm:"column:parent_id"`
	Name       string     `gorm:"column:name;size:400;not null"`
	Position   int64      `gorm:"column:position;not null"`
	ArchivedAt *time.Time `gorm:"column:archived_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*Folder) TableName() string {
	return "folders"
}

// Archived 判断文件夹是否在看板归档中。
func (f *Folder) Archived() bool {
	return f.ArchivedAt != nil
}
