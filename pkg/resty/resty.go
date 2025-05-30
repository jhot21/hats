package resty

import "github.com/go-resty/resty/v2"

type RestyOption func(*resty.Request)

func SetQueryParam(key, value string) RestyOption {
	return func(r *resty.Request) {
		r.SetQueryParam(key, value)
	}
}
