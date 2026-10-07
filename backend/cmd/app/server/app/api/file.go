package api

import (
	"errors"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/global/storage"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/service"
	"mime"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Akvicor/glog"
	"github.com/labstack/echo/v4"
)

// FileDownload 是读取文件内容和缩略图的接口，挂在 /api/file 下，经过 mw.FileAuth。
// 只读，结果直接以 HTTP 状态码表示，供页面中的图片、音视频、PDF 和下载链接使用。
var FileDownload = new(fileDownloadApi)

type fileDownloadApi struct{}

// inline 判断文件能否在页面中直接显示：图片（SVG 除外，它能执行脚本）、PDF、音视频。
// 其他类型一律作为下载，避免浏览器把用户上传的 HTML 等内容当作本站页面执行。
func inline(blob *model.Blob) bool {
	switch {
	case blob.MimeType == "image/svg+xml":
		return false
	case strings.HasPrefix(blob.MimeType, "image/"), strings.HasPrefix(blob.MimeType, "audio/"),
		strings.HasPrefix(blob.MimeType, "video/"), blob.MimeType == "application/pdf":
		return true
	}
	return false
}

// failStatus 把服务错误转换为 HTTP 状态码。
func failStatus(c echo.Context, err error) error {
	var businessError *service.Error
	if errors.As(err, &businessError) {
		switch businessError.Kind {
		case service.KindNotFound:
			return c.NoContent(http.StatusNotFound)
		case service.KindBadRequest:
			return c.NoContent(http.StatusBadRequest)
		}
	}
	glog.Error("%s %s: %v", c.Request().Method, c.Path(), err)
	return c.NoContent(http.StatusInternalServerError)
}

func pathID(c echo.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	return id, err == nil
}

// serve 以 Range 和条件请求的方式返回存储中的文件。name 不为空时作为下载文件名（保留大小写，按 RFC 5987 编码）。
func serve(c echo.Context, path, contentType, disposition, name, etag string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		glog.Error("存储中缺少文件: %s", path)
		return c.NoContent(http.StatusNotFound)
	}
	if err != nil {
		return failStatus(c, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return failStatus(c, err)
	}
	header := c.Response().Header()
	header.Set(echo.HeaderContentType, contentType)
	header.Set("X-Content-Type-Options", "nosniff")
	// 只允许浏览器私有缓存，并且每次使用前向服务端验证（内容未变时返回 304，不重新传输）。
	// 这样登出或令牌被吊销后，缓存中的文件不能再直接打开。
	header.Set("Cache-Control", "private, no-cache")
	header.Set("ETag", `"`+etag+`"`)
	params := map[string]string{}
	if name != "" {
		params["filename"] = name
	}
	header.Set(echo.HeaderContentDisposition, mime.FormatMediaType(disposition, params))
	http.ServeContent(c.Response(), c.Request(), "", info.ModTime(), file)
	return nil
}

func serveBlob(c echo.Context, blob *model.Blob, name string) error {
	disposition := "attachment"
	if inline(blob) {
		disposition = "inline"
	}
	return serve(c, storage.Get().BlobPath(blob.SHA256), blob.MimeType, disposition, name, blob.SHA256)
}

func serveThumbnail(c echo.Context, blob *model.Blob) error {
	size, err := strconv.Atoi(c.Param("size"))
	if err != nil || !slices.Contains([]int{360, 720}, size) || !blob.IsImage() {
		return c.NoContent(http.StatusNotFound)
	}
	contentType := mime.TypeByExtension("." + blob.ThumbnailExt)
	path := storage.Get().ThumbnailPath(blob.SHA256, size, blob.ThumbnailExt)
	return serve(c, path, contentType, "inline", "", blob.SHA256+"-"+strconv.Itoa(size))
}

// Attachment 返回文件附件的内容，文件名为附件名称。
func (a *fileDownloadApi) Attachment(c echo.Context) error {
	id, ok := pathID(c)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	attachment, blob, err := service.Attachment.Open(c.Request().Context(), mw.Session(c).User.ID, id)
	if err != nil {
		return failStatus(c, err)
	}
	return serveBlob(c, blob, attachment.Name)
}

// AttachmentThumbnail 返回文件附件的缩略图，规格为 360 或 720。
func (a *fileDownloadApi) AttachmentThumbnail(c echo.Context) error {
	id, ok := pathID(c)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	_, blob, err := service.Attachment.Open(c.Request().Context(), mw.Session(c).User.ID, id)
	if err != nil {
		return failStatus(c, err)
	}
	return serveThumbnail(c, blob)
}

// UserFile 返回文件管理中某个文件的内容，文件名为最近一次引用它的附件名称。
func (a *fileDownloadApi) UserFile(c echo.Context) error {
	id, ok := pathID(c)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	file, blob, err := service.File.Open(c.Request().Context(), mw.Session(c).User.ID, id)
	if err != nil {
		return failStatus(c, err)
	}
	return serveBlob(c, blob, file.Name)
}

// UserFileThumbnail 返回文件管理中某个文件的缩略图。
func (a *fileDownloadApi) UserFileThumbnail(c echo.Context) error {
	id, ok := pathID(c)
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}
	_, blob, err := service.File.Open(c.Request().Context(), mw.Session(c).User.ID, id)
	if err != nil {
		return failStatus(c, err)
	}
	return serveThumbnail(c, blob)
}
