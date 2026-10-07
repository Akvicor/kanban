package dro

import (
	"kanban/cmd/app/server/model"
	"time"
)

// Panel 是看板中的面板。ArchivedAt 不为空表示在面板归档中，此时 BoardID 是归档前所在的看板。
type Panel struct {
	ID         int64      `json:"id"`
	BoardID    int64      `json:"board_id"`
	Name       string     `json:"name"`
	Position   int64      `json:"position"`
	ArchivedAt *time.Time `json:"archived_at"`
}

// Label 是面板中的标签。
type Label struct {
	ID       int64  `json:"id"`
	PanelID  int64  `json:"panel_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Position int64  `json:"position"`
}

// PriorityLevel 是面板中的优先级挡位，Position 越小优先级越高。
type PriorityLevel struct {
	ID       int64  `json:"id"`
	PanelID  int64  `json:"panel_id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Position int64  `json:"position"`
}

// NewPanel 从面板模型生成面板信息。
func NewPanel(panel *model.Panel) Panel {
	return Panel{ID: panel.ID, BoardID: panel.BoardID, Name: panel.Name, Position: panel.Position, ArchivedAt: panel.ArchivedAt}
}

// NewLabel 从标签模型生成标签信息。
func NewLabel(label *model.Label) Label {
	return Label{ID: label.ID, PanelID: label.PanelID, Name: label.Name, Color: label.Color, Position: label.Position}
}

// NewPriorityLevel 从挡位模型生成挡位信息。
func NewPriorityLevel(level *model.PriorityLevel) PriorityLevel {
	return PriorityLevel{ID: level.ID, PanelID: level.PanelID, Name: level.Name, Color: level.Color, Position: level.Position}
}

// mapList 把模型列表逐项转换为返回结构。
func mapList[M any, D any](items []*M, convert func(*M) D) []D {
	list := make([]D, 0, len(items))
	for _, item := range items {
		list = append(list, convert(item))
	}
	return list
}

// NewPanels 生成面板列表。
func NewPanels(panels []*model.Panel) []Panel { return mapList(panels, NewPanel) }

// NewLabels 生成标签列表。
func NewLabels(labels []*model.Label) []Label { return mapList(labels, NewLabel) }

// NewPriorityLevels 生成挡位列表。
func NewPriorityLevels(levels []*model.PriorityLevel) []PriorityLevel {
	return mapList(levels, NewPriorityLevel)
}
