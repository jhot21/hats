package gokapi

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhot21/hats/internal/util"
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

func (c *GokapiClient) UploadBytes(data []byte, opts *UploadOptions) (GokapiFile, error) {
	mimeType := mimetype.Detect(data)
	if opts.FilenameOverride == "" {
		opts.FilenameOverride = strings.TrimRight(base64.StdEncoding.EncodeToString(data), "=") + mimeType.Extension()
	}

	req := c.restClient.R().
		SetFileReader("file", opts.FilenameOverride, bytes.NewReader(data)).
		SetMultipartFormData(map[string]string{
			"allowedDownloads": fmt.Sprintf("%d", opts.AllowedDownloads),
			"expiryDays":       fmt.Sprintf("%d", opts.ExpiryDays),
			"password":         opts.Password,
		})

	var returnData struct {
		Result   string     `json:"Result"`
		FileInfo GokapiFile `json:"FileInfo"`
	}

	_, err := util.CheckSuccess(req.SetResult(&returnData).Post("files/add"))
	if err != nil {
		return GokapiFile{}, err
	}

	return returnData.FileInfo, nil
}

func (c *GokapiClient) UploadFile(filePath string, opts *UploadOptions) (GokapiFile, error) {
	if opts.FilenameOverride == "" {
		opts.FilenameOverride = filepath.Base(filePath)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return GokapiFile{}, err
	}
	defer f.Close()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return GokapiFile{}, fmt.Errorf("error reading file: %w", err)
	}
	return c.UploadBytes(data, opts)
}

func (c *GokapiClient) GetDownloadUrl(f GokapiFile) string {
	return fmt.Sprintf("%s/downloadFile?id=%s", c.host, f.ID)
}
