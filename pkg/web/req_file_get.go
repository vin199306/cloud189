package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gowsp/cloud189/pkg"
)

type fileInfo struct {
	ParentId    json.Number `json:"parentId,omitempty"`
	FileId      json.Number `json:"fileId,omitempty"`
	FileName    string      `json:"fileName,omitempty"`
	FileSize    int64       `json:"fileSize,omitempty"`
	IsFolder    bool        `json:"isFolder,omitempty"`
	FileModTime int64       `json:"lastOpTime,omitempty"`
	CreateTime  int64       `json:"createTime,omitempty"`
	FileCount   int64       `json:"subFileCount,omitempty"`
	DownloadUrl string      `json:"downloadUrl,omitempty"`
}

func (f *fileInfo) Id() string         { return f.FileId.String() }
func (f *fileInfo) PId() string        { return f.ParentId.String() }
func (f *fileInfo) Name() string       { return f.FileName }
func (f *fileInfo) Size() int64        { return f.FileSize }
func (f *fileInfo) Mode() os.FileMode  { return os.ModePerm }
func (f *fileInfo) ModTime() time.Time { return time.UnixMilli(f.FileModTime) }
func (f *fileInfo) IsDir() bool        { return f.IsFolder }
func (f *fileInfo) Sys() interface{} {
	return pkg.FileExt{
		FileCount:   f.FileCount,
		DownloadUrl: "https:" + f.DownloadUrl,
		CreateTime:  time.UnixMilli(f.CreateTime),
	}
}
func (f *fileInfo) ContentType(ctx context.Context) (string, error) {
	return path.Ext(f.Name()), nil
}
func (f *fileInfo) ETag(ctx context.Context) (string, error) {
	return strconv.FormatInt(f.FileModTime, 10), nil
}

func (c *api) GetFileInfo(path string) (*fileInfo, error) {
	var info fileInfo
	err := c.invoker.Get("/getFileInfo.action", url.Values{"filePath": {path}}, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}