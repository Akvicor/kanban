package repository

import (
	"context"
	"database/sql"
	"kanban/cmd/app/server/global/db"
	"kanban/cmd/config"

	"gorm.io/gorm"
)

// txKey 是 context 中保存事务连接的键。
type txKey struct{}

// conn 返回 ctx 中的事务连接；不在事务中时返回全局连接。仓储函数都通过它访问数据库。
func conn(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.Get().WithContext(ctx)
}

// InTransaction 判断 ctx 是否已经处在事务中。
func InTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(txKey{}).(*gorm.DB)
	return ok
}

// Transaction 在事务中执行 f。ctx 已经处在事务中时直接复用该事务，因此服务函数可以互相组合。
func Transaction(ctx context.Context, f func(ctx context.Context) error) error {
	return transaction(ctx, nil, f)
}

// ReadTransaction 在只读事务中执行 f，事务内的多次查询看到同一时刻的数据。
// 用于读取「同步序号 + 对应数据」这类必须一致的组合。
// PostgreSQL 默认的读已提交级别下，两条查询之间可能插入别的提交，因此使用可重复读；SQLite 的事务本身就是一致的快照。
func ReadTransaction(ctx context.Context, f func(ctx context.Context) error) error {
	var options *sql.TxOptions
	if config.Global.Database.Type == config.DatabasePostgres {
		options = &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	}
	return transaction(ctx, options, f)
}

func transaction(ctx context.Context, options *sql.TxOptions, f func(ctx context.Context) error) error {
	if InTransaction(ctx) {
		return f(ctx)
	}
	return db.Get().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return f(context.WithValue(ctx, txKey{}, tx))
	}, options)
}

// affected 把「没有命中任何行」转换为 gorm.ErrRecordNotFound，便于服务层区分不存在和其他错误。
func affected(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
