package db

import (
	"fmt"
	"kanban/cmd/config"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// createPostgres 连接 PostgreSQL 服务，目标数据库不存在时创建，然后连接目标数据库。
// 检查和创建数据库时不指定 dbname，按 PostgreSQL 的默认规则连接与用户名同名的数据库。
func createPostgres() (exist bool, d *gorm.DB, err error) {
	dbname := config.Global.Database.Database

	server, err := gorm.Open(postgres.Open(postgresDSN("")), gormConfig())
	if err != nil {
		return false, nil, fmt.Errorf("连接 PostgreSQL 服务异常: %w", err)
	}
	serverDB, err := server.DB()
	if err != nil {
		return false, nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	defer func() { _ = serverDB.Close() }()

	if err = serverDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbname).Scan(&exist); err != nil {
		return false, nil, fmt.Errorf("查询 PostgreSQL 数据库异常: %w", err)
	}
	if !exist {
		if _, err = serverDB.Exec("CREATE DATABASE " + quoteIdentifier(dbname)); err != nil {
			return false, nil, fmt.Errorf("创建 PostgreSQL 数据库异常: %w", err)
		}
	}

	d, err = loadPostgres()
	return exist, d, err
}

// loadPostgres 连接配置中的目标数据库。
func loadPostgres() (*gorm.DB, error) {
	d, err := gorm.Open(postgres.Open(postgresDSN(config.Global.Database.Database)), gormConfig())
	if err != nil {
		return nil, fmt.Errorf("连接 PostgreSQL 数据库异常: %w", err)
	}
	return d, nil
}

// postgresDSN 按配置生成连接串，dbname 为空时不指定数据库。
// 会话时区固定为 UTC：业务时间都按时刻保存，展示时再按用户时区换算。
func postgresDSN(dbname string) string {
	database := config.Global.Database
	parts := []string{
		"host=" + quoteDSNValue(database.Host),
		fmt.Sprintf("port=%d", database.Port),
		"user=" + quoteDSNValue(database.Username),
		"password=" + quoteDSNValue(database.Password),
		"sslmode=disable",
		"TimeZone=UTC",
	}
	if dbname != "" {
		parts = append(parts, "dbname="+quoteDSNValue(dbname))
	}
	return strings.Join(parts, " ")
}

// quoteDSNValue 按 libpq 键值连接串规则给值加单引号，避免空值或含空格的密码破坏连接串。
func quoteDSNValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

// quoteIdentifier 给 SQL 标识符加双引号，用于 CREATE DATABASE 这类不能使用参数占位的语句。
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
