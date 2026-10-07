package model

import (
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/types/listsort"
	"time"
)

// List 是面板中的列表，代表一个类别。列表只在所属面板内调整顺序。
// ArchivedAt 不为空表示在所属面板的列表归档中，其中的卡片随列表一起保留。
type List struct {
	ID       int64  `gorm:"column:id;primaryKey"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	PanelID  int64  `gorm:"column:panel_id;not null;index"`
	Name     string `gorm:"column:name;size:400;not null"`
	Color    string `gorm:"column:color;size:7;not null"` // #RRGGBB，为空表示没有颜色
	Position int64  `gorm:"column:position;not null"`     // 在面板中从左到右的顺序

	ShowAge   bool               `gorm:"column:show_age;not null"`          // 当前位于该列的卡片是否显示卡龄
	SortMode  listsort.Mode      `gorm:"column:sort_mode;size:16;not null"` // 卡片排序方式
	SortDir   listsort.Direction `gorm:"column:sort_dir;size:8;not null"`   // 排序方向，手动排序不使用
	HeadAdd   listsort.End       `gorm:"column:head_add;size:8;not null"`   // 列首创建按钮把卡片插到哪一端
	TailAdd   listsort.End       `gorm:"column:tail_add;size:8;not null"`   // 列尾创建按钮把卡片插到哪一端
	RemindOff bool               `gorm:"column:remind_off;not null"`        // 关闭当前位于该列的卡片的提醒通知
	DueOff    bool               `gorm:"column:due_off;not null"`           // 关闭当前位于该列的卡片的截止通知

	ArchivedAt *time.Time `gorm:"column:archived_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*List) TableName() string {
	return "lists"
}

// Archived 判断列表是否在列表归档中。
func (l *List) Archived() bool {
	return l.ArchivedAt != nil
}

// 标签规则的动作。
const (
	LabelAdd    = "add"
	LabelRemove = "remove"
)

// ListLabelRule 是列表某种操作对一个标签的调整：增加或删除。标签必须属于列表所在的面板。
// 同一列表、同一操作、同一动作、同一标签只保留一条。
type ListLabelRule struct {
	ID      int64       `gorm:"column:id;primaryKey"`
	UserID  int64       `gorm:"column:user_id;not null;index"`
	ListID  int64       `gorm:"column:list_id;not null;uniqueIndex:idx_list_label_rule"`
	Op      listrule.Op `gorm:"column:op;size:8;not null;uniqueIndex:idx_list_label_rule"`
	Action  string      `gorm:"column:action;size:8;not null;uniqueIndex:idx_list_label_rule"`
	LabelID int64       `gorm:"column:label_id;not null;uniqueIndex:idx_list_label_rule;index"`
}

// TableName 返回表名。
func (*ListLabelRule) TableName() string {
	return "list_label_rules"
}

// 时间规则作用的字段。
const (
	FieldStart    = "start"
	FieldComplete = "complete"
)

// ListTimeRule 是列表某种操作对开始时间或完成时间的动作。没有记录表示不影响。
// 同一列表、同一操作、同一字段只保留一条。
type ListTimeRule struct {
	ID     int64               `gorm:"column:id;primaryKey"`
	UserID int64               `gorm:"column:user_id;not null;index"`
	ListID int64               `gorm:"column:list_id;not null;uniqueIndex:idx_list_time_rule"`
	Op     listrule.Op         `gorm:"column:op;size:8;not null;uniqueIndex:idx_list_time_rule"`
	Field  string              `gorm:"column:field;size:8;not null;uniqueIndex:idx_list_time_rule"`
	Action listrule.TimeAction `gorm:"column:action;size:8;not null"`
}

// TableName 返回表名。
func (*ListTimeRule) TableName() string {
	return "list_time_rules"
}
