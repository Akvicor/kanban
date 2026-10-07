package app

import (
	"kanban/cmd/def"

	"github.com/urfave/cli/v3"
)

// App 是 Kanban 的 CLI 根命令。
var App *cli.Command

func init() {
	App = &cli.Command{
		Name:                   def.AppName,
		Usage:                  def.AppUsage,
		Version:                def.AppVersion(),
		Description:            def.AppDescription,
		Authors:                []any{"Akvicor <akvicor@ksyaki.co>"},
		UseShortOptionHandling: true,
		DefaultCommand:         "help",
		Commands:               commands,
	}
}
