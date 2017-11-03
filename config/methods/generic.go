package methods

import (
	"errors"
)

func NewGenericMethod(manager string, entry string) (Method, error) {
	return GenericMethod{}, errors.New("Generic method handler is not very useful")
}

type GenericMethod struct {
}

func (m GenericMethod) Get(file string, args ...interface{}) (*Response, error) {
	var (
		result *Response
	)
	return result, errors.New("Generic method handler is not very useful")
}
