package app

import (
	"kanban/cmd/app/server/app/api"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/app/ws"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// setupRoutes 注册前端页面和 /api 下的接口。
// 读取用 GET，写入用 POST；需要登录的接口经过 mw.Auth，管理接口再经过 mw.Admin。
func setupRoutes(e *echo.Echo) {
	e.Use(middleware.Recover())
	e.Use(mw.SecurityHeaders)
	e.Use(mw.Error)
	e.Use(mw.Language)
	// WebSocket 升级请求不压缩：压缩包装会改写 101 响应头，导致升级失败。
	// 文件接口不压缩：Range 请求返回的是原文件的指定区间，压缩后长度和区间都会对不上。
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: func(c echo.Context) bool {
			return strings.EqualFold(c.Request().Header.Get(echo.HeaderUpgrade), "websocket") ||
				strings.HasPrefix(c.Request().URL.Path, mw.FileCookiePath+"/")
		},
	}))

	setupWebRoutes(e.Group(""), getFS())

	apiGroup := e.Group("/api", mw.APIBodyLimit())
	public := apiGroup.Group("")
	user := apiGroup.Group("", mw.Auth)
	admin := apiGroup.Group("/admin", mw.Auth, mw.Admin)
	files := e.Group(mw.FileCookiePath, mw.FileAuth)

	// 系统信息
	{
		public.GET("/sys/info/health", api.Sys.Health)
	}

	// 登录与登出
	{
		public.POST("/auth/login", api.Auth.Login, mw.LoginRateLimit())
		user.POST("/auth/logout", api.Auth.Logout)
		user.POST("/auth/file_cookie", api.Auth.FileCookie)
		user.POST("/auth/file_cookie/revoke", api.Auth.RevokeFileCookie)
	}

	// 同步：WebSocket 在握手消息中认证，因此挂在公开分组
	{
		public.GET("/sync/ws", ws.Handle)
	}

	// 当前用户：信息、个人设置、密码、设备。设备列表通过同步获得
	{
		user.GET("/user/me", api.User.Me)
		user.POST("/user/profile/update", api.User.UpdateProfile)
		user.POST("/user/settings/update", api.User.UpdateSettings)
		user.POST("/user/password/update", api.User.ChangePassword)
		user.POST("/user/device/revoke", api.User.RevokeDevice)
	}

	// 看板目录：文件夹、看板、看板归档、主看板
	{
		user.POST("/folder/create", api.Directory.CreateFolder)
		user.POST("/folder/rename", api.Directory.RenameFolder)
		user.POST("/folder/move", api.Directory.MoveFolder)
		user.POST("/folder/archive", api.Directory.ArchiveFolder)
		user.POST("/board/create", api.Directory.CreateBoard)
		user.POST("/board/rename", api.Directory.RenameBoard)
		user.POST("/board/move", api.Directory.MoveBoard)
		user.POST("/board/archive", api.Directory.ArchiveBoard)
		user.POST("/board/restore", api.Directory.RestoreBoard)
		user.POST("/board/main", api.Directory.SetMainBoard)
	}

	// 面板：标签页、移到其他看板、面板归档、主面板
	{
		user.POST("/panel/create", api.Panel.Create)
		user.POST("/panel/rename", api.Panel.Rename)
		user.POST("/panel/reorder", api.Panel.Reorder)
		user.POST("/panel/move", api.Panel.Move)
		user.POST("/panel/archive", api.Panel.Archive)
		user.POST("/panel/restore", api.Panel.Restore)
		user.POST("/board/main_panel", api.Panel.SetMain)
	}

	// 面板中的标签和优先级挡位
	{
		user.POST("/label/create", api.PanelOption.CreateLabel)
		user.POST("/label/update", api.PanelOption.UpdateLabel)
		user.POST("/label/reorder", api.PanelOption.ReorderLabel)
		user.POST("/label/delete", api.PanelOption.DeleteLabel)
		user.POST("/priority/create", api.PanelOption.CreatePriority)
		user.POST("/priority/update", api.PanelOption.UpdatePriority)
		user.POST("/priority/reorder", api.PanelOption.ReorderPriority)
		user.POST("/priority/delete", api.PanelOption.DeletePriority)
	}

	// 列表：加载面板内容、设置、操作配置、列表归档
	{
		user.GET("/panel/content", api.List.Content)
		user.POST("/list/create", api.List.Create)
		user.POST("/list/rename", api.List.Rename)
		user.POST("/list/settings", api.List.UpdateSettings)
		user.POST("/list/rules", api.List.UpdateRules)
		user.POST("/list/reorder", api.List.Reorder)
		user.POST("/list/archive", api.List.Archive)
		user.POST("/list/restore", api.List.Restore)
	}

	// 卡片：新建、修改、移动、归档、恢复、复制，以及按需加载卡片归档和列表归档中的卡片
	{
		user.GET("/panel/archived_cards", api.Card.Archived)
		user.GET("/list/cards", api.Card.InArchivedList)
		user.POST("/card/create", api.Card.Create)
		user.POST("/card/title", api.Card.UpdateTitle)
		user.POST("/card/description", api.Card.UpdateDescription)
		user.POST("/card/priority", api.Card.SetPriority)
		user.POST("/card/dates", api.Card.SetDates)
		user.POST("/card/label", api.Card.SetLabel)
		user.POST("/card/timer", api.Card.Timer)
		user.POST("/card/move", api.Card.Move)
		user.POST("/card/archive", api.Card.Archive)
		user.POST("/list/archive_cards", api.Card.ArchiveAllInList)
		user.POST("/card/restore", api.Card.Restore)
		user.POST("/card/copy", api.Card.Copy)
	}

	// 卡片中的任务和关联
	{
		user.POST("/task/create", api.Task.Create)
		user.POST("/task/rename", api.Task.Rename)
		user.POST("/task/done", api.Task.SetDone)
		user.POST("/task/move", api.Task.Move)
		user.POST("/task/delete", api.Task.Delete)
		user.POST("/card_link/add", api.Task.AddLink)
		user.POST("/card_link/delete", api.Task.DeleteLink)
	}

	// 上传、附件、封面和文件管理
	{
		user.POST("/upload/prepare", api.Attachment.PrepareUpload)
		user.POST("/upload/chunk", api.Attachment.UploadChunk)
		user.POST("/attachment/create_file", api.Attachment.CreateFile)
		user.POST("/attachment/create_link", api.Attachment.CreateLink)
		user.POST("/attachment/rename", api.Attachment.Rename)
		user.POST("/attachment/delete", api.Attachment.Delete)
		user.POST("/card/cover", api.Attachment.SetCover)
		user.GET("/user_file/list", api.Attachment.ListFiles)
		user.POST("/user_file/delete", api.Attachment.DeleteFile)
	}

	// 通知渠道和卡片的渠道选择
	{
		user.POST("/notify_channel/create", api.NotifyChannel.Create)
		user.POST("/notify_channel/update", api.NotifyChannel.Update)
		user.POST("/notify_channel/delete", api.NotifyChannel.Delete)
		user.POST("/notify_channel/test", api.NotifyChannel.Test)
		user.POST("/card/notify_channel", api.NotifyChannel.SetCardChannel)
	}

	// 文件内容和缩略图：经过 mw.FileAuth，可以用文件 Cookie 认证
	{
		files.GET("/attachment/:id", api.FileDownload.Attachment)
		files.GET("/attachment/:id/thumbnail/:size", api.FileDownload.AttachmentThumbnail)
		files.GET("/user_file/:id", api.FileDownload.UserFile)
		files.GET("/user_file/:id/thumbnail/:size", api.FileDownload.UserFileThumbnail)
	}

	// 管理员：账号管理
	{
		admin.GET("/user/list", api.Admin.ListUsers)
		admin.POST("/user/create", api.Admin.CreateUser)
		admin.POST("/user/disable", api.Admin.DisableUser)
		admin.POST("/user/enable", api.Admin.EnableUser)
		admin.POST("/user/reset_password", api.Admin.ResetPassword)
		admin.POST("/user/delete", api.Admin.DeleteUser)
	}
}
