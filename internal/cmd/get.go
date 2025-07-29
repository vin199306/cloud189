package cmd

import (
	"fmt"

	"github.com/gowsp/cloud189/internal/session"
	"github.com/gowsp/cloud189/pkg/file"
	"github.com/spf13/cobra"
	"github.com/gowsp/cloud189/pkg/app"
	"github.com/gowsp/cloud189/pkg/invoker"
	"github.com/gowsp/cloud189/pkg/drive"
)

var getCmd = &cobra.Command{
    Use:    "get {云盘路径}",
    PreRun: session.Parse,
    Short:  "获取云盘文件的详细信息",
    Args:   cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        err := file.CheckPath(args...)
        if err != nil {
            fmt.Println(err)
            return
        }
        name := args[0]
        api := app.New(invoker.DefaultPath())
		f := drive.New(api)
		info, err := f.Stat(name)
		if err != nil {
			return
		}
		fmt.Println(info)
    },
}