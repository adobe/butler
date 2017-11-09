package methods

import (
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
)

type Method interface {
	Get(string) (*Response, error)
}

type Response struct {
	body       io.ReadCloser
	statusCode int
}

func (r Response) GetResponseBody() io.ReadCloser {
	return r.body
}

func (r Response) GetResponseStatusCode() int {
	return r.statusCode
}

func New(manager *string, method string, entry *string) (Method, error) {
	method = strings.ToLower(method)
	log.Debugf("methods.New() manager=%v method=%v entry=%v", manager, method, entry)
	switch method {
	case "http", "https":
		return NewHttpMethod(manager, entry)
	case "S3", "s3":
		return NewS3Method(manager, entry)
	default:
		return NewGenericMethod(manager, entry)
	}
}
