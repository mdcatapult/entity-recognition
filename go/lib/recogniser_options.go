package lib

import (
	"net/url"
)

type RecogniserOptions struct {
	HttpOptions
}

type HttpOptions struct {
	QueryParameters url.Values `json:"query_parameters"`
}
