package example

import (
	"context"
	"kanban/cmd/config"

	"github.com/urfave/cli/v3"
)

// Flags 是 example 子命令的参数。
var Flags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "config",
		Usage:   "Config",
		Value:   false,
		Aliases: []string{"c"},
	},
	&cli.StringFlag{
		Name:    "path",
		Usage:   "path",
		Value:   "./",
		Aliases: []string{"p"},
	},
}

// Action 输出请求的示例内容。
func Action(_ context.Context, cmd *cli.Command) error {
	if cmd.Bool("config") {
		return config.GenerateExample(cmd.String("path"))
	}
	return nil
}
