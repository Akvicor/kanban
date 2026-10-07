package repository

import (
	"context"
	"kanban/cmd/app/server/common/shortcut"
	"kanban/cmd/app/server/model"

	"gorm.io/gorm"
)

// User 是用户表的仓储。
var User = new(userRepository)

type userRepository struct{}

// FindByID 按 ID 查找用户。
func (*userRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	user := new(model.User)
	return user, conn(ctx).Where("id = ?", id).Take(user).Error
}

// FindByUsername 按用户名查找用户，不区分大小写。库里仍保存用户填写时的大小写。
func (*userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	user := new(model.User)
	return user, conn(ctx).Where("LOWER(username) = LOWER(?)", username).Take(user).Error
}

// ExistsByUsername 判断用户名是否已被使用，不区分大小写。
func (*userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := conn(ctx).Model(&model.User{}).Where("LOWER(username) = LOWER(?)", username).Count(&count).Error
	return count > 0, err
}

// Count 统计用户数。
func (*userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := conn(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

// List 按创建顺序返回全部用户。
func (*userRepository) List(ctx context.Context) ([]*model.User, error) {
	users := make([]*model.User, 0)
	return users, conn(ctx).Order("id ASC").Find(&users).Error
}

// Create 新建用户。
func (*userRepository) Create(ctx context.Context, user *model.User) error {
	return conn(ctx).Create(user).Error
}

// Update 按 ID 更新指定列。values 的键是列名，零值也会写入。
func (*userRepository) Update(ctx context.Context, id int64, values map[string]any) error {
	return affected(conn(ctx).Model(&model.User{}).Where("id = ?", id).Updates(values))
}

// UpdateShortcuts 保存快捷键差异。快捷键列使用 JSON 序列化，按 map 更新时不会序列化，因此通过模型结构体写入。
func (*userRepository) UpdateShortcuts(ctx context.Context, id int64, overrides shortcut.Bindings) error {
	return affected(conn(ctx).Model(&model.User{}).Where("id = ?", id).Select("shortcuts").Updates(&model.User{Shortcuts: overrides}))
}

// NextRevision 把用户的同步序号加一并返回新值。必须在事务中调用：
// 更新会锁住用户行，同一用户的其他写事务要等本事务结束后才能取得下一个序号。
func (*userRepository) NextRevision(ctx context.Context, id int64) (int64, error) {
	if err := affected(conn(ctx).Model(&model.User{}).Where("id = ?", id).UpdateColumn("revision", gorm.Expr("revision + 1"))); err != nil {
		return 0, err
	}
	var revision int64
	return revision, conn(ctx).Model(&model.User{}).Where("id = ?", id).Select("revision").Take(&revision).Error
}

// Lock 锁住用户行直到事务结束，使同一用户的写入互斥执行。必须在事务中调用。
// 用于先读后写、读到的数据会影响写入结果的操作，例如按现有顺序计算新位置。
// 通过一次不改变值的更新取得行锁，SQLite 和 PostgreSQL 都适用；SQLite 的写事务本身已经互斥。
func (*userRepository) Lock(ctx context.Context, id int64) error {
	return affected(conn(ctx).Model(&model.User{}).Where("id = ?", id).UpdateColumn("revision", gorm.Expr("revision")))
}

// Delete 删除用户记录。
func (*userRepository) Delete(ctx context.Context, id int64) error {
	return affected(conn(ctx).Where("id = ?", id).Delete(&model.User{}))
}
