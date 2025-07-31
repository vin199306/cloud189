package drive

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/gowsp/cloud189/pkg"
	"github.com/gowsp/cloud189/pkg/file"
)

func (client *FS) UploadFrom(file pkg.Upload) error {
	uploader := client.api.Uploader()
	_, err := uploader.Write(file)
	return err
}
func (client *FS) Upload(cfg pkg.UploadConfig, cloud string, locals ...string) map[string]string {
	err := cfg.Check()
	results := make(map[string]string)
	if err != nil {
		for _, local := range locals {
			results[local] = err.Error()
		}
		return results
	}
	dir, err := client.stat(cloud)
	if len(locals) > 1 || os.IsNotExist(err) {
		client.Mkdir(cloud[1:])
		dir, _ = client.stat(cloud)
	}
	up := make([]pkg.Upload, 0)
	localMap := make(map[pkg.Upload]string) // 记录upload对象和本地路径的映射
	for _, local := range locals {
		if file.IsNetFile(local) {
			u := file.NewURLFile(dir.Id(), local)
			up = append(up, u)
			localMap[u] = local
			continue
		}
		if file.IsFastFile(local) {
			continue
		}
		files, err := client.uploadLocal(dir, local, cfg.Parten)
		if err != nil {
			results[local] = err.Error()
			continue
		}
		for _, u := range files {
			up = append(up, u)
			localMap[u] = local
		}
	}
	task := cfg.NewTask()
	uploader := client.api.Uploader()
	for _, v := range up {
		r := v
		task.Run(func() {
			result, err := uploader.Write(r)
			if err != nil {
				results[localMap[r]] = err.Error()
			} else {
				results[localMap[r]] = result
			}

		})
	}
	task.Close()
	return results
}

func (client *FS) uploadLocal(parent pkg.File, local string, parten string) ([]pkg.Upload, error) {
	stat, err := os.Stat(local)
	if err != nil {
		return nil, err
	}
	up := make([]pkg.Upload, 0)
	if !stat.IsDir() {
		up = append(up, file.NewLocalFile(parent.Id(), local))
		return up, nil
	}
	dirs := map[string]string{
		".": parent.Id(),
	}
	parten = filepath.Join(local, parten)
	files, err := filepath.Glob(parten)
	if err != nil {
		return nil, err
	}
	for _, localFile := range files {
		info, err := os.Stat(localFile)
		if err != nil {
			log.Println(err)
			continue
		}
		if info.IsDir() {
			filepath.WalkDir(localFile, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					rel := file.Rel(local, path)
					if rel == "." {
						return nil
					}
					f, _ := client.api.Mkdir(parent, rel)
					dirs[rel] = f.Id()
					return nil
				}
				dir, _ := filepath.Split(path)
				rel := file.Rel(local, dir)
				up = append(up, file.NewLocalFile(dirs[rel], path))
				return err
			})
		} else {
			up = append(up, file.NewLocalFile(parent.Id(), localFile))
		}

	}
	return up, nil
}
