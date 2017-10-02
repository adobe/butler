package methods

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Method interface {
	Get(string) (*http.Response, error)
}

func New(method string, entry string) (Method, error) {
	method = strings.ToLower(method)
	log.Debugf("methods.New() method=%v entry=%v", method, entry)
	switch method {
	case "http", "https":
		return NewHttpMethod(entry)
	default:
		return NewGenericMethod(entry)
	}
}
