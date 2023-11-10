package gokapi

import (
	"fmt"
	"os"
	"path/filepath"

	"code.jhot.me/jhot/hats/internal/util"
	"github.com/gabriel-vasile/mimetype"
	"github.com/go-resty/resty/v2"
)

type GokapiClient struct {
	host       string
	restClient *resty.Client
}

func New(host string, token string) *GokapiClient {
	return &GokapiClient{
		host: host,
		restClient: resty.New().
			SetHeader("apikey", token).
			SetBaseURL(fmt.Sprintf("%s/api", host)),
	}
}

func NewFromEnv() *GokapiClient {
	return New(os.Getenv("GOKAPI_HOST"), os.Getenv("GOKAPI_TOKEN"))
}

type GokapiFile struct {
	ID                           string `json:"Id"`
	Name                         string `json:"Name"`
	Size                         string `json:"Size"`
	HotlinkId                    string `json:"HotlinkId"`
	ContentType                  string `json:"ContentType"`
	ExpireAt                     int64  `json:"ExpireAt"`
	SizeBytes                    int64  `json:"SizeBytes"`
	ExpireAtString               string `json:"ExpireAtString"`
	DownloadsRemaining           int    `json:"DownloadsRemaining"`
	DownloadCount                int    `json:"DownloadCount"`
	UnlimitedDownloads           bool   `json:"UnlimitedDownloads"`
	UnlimitedTime                bool   `json:"UnlimitedTime"`
	RequiresClientSideDecryption bool   `json:"RequiresClientSideDecryption"`
	IsEncrypted                  bool   `json:"IsEncrypted"`
	IsPasswordProtected          bool   `json:"IsPasswordProtected"`
	IsSavedOnLocalStorage        bool   `json:"IsSavedOnLocalStorage"`
}

func (c *GokapiClient) ListFiles() ([]GokapiFile, error) {
	var data []GokapiFile
	_, err := util.CheckSuccess(c.restClient.R().SetHeader("Accept", "application/json").SetResult(&data).Get("files/list"))
	if err != nil {
		return data, err
	}

	return data, nil
}

type UploadOptions struct {
	AllowedDownloads int
	ExpiryDays       int
	Password         string
	FilenameOverride string
}

func (c *GokapiClient) UploadFile(filePath string, opts *UploadOptions) (GokapiFile, error) {
	var fileName string
	if opts.FilenameOverride != "" {
		fileName = opts.FilenameOverride
	} else {
		fileName = filepath.Base(filePath)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return GokapiFile{}, err
	}
	defer f.Close()

	mimeType, _ := mimetype.DetectFile(filePath)

	req := c.restClient.R().
		SetMultipartField("file", fileName, mimeType.String(), f).
		SetMultipartFormData(map[string]string{
			"allowedDownloads": fmt.Sprintf("%d", opts.AllowedDownloads),
			"expiryDays":       fmt.Sprintf("%d", opts.ExpiryDays),
			"password":         opts.Password,
		})

	var data struct {
		Result   string     `json:"Result"`
		FileInfo GokapiFile `json:"FileInfo"`
	}

	_, err = util.CheckSuccess(req.SetResult(&data).Post("files/add"))
	if err != nil {
		return GokapiFile{}, err
	}

	return data.FileInfo, nil
}

func (c *GokapiClient) GetDownloadUrl(f GokapiFile) string {
	return fmt.Sprintf("%s/downloadFile?id=%s", c.host, f.ID)
}
