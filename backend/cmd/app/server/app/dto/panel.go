package dto

// CreatePanel 是在看板中新建面板的请求，面板排在最后一个标签页。
type CreatePanel struct {
	BoardID int64  `json:"board_id"`
	Name    string `json:"name"`
}

// Reorder 是调整面板、标签或优先级挡位顺序的请求。Index 是在同级中的位置（从 0 开始），为 null 时放到最后。
type Reorder struct {
	ID    int64 `json:"id"`
	Index *int  `json:"index"`
}

// PanelToBoard 是把面板移到另一个看板，或从归档中恢复到某个看板的请求。
type PanelToBoard struct {
	ID      int64 `json:"id"`
	BoardID int64 `json:"board_id"`
}

// SetMainPanel 是设置看板主面板的请求，PanelID 为 null 时取消。
type SetMainPanel struct {
	BoardID int64  `json:"board_id"`
	PanelID *int64 `json:"panel_id"`
}

// CreateOption 是在面板中新建标签或优先级挡位的请求。
type CreateOption struct {
	PanelID int64  `json:"panel_id"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

// UpdateOption 是修改标签或优先级挡位名称和颜色的请求。
type UpdateOption struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}
