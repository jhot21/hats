package main

import (
	"os"

	"github.com/go-resty/resty/v2"
)

func main() {
	c := resty.New()
	resp, err := c.R().Get("http://127.0.0.1:8888/status")

	if err != nil || !resp.IsSuccess() {
		os.Exit(1)
	}

	if string(resp.Body()) == "OK" {
		os.Exit(0)
	}

	os.Exit(1)
}
