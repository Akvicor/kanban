package model

import "time"

// Board 是看板目录中的看板，是面板的容器。
// FolderID 为空表示位于根。同一父级下的文件夹和看板共用 Position 排序。
// ArchivedAt 不为空表示在看板归档中；归档时 FolderID 保持归档前的位置，用于在归档中按原层级展示。
type Board struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	FolderID *int64 `gorm:"column:folder_id"`
	Name     string `gorm:"column:name;size:400;not null"`
	Position int64  `gorm:"column:position;not null"`
	// MainPanelID 是打开看板时进入的面板，必须是属于该看板且未归档的面板；为空时进入第一个面板。
	// 主面板进入面板归档或被移到其他看板时清空。
	MainPanelID *int64     `gorm:"column:main_panel_id"`
	ArchivedAt  *time.Time `gorm:"column:archived_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*Board) TableName() string {
	return "boards"
}

// Archived 判断看板是否在看板归档中。
func (b *Board) Archived() bool {
	return b.ArchivedAt != nil
}
