package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// Directory 是看板目录的接口：文件夹和看板的新建、改名、移动、归档，看板的恢复和主看板。
// 目录和看板归档的数据通过同步获得；写入成功后返回受影响的文件夹或看板。
var Directory = new(directoryApi)

type directoryApi struct{}

func (a *directoryApi) CreateFolder(c echo.Context) error {
	input := new(dto.CreateFolder)
	if !bind(c, input) {
		return nil
	}
	folder, err := service.Folder.Create(c.Request().Context(), mw.Session(c).User.ID, input.ParentID, input.Name, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewFolder(folder))
}

func (a *directoryApi) RenameFolder(c echo.Context) error {
	input := new(dto.Rename)
	if !bind(c, input) {
		return nil
	}
	folder, err := service.Folder.Rename(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewFolder(folder))
}

func (a *directoryApi) MoveFolder(c echo.Context) error {
	input := new(dto.MoveFolder)
	if !bind(c, input) {
		return nil
	}
	folder, err := service.Folder.Move(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.ParentID, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewFolder(folder))
}

func (a *directoryApi) ArchiveFolder(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Folder.Archive(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

func (a *directoryApi) CreateBoard(c echo.Context) error {
	input := new(dto.CreateBoard)
	if !bind(c, input) {
		return nil
	}
	board, err := service.Board.Create(c.Request().Context(), mw.Session(c).User.ID, input.FolderID, input.Name, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewBoard(board))
}

func (a *directoryApi) RenameBoard(c echo.Context) error {
	input := new(dto.Rename)
	if !bind(c, input) {
		return nil
	}
	board, err := service.Board.Rename(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewBoard(board))
}

func (a *directoryApi) MoveBoard(c echo.Context) error {
	input := new(dto.PlaceBoard)
	if !bind(c, input) {
		return nil
	}
	board, err := service.Board.Move(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.FolderID, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewBoard(board))
}

func (a *directoryApi) ArchiveBoard(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Board.Archive(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

func (a *directoryApi) RestoreBoard(c echo.Context) error {
	input := new(dto.PlaceBoard)
	if !bind(c, input) {
		return nil
	}
	board, err := service.Board.Restore(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.FolderID, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewBoard(board))
}

// SetMainBoard 设置或取消主看板，返回修改后的个人设置。
func (a *directoryApi) SetMainBoard(c echo.Context) error {
	input := new(dto.SetMainBoard)
	if !bind(c, input) {
		return nil
	}
	user, err := service.Board.SetMain(c.Request().Context(), mw.Session(c).User.ID, input.ID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewSettings(user))
}
