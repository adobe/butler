package main

import (
	. "gopkg.in/check.v1"
	"testing"

	log "github.com/sirupsen/logrus"
)

func Test(t *testing.T) { TestingT(t) }

type ButlerTestSuite struct {
}

var _ = Suite(&ButlerTestSuite{})

func (s *ButlerTestSuite) SetUpSuite(c *C) {
	//ParseConfigFiles(&Files, FileList)
}

// Test Suite for the butler prometheus SUCCESS/FAILURE enumeration
func (s *ButlerTestSuite) TestPrometheusEnums(c *C) {
	// FAILURE and SUCCESS are float64, hence the decimal point.
	c.Assert(FAILURE, Equals, 0.0)
	c.Assert(SUCCESS, Equals, 1.0)
}

// Test Suite for butler.SetLogLevel()
func (s *ButlerTestSuite) TestSetLogLevel(c *C) {
	tests := []struct {
		name	string
		level	log.Level
	}{
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"fatal", log.FatalLevel},
		{"panic", log.PanicLevel},
		{"breakme!", log.InfoLevel},
	}
	for _, entry := range tests {
		logLevel := SetLogLevel(entry.name)
		c.Assert(logLevel, Equals, entry.level)
	}
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
/*
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
*/
