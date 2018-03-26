package methods

import (
	"errors"
	"net/url"
)

func NewGenericMethod(manager *string, entry *string) (Method, error) {
	return GenericMethod{}, errors.New("Generic method handler is not very useful")
}

type GenericMethod struct {
}

func (m GenericMethod) Get(u *url.URL) (*Response, error) {
	var (
		result *Response
	)
	return result, errors.New("Generic method handler is not very useful")
}
