package dro

import (
	"kanban/cmd/app/server/model"
	"time"
)

// Folder 是目录中的文件夹。ArchivedAt 不为空表示在看板归档中。
type Folder struct {
	ID         int64      `json:"id"`
	ParentID   *int64     `json:"parent_id"`
	Name       string     `json:"name"`
	Position   int64      `json:"position"`
	ArchivedAt *time.Time `json:"archived_at"`
}

// Board 是目录中的看板。ArchivedAt 不为空表示在看板归档中，此时 FolderID 是归档前所在的文件夹。
type Board struct {
	ID          int64      `json:"id"`
	FolderID    *int64     `json:"folder_id"`
	Name        string     `json:"name"`
	Position    int64      `json:"position"`
	MainPanelID *int64     `json:"main_panel_id"`
	ArchivedAt  *time.Time `json:"archived_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NewFolder 从文件夹模型生成文件夹信息。
func NewFolder(folder *model.Folder) Folder {
	return Folder{
		ID:         folder.ID,
		ParentID:   folder.ParentID,
		Name:       folder.Name,
		Position:   folder.Position,
		ArchivedAt: folder.ArchivedAt,
	}
}

// NewFolders 生成文件夹列表。
func NewFolders(folders []*model.Folder) []Folder {
	list := make([]Folder, 0, len(folders))
	for _, folder := range folders {
		list = append(list, NewFolder(folder))
	}
	return list
}

// NewBoard 从看板模型生成看板信息。
func NewBoard(board *model.Board) Board {
	return Board{
		ID:          board.ID,
		FolderID:    board.FolderID,
		Name:        board.Name,
		Position:    board.Position,
		MainPanelID: board.MainPanelID,
		ArchivedAt:  board.ArchivedAt,
		CreatedAt:   board.CreatedAt,
	}
}

// NewBoards 生成看板列表。
func NewBoards(boards []*model.Board) []Board {
	list := make([]Board, 0, len(boards))
	for _, board := range boards {
		list = append(list, NewBoard(board))
	}
	return list
}
