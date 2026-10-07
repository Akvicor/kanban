package dto

import (
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/common/types/editormode"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/common/types/panmodifier"
)

// ID 是只携带对象 ID 的请求。
type ID struct {
	ID int64 `json:"id"`
}

// Login 是登录请求。DeviceName 由客户端按浏览器和系统生成，用于在设备列表中区分设备。
type Login struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name"`
}

// ChangePassword 是用户修改自己密码的请求。
type ChangePassword struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UpdateSettings 是个人设置修改请求。字段缺省表示不修改该项。
// Shortcuts 只需包含要修改的操作，值为该操作的全部按键；空数组表示不绑定按键。
type UpdateSettings struct {
	Timezone            *string           `json:"timezone"`
	Palette             *palette.Type     `json:"palette"`
	Locale              *locale.Type      `json:"locale"`       // 界面与通知语言；空值表示跟随系统
	PanModifier         *panmodifier.Type `json:"pan_modifier"` // 拖动平移看板的修饰键；空值表示关闭手势
	OpenMainBoardOnHome *bool             `json:"open_main_board_on_home"`
	RemindTemplate      *string           `json:"remind_template"`
	DueTemplate         *string           `json:"due_template"`
	Shortcuts           shortcut.Bindings `json:"shortcuts"`
	EditorMode          *editormode.Type  `json:"editor_mode"`
}

// UpdateProfile 是修改自己的用户名和昵称的请求。两项都要提供。
type UpdateProfile struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

// AdminCreateUser 是管理员创建用户的请求，只需要用户名和密码。
type AdminCreateUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AdminResetPassword 是管理员重置用户密码的请求。
type AdminResetPassword struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}
