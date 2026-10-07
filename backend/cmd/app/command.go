package app

import (
	"kanban/cmd/app/example"
	"kanban/cmd/app/migrate"
	"kanban/cmd/app/server"

	"github.com/urfave/cli/v3"
)

// commands 是根命令下的子命令：启动服务、迁移数据库、生成示例配置。
var commands = []*cli.Command{
	{
		Name:                   "server",
		Usage:                  "HTTP Server",
		UseShortOptionHandling: true,
		Action:                 server.Action,
		Flags:                  server.Flags,
	},
	{
		Name:                   "migrate",
		Usage:                  "Migrate Database",
		UseShortOptionHandling: true,
		Action:                 migrate.Action,
		Flags:                  migrate.Flags,
	},
	{
		Name:                   "example",
		Usage:                  "Generate Example",
		UseShortOptionHandling: true,
		Action:                 example.Action,
		Flags:                  example.Flags,
	},
}
