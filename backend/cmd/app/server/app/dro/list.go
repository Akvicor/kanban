package dro

import (
	"kanban/cmd/app/server/common/listrule"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/model"
	"time"
)

// List 是面板中的列表，连同它的操作配置。ArchivedAt 不为空表示在列表归档中。
type List struct {
	ID         int64              `json:"id"`
	PanelID    int64              `json:"panel_id"`
	Name       string             `json:"name"`
	Color      string             `json:"color"`
	Position   int64              `json:"position"`
	ShowAge    bool               `json:"show_age"`
	SortMode   listsort.Mode      `json:"sort_mode"`
	SortDir    listsort.Direction `json:"sort_dir"`
	HeadAdd    listsort.End       `json:"head_add"`
	TailAdd    listsort.End       `json:"tail_add"`
	RemindOff  bool               `json:"remind_off"`
	DueOff     bool               `json:"due_off"`
	Rules      listrule.Rules     `json:"rules"`
	ArchivedAt *time.Time         `json:"archived_at"`
}

// NewList 从列表模型和它的操作配置生成列表信息。
func NewList(list *model.List, rules listrule.Rules) List {
	return List{
		ID:         list.ID,
		PanelID:    list.PanelID,
		Name:       list.Name,
		Color:      list.Color,
		Position:   list.Position,
		ShowAge:    list.ShowAge,
		SortMode:   list.SortMode,
		SortDir:    list.SortDir,
		HeadAdd:    list.HeadAdd,
		TailAdd:    list.TailAdd,
		RemindOff:  list.RemindOff,
		DueOff:     list.DueOff,
		Rules:      rules,
		ArchivedAt: list.ArchivedAt,
	}
}

// PanelContent 是打开面板时一次加载的内容：面板的全部列表（包括列表归档中的）及其操作配置，
// 以及不在列表归档中的列表里的卡片（不含卡片归档）连同任务、操作记录、关联和附件。
// CardBundle 中的 Revision 是读取时该用户的同步序号，客户端把这些数据视为该序号时的状态，之后按推送更新。
type PanelContent struct {
	PanelID int64  `json:"panel_id"`
	Lists   []List `json:"lists"`
	CardBundle
}
