package lib

import (
	"net/url"
)

type RecogniserOptions struct {
	Name string
	HttpOptions
}

type HttpOptions struct {
	QueryParameters url.Values `json:"queryParameters"`
}
