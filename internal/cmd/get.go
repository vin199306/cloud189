package cmd

import (
	"fmt"
	"reflect" 
	"github.com/gowsp/cloud189/internal/session"
	"github.com/gowsp/cloud189/pkg/file"
	"github.com/spf13/cobra"
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
		info, err := App().Stat(name)
		if err != nil {
			return
		}
		// 使用反射打印所有方法
        fmt.Println("=== 文件对象支持的方法 ===")
        t := reflect.TypeOf(info)
        for i := 0; i < t.NumMethod(); i++ {
            method := t.Method(i)
            fmt.Printf("%s: %s\n", method.Name, method.Type)
        }
    },
}