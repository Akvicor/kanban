// Package dbtest 为测试准备临时数据库，只被 _test.go 文件引用，不参与正式程序。
//
// Run 在 SQLite 和 PostgreSQL 上各运行一次测试函数，保证业务逻辑在两种数据库上行为一致。
// SQLite 使用测试临时目录中的数据库文件；PostgreSQL 通过环境变量指定实例，未设置时跳过：
//
//	KANBAN_TEST_POSTGRES_HOST、KANBAN_TEST_POSTGRES_PORT、KANBAN_TEST_POSTGRES_USER、KANBAN_TEST_POSTGRES_PASSWORD
//
// PostgreSQL 测试为每次运行创建独立的临时数据库，结束后删除。
package dbtest

import (
	"fmt"
	"kanban/cmd/app/server/global/db"
	"kanban/cmd/app/server/model"
	"kanban/cmd/config"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// counter 让同一进程内多次创建的 PostgreSQL 临时库名不重复。
var counter atomic.Int64

// Run 在每种可用的数据库上建好表结构后运行 f。
func Run(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	t.Run("sqlite", func(t *testing.T) {
		use(t, config.DatabaseModel{
			Type: config.DatabaseSQLite,
			File: filepath.Join(t.TempDir(), "kanban.db"),
		})
		f(t)
	})
	t.Run("postgres", func(t *testing.T) {
		database, ok := Postgres(t)
		if !ok {
			t.Skip("未设置 KANBAN_TEST_POSTGRES_HOST，跳过 PostgreSQL")
		}
		use(t, database)
		f(t)
	})
}

// Postgres 按环境变量返回一个新的临时库配置，未设置环境变量时返回 false。
func Postgres(t *testing.T) (config.DatabaseModel, bool) {
	t.Helper()
	host := os.Getenv("KANBAN_TEST_POSTGRES_HOST")
	if host == "" {
		return config.DatabaseModel{}, false
	}
	port, err := strconv.Atoi(os.Getenv("KANBAN_TEST_POSTGRES_PORT"))
	if err != nil {
		t.Fatalf("KANBAN_TEST_POSTGRES_PORT 无效: %v", err)
	}
	return config.DatabaseModel{
		Type:     config.DatabasePostgres,
		Host:     host,
		Port:     port,
		Database: fmt.Sprintf("kanban_test_%d_%d_%d", os.Getpid(), time.Now().UnixNano(), counter.Add(1)),
		Username: os.Getenv("KANBAN_TEST_POSTGRES_USER"),
		Password: os.Getenv("KANBAN_TEST_POSTGRES_PASSWORD"),
	}, true
}

// use 切换全局配置和数据库连接到临时库，建好表结构，并在测试结束时关闭连接、删除临时库、恢复原配置。
func use(t *testing.T, database config.DatabaseModel) {
	t.Helper()
	previous := config.Global
	config.Global = &config.Model{Database: database}
	t.Cleanup(func() {
		closeCurrent()
		if database.Type == config.DatabasePostgres {
			DropPostgres(t, database)
		}
		config.Global = previous
	})

	if _, err := db.Create(); err != nil {
		t.Fatalf("创建临时数据库失败: %v", err)
	}
	if err := db.Get().AutoMigrate(model.All()...); err != nil {
		t.Fatalf("创建表结构失败: %v", err)
	}
}

// closeCurrent 关闭当前数据库连接。
func closeCurrent() {
	if d := db.Get(); d != nil {
		if sqlDB, err := d.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}

// DropPostgres 关闭当前连接，连接与用户名同名的默认库，删除测试创建的临时库。
// 调用后全局配置指向默认库，调用方需要自行恢复原配置。
func DropPostgres(t *testing.T, database config.DatabaseModel) {
	t.Helper()
	closeCurrent()
	maintenance := database
	maintenance.Database = database.Username
	config.Global = &config.Model{Database: maintenance}
	if err := db.Load(); err != nil {
		t.Errorf("删除临时库前连接失败: %v", err)
		return
	}
	defer closeCurrent()
	if err := db.Get().Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, database.Database)).Error; err != nil {
		t.Errorf("删除临时库 %s 失败: %v", database.Database, err)
	}
}
