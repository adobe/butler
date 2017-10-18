package methods

import (
	"errors"
	"net/http"
)

func NewGenericMethod(manager string, entry string) (Method, error) {
	return GenericMethod{}, errors.New("Generic method handler is not very useful")
}

type GenericMethod struct {
}

func (m GenericMethod) Get(file string) (*http.Response, error) {
	var (
		result *http.Response
	)
	return result, errors.New("Generic method handler is not very useful")
}
