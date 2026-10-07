package server

import (
	"context"
	"fmt"
	"kanban/cmd/app/server/app"
	"kanban/cmd/app/server/global/db"
	"kanban/cmd/app/server/global/storage"
	"kanban/cmd/app/server/service"
	"kanban/cmd/config"

	"github.com/urfave/cli/v3"
)

// Flags 是 server 子命令的参数。
var Flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "config",
		Usage:   "config file path",
		Value:   "./data/config.yaml",
		Aliases: []string{"c"},
	},
}

// Action 加载配置和数据库，然后启动 HTTP 服务。
func Action(ctx context.Context, cmd *cli.Command) error {
	if err := config.Load(cmd.String("config")); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	if err := db.Load(); err != nil {
		return fmt.Errorf("加载数据库失败: %w", err)
	}
	if err := storage.Load(); err != nil {
		return fmt.Errorf("打开文件存储失败: %w", err)
	}
	service.StartFileJobs(ctx)
	service.StartDeviceJobs(ctx)
	service.Notifier.Start(ctx)
	return app.Run()
}
