package model

import "time"

// Panel 是看板中的一块独立工作区，以标签页显示。列表、卡片、标签、优先级挡位、列表归档和卡片归档都属于面板。
// ArchivedAt 不为空表示在面板归档中，此时 BoardID 是归档前所在的看板。
// 面板本身未归档、但所在看板已归档时，面板随看板算作已归档。
type Panel struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	UserID     int64      `gorm:"column:user_id;not null;index"`
	BoardID    int64      `gorm:"column:board_id;not null;index"`
	Name       string     `gorm:"column:name;size:400;not null"`
	Position   int64      `gorm:"column:position;not null"` // 在看板标签页中的顺序
	ArchivedAt *time.Time `gorm:"column:archived_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*Panel) TableName() string {
	return "panels"
}

// Archived 判断面板是否在面板归档中。
func (p *Panel) Archived() bool {
	return p.ArchivedAt != nil
}

// Label 是面板中的标签，卡片和列表操作配置只能使用所属面板的标签。
// Position 决定标签在面板中的顺序，快捷键「切换第 n 个标签」按这个顺序计数。
type Label struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	PanelID  int64  `gorm:"column:panel_id;not null;index"`
	Name     string `gorm:"column:name;size:200;not null"`
	Color    string `gorm:"column:color;size:7;not null"` // #RRGGBB
	Position int64  `gorm:"column:position;not null"`
}

// TableName 返回表名。
func (*Label) TableName() string {
	return "labels"
}

// PriorityLevel 是面板中的优先级挡位。Position 越小优先级越高；卡片没有挡位时视为最低优先级。
type PriorityLevel struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	PanelID  int64  `gorm:"column:panel_id;not null;index"`
	Name     string `gorm:"column:name;size:80;not null"`
	Color    string `gorm:"column:color;size:7;not null"` // #RRGGBB
	Position int64  `gorm:"column:position;not null"`
}

// TableName 返回表名。
func (*PriorityLevel) TableName() string {
	return "priority_levels"
}
