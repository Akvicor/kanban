package dto

import (
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/types/listsort"
)

// CreateList 是在面板最右边新建列表的请求。
type CreateList struct {
	PanelID int64  `json:"panel_id"`
	Name    string `json:"name"`
}

// ListSettings 是修改列表展示和通知设置的请求，各项都需要提供。Color 为空字符串表示没有颜色。
type ListSettings struct {
	ID        int64              `json:"id"`
	Color     string             `json:"color"`
	ShowAge   bool               `json:"show_age"`
	SortMode  listsort.Mode      `json:"sort_mode"`
	SortDir   listsort.Direction `json:"sort_dir"`
	HeadAdd   listsort.End       `json:"head_add"`
	TailAdd   listsort.End       `json:"tail_add"`
	RemindOff bool               `json:"remind_off"`
	DueOff    bool               `json:"due_off"`
}

// ListRules 是替换列表操作配置的请求。
type ListRules struct {
	ID    int64          `json:"id"`
	Rules listrule.Rules `json:"rules"`
}

// RestoreList 是从列表归档恢复列表的请求，AtStart 为 true 时放在面板最左边，否则放在最右边。
type RestoreList struct {
	ID      int64 `json:"id"`
	AtStart bool  `json:"at_start"`
}
