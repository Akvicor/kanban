package migrate

import (
	"context"
	"fmt"
	"kanban/cmd/app/server/global/db"
	"kanban/cmd/app/server/model"
	"kanban/cmd/config"

	"github.com/urfave/cli/v3"
)

// Flags 是 migrate 子命令的参数。
var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "config",
		Usage:   "config file path",
		Value:   "./data/config.yaml",
		Aliases: []string{"c"},
	},
}

// Action 创建或连接数据库，创建和升级表结构，库中没有用户时按环境变量创建初始管理员。
func Action(ctx context.Context, cmd *cli.Command) error {
	if err := config.Load(cmd.String("config")); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if _, err := db.Create(); err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}
	if err := db.Get().AutoMigrate(model.All()...); err != nil {
		return fmt.Errorf("初始化数据库表结构失败: %w", err)
	}
	return initAdmin(ctx)
}
