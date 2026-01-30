/*
Copyright 2017-2026 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package reloaders

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ReloaderTestSuite struct{}

var _ = Suite(&ReloaderTestSuite{})

func (s *ReloaderTestSuite) TestNewReloaderError(c *C) {
	err := NewReloaderError()
	c.Assert(err, NotNil)
	c.Assert(err.Code, Equals, 0)
	c.Assert(err.Message, Equals, "")
}

func (s *ReloaderTestSuite) TestReloaderErrorWithCode(c *C) {
	err := NewReloaderError().WithCode(500)
	c.Assert(err.Code, Equals, 500)
}

func (s *ReloaderTestSuite) TestReloaderErrorWithMessage(c *C) {
	err := NewReloaderError().WithMessage("test error message")
	c.Assert(err.Message, Equals, "test error message")
}

func (s *ReloaderTestSuite) TestReloaderErrorChaining(c *C) {
	err := NewReloaderError().WithCode(404).WithMessage("not found")
	c.Assert(err.Code, Equals, 404)
	c.Assert(err.Message, Equals, "not found")
}

func (s *ReloaderTestSuite) TestReloaderErrorErrorMethod(c *C) {
	err := NewReloaderError().WithCode(500).WithMessage("internal server error")
	errStr := err.Error()
	c.Assert(errStr, Equals, "internal server error. code=500")
}

func (s *ReloaderTestSuite) TestGenericReloader(c *C) {
	reloader, err := NewGenericReloader("test-manager", "generic", []byte("test"))
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Generic reloader is not very useful")

	// Test the interface methods
	c.Assert(reloader.GetMethod(), Equals, "none")
	c.Assert(reloader.Reload(), IsNil)
}

func (s *ReloaderTestSuite) TestGenericReloaderWithCustomError(c *C) {
	customErr := NewReloaderError().WithCode(1).WithMessage("custom error")
	reloader, err := NewGenericReloaderWithCustomError("test-manager", "generic", customErr)
	c.Assert(err, Equals, customErr)
	c.Assert(reloader.GetMethod(), Equals, "none")
}

func (s *ReloaderTestSuite) TestGenericReloaderSetCounter(c *C) {
	reloader, _ := NewGenericReloader("test-manager", "generic", []byte("test"))
	result := reloader.SetCounter(5)
	// GenericReloader.SetCounter returns the same reloader (no-op)
	c.Assert(result.GetMethod(), Equals, "none")
}

func (s *ReloaderTestSuite) TestGenericReloaderGetOpts(c *C) {
	reloader, _ := NewGenericReloader("test-manager", "generic", []byte("test"))
	opts := reloader.GetOpts()
	c.Assert(opts, NotNil)
}

func (s *ReloaderTestSuite) TestGenericReloaderSetOpts(c *C) {
	reloader := GenericReloader{}
	result := reloader.SetOpts(GenericReloaderOpts{})
	c.Assert(result, Equals, true)
}
