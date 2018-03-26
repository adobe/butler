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
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigGood)), IsNil)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigBad1)), NotNil)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigBad2)), NotNil)
	c.Assert(runTextValidate(bytes.NewReader(testTextConfigBad3)), NotNil)
}

func (s *ConfigTestSuite) TestrunJsonValidate(c *C) {
	var testJsonConfigGood = []byte(`{"foo": "bar", "baz": ["one", "two", "three"] }`)
	var testJsonConfigBad = []byte(`{"foo": "bar", ["one", "two", "three"] }`)
	c.Assert(runJsonValidate(bytes.NewReader(testJsonConfigGood)), IsNil)
	c.Assert(runJsonValidate(bytes.NewReader(testJsonConfigBad)), NotNil)
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
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigGood)), IsNil)
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigBad1)), NotNil)
	c.Assert(runYamlValidate(bytes.NewReader(testYamlConfigBad2)), NotNil)
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
