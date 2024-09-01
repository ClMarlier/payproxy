package internal

import "net/http"

type Request struct {
	Method  string
	Headers http.Header
	Body    []byte
	Params  string
	Url     string
}
