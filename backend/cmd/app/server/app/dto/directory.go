package dto

// 目录中的位置：ParentID / FolderID 为 null 表示根；Index 是在同级中的位置（从 0 开始），为 null 时排在末尾。

// CreateFolder 是新建文件夹的请求。
type CreateFolder struct {
	ParentID *int64 `json:"parent_id"`
	Name     string `json:"name"`
	Index    *int   `json:"index"`
}

// MoveFolder 是移动文件夹的请求。
type MoveFolder struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id"`
	Index    *int   `json:"index"`
}

// CreateBoard 是新建看板的请求。
type CreateBoard struct {
	FolderID *int64 `json:"folder_id"`
	Name     string `json:"name"`
	Index    *int   `json:"index"`
}

// PlaceBoard 是移动看板，或把看板从看板归档恢复到目录的请求。
type PlaceBoard struct {
	ID       int64  `json:"id"`
	FolderID *int64 `json:"folder_id"`
	Index    *int   `json:"index"`
}

// Rename 是修改文件夹或看板名称的请求。
type Rename struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// SetMainBoard 是设置主看板的请求，ID 为 null 时取消主看板。
type SetMainBoard struct {
	ID *int64 `json:"id"`
}
