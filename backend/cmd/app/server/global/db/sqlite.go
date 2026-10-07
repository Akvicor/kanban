package db

import (
	"context"
	"fmt"
	"kanban/cmd/config"

	"github.com/Akvicor/util"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// createSQLite 打开 SQLite 数据库，文件不存在时新建。
func createSQLite() (exist bool, d *gorm.DB, err error) {
	exist = util.FileStat(config.Global.Database.File).IsExist()
	d, err = openSQLite("rwc")
	if err != nil {
		return exist, nil, err
	}
	// SQLite 在第一次访问时才真正写出数据库文件。
	sqlDB, err := d.DB()
	if err != nil {
		return exist, nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}
	if err = sqlDB.PingContext(context.Background()); err != nil {
		_ = sqlDB.Close()
		return exist, nil, fmt.Errorf("创建 SQLite 数据库失败: %w", err)
	}
	return exist, d, nil
}

// loadSQLite 打开已存在的 SQLite 数据库文件。
func loadSQLite() (*gorm.DB, error) {
	if util.FileStat(config.Global.Database.File).NotFile() {
		return nil, fmt.Errorf("数据库文件不存在: %s", config.Global.Database.File)
	}
	return openSQLite("rw")
}

// openSQLite 使用纯 Go SQLite 驱动打开数据库，mode 是 SQLite URI 的 mode 参数。
//
// 多个请求会同时写库，连接参数按并发写入设置：
//   - journal_mode(WAL)：读和写互不阻塞。数据库目录中会多出 -wal 和 -shm 文件，备份时与数据库文件一起复制。
//   - busy_timeout：遇到其他连接持有写锁时等待，而不是立即失败。
//   - _txlock=immediate：事务开始时就取得写锁，避免两个事务都先读后写时互相等待而失败。
//
// 不使用 cache=shared：共享缓存下锁冲突直接报错，busy_timeout 不起作用。
func openSQLite(mode string) (*gorm.DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=%s&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_txlock=immediate", config.Global.Database.File, mode)
	d, err := gorm.Open(sqlite.Open(dsn), gormConfig())
	if err != nil {
		return nil, fmt.Errorf("连接 SQLite 数据库异常: %w", err)
	}
	return d, nil
}
