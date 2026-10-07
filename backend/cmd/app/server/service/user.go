package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/passwd"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/common/timezone"
	"kanban/cmd/app/server/common/types/editormode"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/common/types/panmodifier"
	"kanban/cmd/app/server/common/types/role"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm"
)

// User 是账号管理服务：创建初始管理员，以及管理员对普通用户的创建、停用、启用、重置密码和删除。
var User = new(userService)

type userService struct{}

// 用户名长度按字符计算。
const (
	usernameMinLength = 1
	usernameMaxLength = 32
)

// validateUsername 检查用户名：长度 1–32 个字符，不含空白和控制字符。
func validateUsername(username string) error {
	length := utf8.RuneCountInString(username)
	if length < usernameMinLength || length > usernameMaxLength {
		return badRequest(resp.UsernameLength, "用户名长度为 1 到 32 个字符")
	}
	for _, r := range username {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return badRequest(resp.UsernameWhitespace, "用户名不能包含空白字符")
		}
	}
	return nil
}

// validateNickname 检查昵称：去掉首尾空白后 1–32 个字符，可以含空格，不能含控制字符。不必唯一。
func validateNickname(nickname string) (string, error) {
	nickname = strings.TrimSpace(nickname)
	length := utf8.RuneCountInString(nickname)
	if length < usernameMinLength || length > usernameMaxLength {
		return "", badRequest(resp.NicknameLength, "昵称长度为 1 到 32 个字符")
	}
	for _, r := range nickname {
		if unicode.IsControl(r) {
			return "", badRequest(resp.NicknameControl, "昵称不能包含控制字符")
		}
	}
	return nickname, nil
}

// newUser 按默认设置构造新用户：时区取服务端当前时区，配色为清爽，语言跟随系统，正文模板为空即使用随语言的默认模板，不打开主看板。昵称初始与用户名相同。
func newUser(username, password string, rol role.Type, now time.Time) (*model.User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := passwd.Validate(password); err != nil {
		return nil, passwordInvalid(err)
	}
	hash, err := passwd.Hash(password)
	if err != nil {
		return nil, err
	}
	return &model.User{
		Username:       username,
		Nickname:       username,
		PasswordHash:   hash,
		Role:           rol,
		CreatedAt:      now,
		Timezone:       timezone.Server(),
		Palette:        palette.Default,
		Locale:         "",
		RemindTemplate: "",
		DueTemplate:    "",
		Shortcuts:      shortcut.Bindings{},
		EditorMode:     editormode.Default,
		PanModifier:    panmodifier.Default,
	}, nil
}

// create 在事务中检查用户名并写入新用户。
func (s *userService) create(ctx context.Context, user *model.User) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		exist, err := repository.User.ExistsByUsername(ctx, user.Username)
		if err != nil {
			return err
		}
		if exist {
			return conflict(resp.UsernameTaken, "用户名已被使用")
		}
		return repository.User.Create(ctx, user)
	})
}

// EnsureInitialAdmin 在库中没有任何用户时创建初始管理员。
// 返回 created 表示本次是否创建；库中已有用户时不做任何修改，也不校验传入的用户名和密码。
func (s *userService) EnsureInitialAdmin(ctx context.Context, username, password string) (created bool, err error) {
	err = repository.Transaction(ctx, func(ctx context.Context) error {
		count, err := repository.User.Count(ctx)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if username == "" || password == "" {
			return badRequest(resp.InitialAdminRequired, "库中还没有用户，需要提供初始管理员的用户名和密码")
		}
		user, err := newUser(username, password, role.Admin, time.Now().UTC())
		if err != nil {
			return err
		}
		if err = s.create(ctx, user); err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

// UpdateProfile 修改自己的用户名和昵称。用户名仍用于登录，不能与其他人重复。改名不吊销已登录的设备。
func (s *userService) UpdateProfile(ctx context.Context, userID int64, username, nickname string) (*model.User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	nickname, err := validateNickname(nickname)
	if err != nil {
		return nil, err
	}
	var user *model.User
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		var err error
		if user, err = repository.User.FindByID(ctx, userID); err != nil {
			return err
		}
		if username != user.Username {
			exist, err := repository.User.ExistsByUsername(ctx, username)
			if err != nil {
				return err
			}
			if exist {
				return conflict(resp.UsernameTaken, "用户名已被使用")
			}
		}
		user.Username, user.Nickname = username, nickname
		if err = repository.User.Update(ctx, userID, map[string]any{"username": username, "nickname": nickname}); err != nil {
			return err
		}
		return recordChange(ctx, userID, hub.OpUpsert, EntityAccount, userID, dro.NewAccount(user))
	})
	return user, err
}

// Create 由管理员创建普通用户，只需要用户名和密码，其余设置取默认值。昵称初始与用户名相同。
func (s *userService) Create(ctx context.Context, username, password string) (*model.User, error) {
	user, err := newUser(username, password, role.User, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err = s.create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// List 返回全部账号。管理接口只返回账号信息，不涉及用户的看板数据。
func (s *userService) List(ctx context.Context) ([]*model.User, error) {
	return repository.User.List(ctx)
}

// FindByID 按 ID 查找用户，不存在时返回业务错误。
func (s *userService) FindByID(ctx context.Context, id int64) (*model.User, error) {
	user, err := repository.User.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.UserNotFound, "用户不存在")
	}
	return user, err
}

// managedTarget 查找管理员要操作的账号。管理员不能通过管理接口操作自己，自己的密码在个人设置中修改。
func (s *userService) managedTarget(ctx context.Context, actorID, targetID int64) (*model.User, error) {
	if actorID == targetID {
		return nil, forbidden(resp.SelfActionForbidden, "不能对自己的账号执行此操作")
	}
	return s.FindByID(ctx, targetID)
}

// Disable 停用账号：记录停用时间，吊销该账号的全部设备令牌并断开连接。数据全部保留。
func (s *userService) Disable(ctx context.Context, actorID, targetID int64) error {
	return write(ctx, func(ctx context.Context) error {
		target, err := s.managedTarget(ctx, actorID, targetID)
		if err != nil {
			return err
		}
		if target.Disabled() {
			return nil
		}
		if err = repository.User.Update(ctx, targetID, map[string]any{"disabled_at": time.Now().UTC()}); err != nil {
			return err
		}
		if err = revokeDevicesExcept(ctx, targetID, 0); err != nil {
			return err
		}
		// 停用后用户的卡片不再满足发送条件，立即清理未发送的记录。
		return wakeNotifier(ctx)
	})
}

// Enable 启用已停用的账号。停用时吊销的令牌不会恢复，用户需要重新登录。
func (s *userService) Enable(ctx context.Context, actorID, targetID int64) error {
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		if _, err := s.managedTarget(ctx, actorID, targetID); err != nil {
			return err
		}
		return repository.User.Update(ctx, targetID, map[string]any{"disabled_at": nil})
	})
	if err == nil {
		Notifier.Wake() // 停用期间到达的通知在启用后补发
	}
	return err
}

// ResetPassword 由管理员重置密码，并吊销该账号的全部设备令牌。
func (s *userService) ResetPassword(ctx context.Context, actorID, targetID int64, password string) error {
	if err := passwd.Validate(password); err != nil {
		return passwordInvalid(err)
	}
	hash, err := passwd.Hash(password)
	if err != nil {
		return err
	}
	return write(ctx, func(ctx context.Context) error {
		if _, err := s.managedTarget(ctx, actorID, targetID); err != nil {
			return err
		}
		if err := repository.User.Update(ctx, targetID, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		return revokeDevicesExcept(ctx, targetID, 0)
	})
}

// Delete 删除账号及其全部数据，提交后断开该账号的全部连接。
func (s *userService) Delete(ctx context.Context, actorID, targetID int64) error {
	return write(ctx, func(ctx context.Context) error {
		if _, err := s.managedTarget(ctx, actorID, targetID); err != nil {
			return err
		}
		if err := deleteUserData(ctx, targetID); err != nil {
			return err
		}
		return afterCommit(ctx, func() { hub.Default.DisconnectUser(targetID) })
	})
}

// deleteUserData 删除属于该用户的全部记录，最后删除用户本身。用户已不存在，因此不记录变更。
// 表之间没有数据库外键，新增带 user_id 的表时必须在这里加上对应的删除。
func deleteUserData(ctx context.Context, userID int64) error {
	if err := repository.Device.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.ChangeLog.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := deleteUserFiles(ctx, userID); err != nil {
		return err
	}
	if err := repository.Notify.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.CardLink.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.CardAction.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.Task.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.Card.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.List.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.PriorityLevel.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.Label.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.Panel.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.Board.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	if err := repository.Folder.DeleteByUser(ctx, userID); err != nil {
		return err
	}
	return repository.User.Delete(ctx, userID)
}
