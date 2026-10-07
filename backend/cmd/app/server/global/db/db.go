package db

import (
	"context"
	"errors"
	"fmt"
	"kanban/cmd/config"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

var db *gorm.DB
var dblock = &sync.RWMutex{}

// Get 返回当前数据库连接，Create 或 Load 成功前为 nil。
func Get() *gorm.DB {
	dblock.RLock()
	defer dblock.RUnlock()
	return db
}

// Create 连接数据库，数据库不存在时先创建。只供 migrate 使用。
// exist 表示连接前数据库是否已经存在。
func Create() (exist bool, err error) {
	dblock.Lock()
	defer dblock.Unlock()
	var d *gorm.DB
	switch config.Global.Database.Type {
	case config.DatabaseSQLite:
		exist, d, err = createSQLite()
	case config.DatabasePostgres:
		exist, d, err = createPostgres()
	default:
		err = fmt.Errorf("不支持的数据库类型: %q", config.Global.Database.Type)
	}
	if err != nil {
		return exist, err
	}
	db = d
	return exist, nil
}

// Load 连接已存在的数据库。数据库必须先由 migrate 创建，server 不会自动建库。
func Load() error {
	dblock.Lock()
	defer dblock.Unlock()
	var d *gorm.DB
	var err error
	switch config.Global.Database.Type {
	case config.DatabaseSQLite:
		d, err = loadSQLite()
	case config.DatabasePostgres:
		d, err = loadPostgres()
	default:
		err = fmt.Errorf("不支持的数据库类型: %q", config.Global.Database.Type)
	}
	if err != nil {
		return err
	}
	db = d
	return nil
}

// Ping 检查当前数据库连接是否可用。
func Ping(ctx context.Context) error {
	d := Get()
	if d == nil {
		return errors.New("数据库尚未加载")
	}
	sqlDB, err := d.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}
	if err = sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库连接检查失败: %w", err)
	}
	return nil
}

// gormConfig 是两种数据库共用的 gorm 配置。
// 迁移时不创建外键约束，数据之间的关联和级联由服务层在事务中维护，两种数据库行为一致。
func gormConfig() *gorm.Config {
	logMode := logger.Default.LogMode(logger.Silent)
	if utils.Contains(config.Global.Log.Debug, "database") {
		logMode = logger.Default.LogMode(logger.Info)
	}
	return &gorm.Config{
		Logger:                                   logMode,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	}
}
