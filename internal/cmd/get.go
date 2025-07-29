package cmd

import (
	"fmt"
	"reflect" 
	"github.com/gowsp/cloud189/internal/session"
	//"github.com/gowsp/cloud189/pkg/file"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
    Use:    "get {云盘路径}",
    PreRun: session.Parse,
    Short:  "获取云盘文件的详细信息",
    Args:   cobra.MinimumNArgs(2),
    Run: func(cmd *cobra.Command, args []string) {
        // err := file.CheckPath(args...)
        // if err != nil {
        //     fmt.Println(err)
        //     return
        // }
        name := args[0]
		
		info, err := App().Stat(name)
		if err != nil {
			return
		}
		// 使用反射调用指定方法
		methodName := args[1]
        value := reflect.ValueOf(info)
        method := value.MethodByName(methodName)
        
        if !method.IsValid() {
            fmt.Printf("错误：方法 %s 不存在\n", methodName)
            fmt.Println("可用方法：Id, Name, Size, ModTime, IsDir, PId")
            return
        }
        
        // 调用方法
        results := method.Call(nil)
        
        // 处理返回值
        if len(results) > 0 {
            // 格式化输出不同类型的返回值
            for _, result := range results {
                if result.IsValid() {
                    fmt.Printf("%s: %v\n", methodName, result.Interface())
                }
            }
        } else {
            fmt.Printf("方法 %s 没有返回值\n", methodName)
        }
    },
}