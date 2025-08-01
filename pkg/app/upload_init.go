package app

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gowsp/cloud189/pkg"
	"github.com/gowsp/cloud189/pkg/file"
	"github.com/gowsp/cloud189/pkg/invoker"
	"github.com/gowsp/cloud189/pkg/util"
)

type Upload struct {
	session *invoker.Session
	invoker *invoker.Invoker
}

func (client *api) Uploader() pkg.ReadWriter {
	client.invoker.Get("/keepUserSession.action", nil, "")
	return &Upload{session: client.conf.Session, invoker: client.invoker}
}
func (client *Upload) Write(upload pkg.Upload) (string, error) {
	data, err := client.init(upload)
	if err != nil {
		return "error", err
	}
	if data.IsExists() {
		uploadResult, err := client.commit(upload, data.UploadFileId, "0")
		if len(uploadResult.File.Id) > 16 {
			return "exist", err
		}
		return "error", err
	}
	count := upload.SliceNum()
	parts := make([]pkg.UploadPart, count)
	names := make([]string, count)
	for i := 0; i < count; i++ {
		part := upload.Part(int64(i))
		parts[i] = part
		names[i] = fmt.Sprintf("%d-%s", i+1, part.Name())
	}
	rsp, err := client.getUploadUrl(data.UploadFileId, names)
	if err != nil {
		return "error", err
	}
	err = rsp.upload(upload, parts)
	if err != nil {
		return "error", err
	}
	uploadResult, err := client.commit(upload, data.UploadFileId, strconv.Itoa(upload.SliceNum()))
	if len(uploadResult.File.Id) > 16 {
		return "completed", err
	}
	return "error", err
}

func (up *Upload) encrypt(f url.Values) string {
	e := util.EncodeParam(f)
	data := util.AesEncrypt([]byte(e), []byte(up.session.Secret[0:16]))
	return hex.EncodeToString(data)
}

func (up *Upload) do(req *http.Request, retry int, result any) error {
	resp, err := up.invoker.DoWithResp(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		defer resp.Body.Close()
		return json.NewDecoder(resp.Body).Decode(&result)
	}
	var e uperror
	json.NewDecoder(resp.Body).Decode(&e)
	if e.Code == "UserDayFlowOverLimited" {
		return errors.New("上传超过当日流量限制")
	}
	if retry > 5 {
		return err
	}
	time.Sleep(time.Second)
	return up.do(req, retry+1, result)
}

type uperror struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func (i *Upload) Get(path string, params url.Values, result any) error {
	vals := make(url.Values)
	vals.Set("params", i.encrypt(params))
	req, err := http.NewRequest(http.MethodGet, "https://upload.cloud.189.cn"+path+"?"+vals.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("decodefields", "familyId,parentFolderId,fileName,fileMd5,fileSize,sliceMd5,sliceSize,albumId,extend,lazyCheck,isLog")
	req.Header.Set("accept", "application/json;charset=UTF-8")
	req.Header.Set("cache-control", "no-cache")
	return i.do(req, 0, result)
}

type uploadInfo struct {
	UploadType     int    `json:"uploadType,omitempty"`
	UploadHost     string `json:"uploadHost,omitempty"`
	UploadFileId   string `json:"uploadFileId,omitempty"`
	FileDataExists int    `json:"fileDataExists,omitempty"`
}

func (i *uploadInfo) IsExists() bool {
	return i.FileDataExists == 1
}

type initResp struct {
	Code string     `json:"code,omitempty"`
	Data uploadInfo `json:"data,omitempty"`
}

func (r *initResp) GetCode() string {
	return r.Code
}

func (c *Upload) init(i pkg.Upload) (*uploadInfo, error) {
	params := make(url.Values)
	params.Set("parentFolderId", i.ParentId())
	params.Set("fileName", i.Name())
	params.Set("fileSize", strconv.FormatInt(i.Size(), 10))
	params.Set("sliceSize", strconv.Itoa(file.Slice))

	if i.LazyCheck() {
		params.Set("lazyCheck", "1")
	} else {
		params.Set("fileMd5", i.FileMD5())
		params.Set("sliceMd5", i.SliceMD5())
	}
	params.Set("extend", `{"opScene":"1","relativepath":"","rootfolderid":""}`)
	var upload initResp
	if err := c.Get(apiPath("/person/initMultiUpload"), params, &upload); err != nil {
		return nil, err
	}
	if upload.Data.UploadFileId == "" {
		return nil, errors.New("error get upload fileid")
	}
	return &upload.Data, nil
}

type uploadUrlResp struct {
	Code string `json:"code,omitempty"`
	Data map[string]struct {
		RequestURL    string `json:"requestURL,omitempty"`
		RequestHeader string `json:"requestHeader,omitempty"`
	} `json:"uploadUrls,omitempty"`
}

func (rsp *uploadUrlResp) upload(info pkg.Upload, parts []pkg.UploadPart) error {
	print := os.Getenv("EXE_MODE") == "1"
	if print {
		log.Println("start upload", info.Name())
	}
	for _, part := range parts {
		num := strconv.Itoa(part.Num() + 1)
		upload := rsp.Data["partNumber_"+num]
		req, _ := http.NewRequest(http.MethodPut, upload.RequestURL, part.Data())
		headers := strings.Split(upload.RequestHeader, "&")
		for _, v := range headers {
			i := strings.Index(v, "=")
			req.Header.Set(v[0:i], v[i+1:])
		}
		if print {
			log.Println("upload part", num, "/", len(parts))
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("upload error %s", string(data))
		}
	}
	if print {
		log.Println("upload", info.Name(), "completed")
	}
	return nil
}

func (client *Upload) getUploadUrl(fileId string, names []string) (*uploadUrlResp, error) {
	p := make(url.Values)
	p.Set("partInfo", strings.Join(names, ","))
	p.Set("uploadFileId", fileId)
	urlResp := new(uploadUrlResp)
	return urlResp, client.Get(apiPath("/person/getMultiUploadUrls"), p, urlResp)
}

type uploadResult struct {
	Code string `json:"code,omitempty"`
	File struct {
		Id         string `json:"userFileId,omitempty"`
		FileSize   int64  `json:"fileSize,omitempty"`
		FileName   string `json:"fileName,omitempty"`
		FileMd5    string `json:"fileMd5,omitempty"`
		CreateDate string `json:"createDate,omitempty"`
	} `json:"file,omitempty"`
}

func (r *uploadResult) GetCode() string {
	return r.Code
}

func (client *Upload) commit(i pkg.Upload, fileId, lazyCheck string) (uploadResult, error) {
	var result uploadResult
	params := make(url.Values)
	if lazyCheck == "1" {
		params.Set("fileMd5", i.FileMD5())
		params.Set("sliceMd5", i.SliceMD5())
		params.Set("lazyCheck", lazyCheck)
	}
	params.Set("uploadFileId", fileId)
	if i.Overwrite() {
		params.Set("opertype", "3")
	}
	err := client.Get(apiPath("/person/commitMultiUploadFile"), params, &result)
	// 打印服务器返回的详细信息
	fmt.Printf("userFileId: %s\n", result.File.Id)
	fmt.Printf("fileSize: %d\n", result.File.FileSize)
	return result, err
}

func apiPath(path string) string {
	// if cmd.UseFamilyCloud() {
	// 	return "/family" + path[len("/person"):]
	// }
	return path
}
