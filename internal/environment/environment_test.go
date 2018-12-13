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

package environment

import (
	. "gopkg.in/check.v1"

	"os"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ButlerTestSuite struct {
}

var _ = Suite(&ButlerTestSuite{})

/*
func (s *ButlerTestSuite) SetUpSuite(c *C) {
	//ParseConfigFiles(&Files, FileList)
}
*/

func (s *ButlerTestSuite) TestGetVar(c *C) {
	Test1 := GetVar(1)
	c.Assert(Test1, Equals, "1")

	Test2 := GetVar("hi")
	c.Assert(Test2, Equals, "hi")

	Test3 := GetVar("env:DOES_NOT_EXIST")
	c.Assert(Test3, Equals, "")

	os.Setenv("DOES_EXIST", "YES")
	Test4 := GetVar("env:DOES_EXIST")
	c.Assert(Test4, Equals, "YES")
	os.Unsetenv("DOES_EXIST")

	Test5 := GetVar("what what what")
	c.Assert(Test5, Equals, "what what what")

	type Foo struct {
		Bar string
	}
	Test6 := GetVar(Foo{})
	c.Assert(Test6, Equals, "")
}
