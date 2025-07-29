package cmd

import (
	"fmt"

	"github.com/gowsp/cloud189/internal/session"
	"github.com/gowsp/cloud189/pkg/file"
	"github.com/gowsp/cloud189/pkg/web/fileInfo"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
    Use:    "get {云盘路径}",
    PreRun: session.Parse,
    Short:  "获取云盘文件的详细信息",
    Args:   cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        err := file.CheckPath(args...)
        if err != nil {
            fmt.Println(err)
            return
        }
        name := args[0]
		
		info, err := App().Stat(name)
		if err != nil {
			return
		}
		// 类型断言
		if info, ok := info.(*fileInfo); ok {
			fmt.Println(info.Info()) // 调用具体类型的方法
		} else {
			fmt.Println("无法转换为 fileInfo 类型")
		}
		//fmt.Println(info.Info().FileSize)			
    },
}