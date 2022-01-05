package lib

import (
	"net/url"
)

type RecogniserOptions struct {
	//RecogniserName string
	HttpOptions
}

type HttpOptions struct {
	QueryParameters url.Values `json:"queryParameters"`
}
