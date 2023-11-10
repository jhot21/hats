package util

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

// CheckSuccess wraps a resty request and if the status code is not 2XX will make the error non-nil
func CheckSuccess(resp *resty.Response, err error) (*resty.Response, error) {
	if err == nil && !resp.IsSuccess() {
		err = fmt.Errorf("%d status received: %s", resp.StatusCode(), string(resp.Body()))
	}
	return resp, err
}
