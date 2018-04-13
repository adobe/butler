/*
Copyright 2017 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

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
