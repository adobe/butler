package main

import (
	. "gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ButlerTestSuite struct {
}

var _ = Suite(&ButlerTestSuite{})

func (s *ButlerTestSuite) SetUpSuite(c *C) {
	ParseConfigFiles(&Files, FileList)
}

/*
func (s *ButlerTestSuite) TestParseConfigFilesJsonOkDefault(c *C) {
	err := ParseConfigFilesJson(&Files, "")
	c.Assert(err, IsNil)
	c.Assert(Files.Files, HasLen, 3)
}

func (s *ButlerTestSuite) TestParseConfigFilesJsonOkCustom(c *C) {
	configFiles := `{"files": ["prometheus.yml", "alerts/commonalerts.yml", "alerts/tenant.yml", "a/b"]}`
	err := ParseConfigFilesJson(&Files, configFiles)
	c.Assert(err, IsNil)
	c.Assert(Files.Files, HasLen, 4)
}

func (s *ButlerTestSuite) TestParseConfigFilesJsonNotOkCustom(c *C) {
	configFiles := `{"files": ["prometheus.yml", "alerts/commonalerts.yml", "alerts/tenant.yml",}`
	err := ParseConfigFilesJson(&Files, configFiles)
	c.Assert(err, IsNil)
	c.Assert(Files.Files, HasLen, 3)
}
*/

func (s *ButlerTestSuite) TestParseConfigFilesOkDefault(c *C) {
	err := ParseConfigFiles(&Files, FileList)
	c.Assert(err, IsNil)
	c.Assert(Files.Files, HasLen, 3)
}

func (s *ButlerTestSuite) TestParseConfigFilesOkCustom(c *C) {
	configFiles := "prometheus.yml,alerts/alerts.yml  , alerts/tenant.yml, heyfoo.yml,,"
	err := ParseConfigFiles(&Files, configFiles)
	c.Assert(err, IsNil)
	c.Assert(Files.Files, HasLen, 4)
}

func (s *ButlerTestSuite) TestGetPrometheusPaths(c *C) {
	paths := GetPrometheusPaths()
	c.Assert(paths, HasLen, 3)
}

func (s *ButlerTestSuite) TestGetPCMSUrls(c *C) {
	urls := GetPCMSUrls()
	c.Assert(urls, HasLen, 3)
}
