package db_test

import (
	"context"
	"kanban/cmd/app/server/global/db"
	"kanban/cmd/app/server/testutil/dbtest"
	"kanban/cmd/config"
	"os"
	"path/filepath"
	"testing"
)

// useConfig 在测试期间替换全局配置，并在结束时关闭连接、恢复原配置。
func useConfig(t *testing.T, database config.DatabaseModel) {
	t.Helper()
	previous := config.Global
	config.Global = &config.Model{Database: database}
	t.Cleanup(func() {
		closeCurrent()
		config.Global = previous
	})
}

// closeCurrent 关闭当前连接，避免临时数据库在测试结束时仍被占用。
func closeCurrent() {
	if d := db.Get(); d != nil {
		if sqlDB, err := d.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}

// checkCreateAndLoad 验证首次 Create 新建数据库，再次 Create 识别为已存在，之后 Load 可以连接。
func checkCreateAndLoad(t *testing.T) {
	t.Helper()
	exist, err := db.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if exist {
		t.Fatal("首次 Create() 报告数据库已存在")
	}
	closeCurrent()

	if exist, err = db.Create(); err != nil || !exist {
		t.Fatalf("再次 Create() exist=%v err=%v", exist, err)
	}
	closeCurrent()

	if err = db.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err = db.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

// TestSQLiteCreateAndLoad 验证 SQLite：Load 不会新建数据库文件，只有 Create 会。
func TestSQLiteCreateAndLoad(t *testing.T) {
	file := filepath.Join(t.TempDir(), "kanban.db")
	useConfig(t, config.DatabaseModel{Type: config.DatabaseSQLite, File: file})

	if err := db.Load(); err == nil {
		t.Fatal("数据库文件不存在时 Load() 未返回错误")
	}
	checkCreateAndLoad(t)
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("Create() 未写出数据库文件: %v", err)
	}
}

// TestPostgresCreateAndLoad 验证 PostgreSQL：Create 创建不存在的数据库。实例由 dbtest 的环境变量指定。
func TestPostgresCreateAndLoad(t *testing.T) {
	database, ok := dbtest.Postgres(t)
	if !ok {
		t.Skip("未设置 KANBAN_TEST_POSTGRES_HOST，跳过 PostgreSQL 测试")
	}
	useConfig(t, database)
	t.Cleanup(func() { dbtest.DropPostgres(t, database) })
	checkCreateAndLoad(t)
}
