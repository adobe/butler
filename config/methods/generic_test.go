package methods

import (
	"net/url"
	//"testing"
	. "gopkg.in/check.v1"
)

var _ = Suite(&GenericTestSuite{})

type GenericTestSuite struct {
}

func (s *GenericTestSuite) TestNewGenericMethod(c *C) {
	method, err := NewGenericMethod(nil, nil)
	c.Assert(err, NotNil)
	c.Assert(method, Equals, GenericMethod{})
}

func (s *GenericTestSuite) TestGet(c *C) {
	method, err := NewGenericMethod(nil, nil)
	c.Assert(err, NotNil)
	c.Assert(method, Equals, GenericMethod{})

	u, err := url.Parse("hiya")
	c.Assert(err, IsNil)
	resp, err2 := method.Get(u)
	c.Assert(err2, NotNil)
	c.Assert(resp, IsNil)
}
