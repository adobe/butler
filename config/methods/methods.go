package methods

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Method interface {
	Get(string) (*http.Response, error)
}

func New(manager string, method string, entry string) (Method, error) {
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
