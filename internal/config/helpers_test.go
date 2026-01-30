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

package config

import (
	"bytes"

	. "gopkg.in/check.v1"
)

func (s *ConfigTestSuite) TestIsValidScheme(c *C) {
	var t bool
	for _, scheme := range ValidSchemes {
		t = IsValidScheme(scheme)
		c.Assert(t, Equals, true)
	}
	t = IsValidScheme("asdsadf")
	c.Assert(t, Equals, false)
}

func (s *ConfigTestSuite) TestrunTextValidate(c *C) {
	var testTextConfigGood = []byte(`#butlerstart
some text
some more text
#butlerend`)
	var testTextConfigBad1 = []byte(`some text
some more text
#butlerend`)
	var testTextConfigBad2 = []byte(`#butlerstart
some text
some more text`)
	var testTextConfigBad3 = []byte(`some text
some more text`)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigGood), "test-manager"), IsNil)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigBad1), "test-manager"), NotNil)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigBad2), "test-manager"), NotNil)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigBad3), "test-manager"), NotNil)
}

func (s *ConfigTestSuite) TestrunJsonValidate(c *C) {
	var testJSONConfigGood = []byte(`{"foo": "bar", "baz": ["one", "two", "three"] }`)
	var testJSONConfigBad = []byte(`{"foo": "bar", ["one", "two", "three"] }`)
	c.Assert(runJSONValidate(bytes.NewReader(testJSONConfigGood), "test-manager"), IsNil)
	c.Assert(runJSONValidate(bytes.NewReader(testJSONConfigBad), "test-manager"), NotNil)
}

func (s *ConfigTestSuite) TestrunYamlValidate(c *C) {
	var testYamlConfigGood = []byte(`#butlerstart
modules:
  http_200x:
    prober: http
    http:
  icmp:
    prober:icmp
#butlerend`)
	var testYamlConfigBad1 = []byte(`modules:
  http_200x:
    prober: http
    http:
  icmp:
    prober:icmp
#butlerend`)
	var testYamlConfigBad2 = []byte(`modules:
http_200x:
    prober: http
    http:
  icmp:
    prober:icmp`)
	// Test with skipButlerHeader = false (default behavior, requires headers)
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigGood), "test-manager", false), IsNil)
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigBad1), "test-manager", false), NotNil)
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigBad2), "test-manager", false), NotNil)

	// Test with skipButlerHeader = true (skips header validation, only checks YAML syntax)
	var testYamlNoHeaders = []byte(`modules:
  http_200x:
    prober: http
    http:
  icmp:
    prober: icmp`)
	c.Assert(runYamlValidate(bytes.NewReader(testYamlNoHeaders), "test-manager", true), IsNil)
	// Bad YAML syntax should still fail even with skipButlerHeader = true
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigBad2), "test-manager", true), NotNil)
}

func (s *ConfigTestSuite) TestgetFileExtension(c *C) {
	c.Assert(getFileExtension("foo.yaml"), Equals, "yaml")
	c.Assert(getFileExtension("foo.yml"), Equals, "yaml")
	c.Assert(getFileExtension("foo.json"), Equals, "json")
	c.Assert(getFileExtension("foo.asdfasdf"), Equals, "text")
}

func (s *ConfigTestSuite) TestcheckButlerHeaderFooter(c *C) {
	c.Assert(checkButlerHeaderFooter([]byte(butlerHeader)), Equals, true)
	c.Assert(checkButlerHeaderFooter([]byte(butlerFooter)), Equals, true)
	c.Assert(checkButlerHeaderFooter([]byte("asdfawsdf")), Equals, false)
}
