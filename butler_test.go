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

// Test Suite for butler.SetLogLevel()
func (s *ButlerTestSuite) TestSetLogLevel(c *C) {
	tests := []struct {
		name  string
		level log.Level
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
