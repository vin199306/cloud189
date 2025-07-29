package cmd

import (
	"fmt"

	"github.com/gowsp/cloud189/internal/session"
	"github.com/gowsp/cloud189/pkg/file"
	"github.com/gowsp/cloud189/pkg/app"
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
        client := App()
        fileInfo, err := client.Stat(name)
        if err != nil {
            fmt.Println(err)
            return
        }
        fmt.Printf("文件 ID: %s\n", fileInfo.Id())
        fmt.Printf("文件名: %s\n", fileInfo.Name())
        fmt.Printf("文件大小: %s\n", file.ReadableSize(uint64(fileInfo.Size())))
        fmt.Printf("修改时间: %s\n", fileInfo.ModTime().Format("2006-01-02 15:04:05"))
        fmt.Printf("是否为目录: %v\n", fileInfo.IsDir())
    },
}