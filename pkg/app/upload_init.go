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
func (client *Upload) Write(upload pkg.Upload) error {
	data, err := client.init(upload)
	if err != nil {
		return err
	}
	if data.IsExists() {
		return client.commit(upload, data.UploadFileId, "0")
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
		return err
	}
	err = rsp.upload(upload, parts)
	if err != nil {
		return err
	}
	return client.commit(upload, data.UploadFileId, "1")
}

func (up *Upload) encrypt(f url.Values) string {
	e := util.EncodeParam(f)
	data := util.AesEncrypt([]byte(e), []byte("062F8E1529A61D66ECE686F4439A2238"[0:16]))
	return hex.EncodeToString(data)
}

// Decrypt 解密函数
func (up *Upload) decrypt(encrypted string) url.Values{
	// 1. 将十六进制字符串转换为字节切片
	data, err := hex.DecodeString(encrypted)
	if err != nil {

	}

	// 2. 使用 AES 算法解密
	decryptedData := util.AesDecrypt([]byte(up.session.Secret[0:16]), []byte(data))

	// 3. 将解密后的字节切片转换为字符串
	decryptedStr := string(decryptedData)

	// 4. 将字符串解析为 url.Values 类型
	params, err := url.ParseQuery(decryptedStr)
	if err != nil {

	}

	return params
}

func (up *Upload) do(req *http.Request, retry int, result any) error {
	resp, err := up.invoker.DoWithResp(req)
	log.Printf("resp: %v", resp)
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
	log.Printf("decrypt: %v", i.decrypt("e9ff75c806d40d8e398852caf8ffb4378002ca37d2f1ec51e5727a005fbf5aaf533cc77c541afb457159d2dd30de6460c878c1b8bd9e4090556f31447c1a58557937329b1e5e446a660196b051599f8710c886740819014bfc38edb11f0aad5b87b2c26b4e736883baed435463e26da9f23a26e0637cfb7191641f4d7e83526231dd6daa6870693ec5ad09bf8b9a7c604203891dfecdcfc6e8dcf59ac376b37e068bdb64ec69731e6c1f14d8d033865c80b925b106da510b4c1336e6030847285ca79075774c9c9de45bf33a056cc313d"))
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

const initMultiUploadURL = "/family/initMultiUpload"

func (c *Upload) init(i pkg.Upload) (*uploadInfo, error) {
    // 检查关键参数的有效性
    if i.ParentId() == "" {
        return nil, errors.New("parentFolderId is empty")
    }
    if i.Name() == "" {
        return nil, errors.New("fileName is empty")
    }

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
    if err := c.Get(initMultiUploadURL, params, &upload); err != nil {
        log.Printf("Failed to call initMultiUpload: %v", err)
        return nil, err
    }

    if upload.Data.UploadFileId == "" {
        log.Printf("Failed to call initMultiUpload: %v", upload.Data)
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
			log.Println("upload part", num)
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
	return urlResp, client.Get("/family/getMultiUploadUrls", p, urlResp)
}

type uploadResult struct {
	Code string `json:"code,omitempty"`
	File struct {
		Id         string `json:"userFileId,omitempty"`
		FileSize   int64  `json:"file_size,omitempty"`
		FileName   string `json:"file_name,omitempty"`
		FileMd5    string `json:"file_md_5,omitempty"`
		CreateDate string `json:"create_date,omitempty"`
	} `json:"file,omitempty"`
}

func (r *uploadResult) GetCode() string {
	return r.Code
}

func (client *Upload) commit(i pkg.Upload, fileId, lazyCheck string) error {
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
	return client.Get("/family/commitMultiUploadFile", params, &result)
}
