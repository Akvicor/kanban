package api

import (
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/service"
	"strconv"

	"github.com/labstack/echo/v4"
)

// Attachment 是上传、附件和文件管理的接口。写入成功后返回受影响的附件或卡片。
var Attachment = new(attachmentApi)

type attachmentApi struct{}

func (a *attachmentApi) respond(c echo.Context, attachment *model.Attachment, err error) error {
	if err != nil {
		return fail(c, err)
	}
	view, err := service.Attachment.View(c.Request().Context(), attachment)
	if err != nil {
		return fail(c, err)
	}
	return success(c, view)
}

// PrepareUpload 开始或继续上传一份文件，返回是否已存在，或上传会话和已收到的字节数。
func (a *attachmentApi) PrepareUpload(c echo.Context) error {
	input := new(dto.PrepareUpload)
	if !bind(c, input) {
		return nil
	}
	state, err := service.Upload.Prepare(c.Request().Context(), mw.Session(c).User.ID, input.SHA256, input.Size)
	if err != nil {
		return fail(c, err)
	}
	return success(c, state)
}

// UploadChunk 追加一个分片。会话和偏移在查询参数中，请求体是分片的原始字节。
func (a *attachmentApi) UploadChunk(c echo.Context) error {
	sessionID, ok := queryID(c, "session_id")
	if !ok {
		return nil
	}
	offset, err := strconv.ParseInt(c.QueryParam("offset"), 10, 64)
	if err != nil {
		return resp.Fail(c, resp.MalformedRequest)
	}
	state, err := service.Upload.Chunk(c.Request().Context(), mw.Session(c).User.ID, sessionID, offset, c.Request().Body)
	if err != nil {
		return fail(c, err)
	}
	return success(c, state)
}

// CreateFile 用已上传的文件新建文件附件。
func (a *attachmentApi) CreateFile(c echo.Context) error {
	input := new(dto.CreateFileAttachment)
	if !bind(c, input) {
		return nil
	}
	attachment, err := service.Attachment.CreateFile(c.Request().Context(), mw.Session(c).User.ID, input.CardID, input.SHA256, input.Name)
	return a.respond(c, attachment, err)
}

// CreateLink 新建链接附件。
func (a *attachmentApi) CreateLink(c echo.Context) error {
	input := new(dto.CreateLinkAttachment)
	if !bind(c, input) {
		return nil
	}
	attachment, err := service.Attachment.CreateLink(c.Request().Context(), mw.Session(c).User.ID, input.CardID, input.URL, input.Name)
	return a.respond(c, attachment, err)
}

func (a *attachmentApi) Rename(c echo.Context) error {
	input := new(dto.RenameAttachment)
	if !bind(c, input) {
		return nil
	}
	attachment, err := service.Attachment.Rename(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name)
	return a.respond(c, attachment, err)
}

func (a *attachmentApi) Delete(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Attachment.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// SetCover 设置或取消卡片封面，返回卡片。
func (a *attachmentApi) SetCover(c echo.Context) error {
	input := new(dto.SetCardCover)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Attachment.SetCover(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.AttachmentID)
	return Card.respondCard(c, card, err)
}

// ListFiles 返回用户的全部文件，用于文件管理。
func (a *attachmentApi) ListFiles(c echo.Context) error {
	files, err := service.File.List(c.Request().Context(), mw.Session(c).User.ID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, files)
}

// DeleteFile 删除引用数为 0 的用户文件。
func (a *attachmentApi) DeleteFile(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.File.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}
