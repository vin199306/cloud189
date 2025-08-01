package cmd

import (
	"fmt"
	"os"

	"github.com/gowsp/cloud189/internal/session"
	"github.com/gowsp/cloud189/pkg"
	"github.com/gowsp/cloud189/pkg/file"
	"github.com/spf13/cobra"
)

var upCfg pkg.UploadConfig
var mvAfterUpload bool

func init() {
	upCmd.Flags().Uint32VarP(&upCfg.Num, "parallel", "p", 5, "number of parallels for file upload")
	upCmd.Flags().StringVarP(&upCfg.Parten, "name", "n", "", "filter filename regular expression")
	upCmd.Flags().BoolVar(&mvAfterUpload, "mv", false, "上传成功后删除本地文件")
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "upload file",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		length := len(args)
		cloud := session.Join(args[length-1])
		err := file.CheckPath(cloud)
		if err != nil {
			fmt.Println(err)
			return
		}
		locals := args[:length-1]

		if err := App().Upload(upCfg, cloud, locals...); err != nil {
			if mvAfterUpload {
				for _, f := range locals {
					if err[f] == "exist" || err[f] == "completed" {
						err := os.RemoveAll(f)
						fmt.Printf("file: %s", err[f])	
						if err != nil {
							fmt.Printf("删除本地文件失败: %s, 错误: %v\n", f, err)
						} else {
							fmt.Printf("已删除本地文件: %s\n", f)
						}
					}

				}
			}
		}
	},
}
