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
	. "gopkg.in/check.v1"
)

func (s *ConfigTestSuite) TestNewValidateOpts(c *C) {
	opts := NewValidateOpts()
	c.Assert(opts.ContentType, Equals, "text")
	c.Assert(opts.SkipButlerHeader, Equals, false)
	c.Assert(opts.Data, IsNil)
	c.Assert(opts.FileName, Equals, "")
	c.Assert(opts.Manager, Equals, "")
}

func (s *ConfigTestSuite) TestValidateOptsWithContentType(c *C) {
	opts := NewValidateOpts().WithContentType("yaml")
	c.Assert(opts.ContentType, Equals, "yaml")
}

func (s *ConfigTestSuite) TestValidateOptsWithData(c *C) {
	testData := []byte("test data")
	opts := NewValidateOpts().WithData(testData)
	c.Assert(opts.Data, DeepEquals, testData)
}

func (s *ConfigTestSuite) TestValidateOptsWithFileName(c *C) {
	opts := NewValidateOpts().WithFileName("config.yaml")
	c.Assert(opts.FileName, Equals, "config.yaml")
}

func (s *ConfigTestSuite) TestValidateOptsWithManager(c *C) {
	opts := NewValidateOpts().WithManager("prometheus")
	c.Assert(opts.Manager, Equals, "prometheus")
}

func (s *ConfigTestSuite) TestValidateOptsWithSkipButlerHeader(c *C) {
	opts := NewValidateOpts().WithSkipButlerHeader(true)
	c.Assert(opts.SkipButlerHeader, Equals, true)
}

func (s *ConfigTestSuite) TestValidateOptsChaining(c *C) {
	testData := []byte("test data")
	opts := NewValidateOpts().
		WithContentType("json").
		WithData(testData).
		WithFileName("test.json").
		WithManager("alertmanager").
		WithSkipButlerHeader(true)

	c.Assert(opts.ContentType, Equals, "json")
	c.Assert(opts.Data, DeepEquals, testData)
	c.Assert(opts.FileName, Equals, "test.json")
	c.Assert(opts.Manager, Equals, "alertmanager")
	c.Assert(opts.SkipButlerHeader, Equals, true)
}

func (s *ConfigTestSuite) TestRepoFileEventSetSuccess(c *C) {
	rfe := &RepoFileEvent{
		Success: make(map[string]bool),
		Error:   make(map[string]error),
		TmpFile: make(map[string]string),
	}

	err := rfe.SetSuccess("config.yml", nil)
	c.Assert(err, IsNil)
	c.Assert(rfe.Success["config.yml"], Equals, true)
	c.Assert(rfe.Error["config.yml"], IsNil)
}

func (s *ConfigTestSuite) TestRepoFileEventSetFailure(c *C) {
	rfe := &RepoFileEvent{
		Success: make(map[string]bool),
		Error:   make(map[string]error),
		TmpFile: make(map[string]string),
	}

	testErr := NewReloaderErrorForTest("test error")
	err := rfe.SetFailure("config.yml", testErr)
	c.Assert(err, IsNil)
	c.Assert(rfe.Success["config.yml"], Equals, false)
	c.Assert(rfe.Error["config.yml"], Equals, testErr)
}

func (s *ConfigTestSuite) TestRepoFileEventSetTmpFile(c *C) {
	rfe := &RepoFileEvent{
		Success: make(map[string]bool),
		Error:   make(map[string]error),
		TmpFile: make(map[string]string),
	}

	err := rfe.SetTmpFile("config.yml", "/tmp/butler-12345")
	c.Assert(err, IsNil)
	c.Assert(rfe.TmpFile["config.yml"], Equals, "/tmp/butler-12345")
}

func (s *ConfigTestSuite) TestConfigSettingsGetAllConfigLocalPaths(c *C) {
	// Create a ConfigSettings with managers
	cs := &ConfigSettings{
		Managers: map[string]*Manager{
			"prometheus": {
				DestPath:          "/opt/prometheus",
				PrimaryConfigName: "prometheus.yml",
				ManagerOpts: map[string]*ManagerOpts{
					"prometheus.repo1": {
						AdditionalConfigsFullLocalPaths: []string{
							"/opt/prometheus/alerts/alert1.yml",
							"/opt/prometheus/rules/rule1.yml",
						},
					},
					"prometheus.repo2": {
						AdditionalConfigsFullLocalPaths: []string{
							"/opt/prometheus/alerts/alert2.yml",
						},
					},
				},
			},
		},
	}

	paths := cs.GetAllConfigLocalPaths("prometheus")
	c.Assert(len(paths), Equals, 4) // 1 primary + 3 additional
	c.Assert(paths[0], Equals, "/opt/prometheus/prometheus.yml")
}

func (s *ConfigTestSuite) TestConfigSettingsGetAllConfigLocalPathsNonExistent(c *C) {
	cs := &ConfigSettings{
		Managers: map[string]*Manager{},
	}

	paths := cs.GetAllConfigLocalPaths("nonexistent")
	c.Assert(len(paths), Equals, 0)
}

func (s *ConfigTestSuite) TestConfigSettingsGetAllConfigLocalPathsNilManagers(c *C) {
	cs := &ConfigSettings{}

	paths := cs.GetAllConfigLocalPaths("prometheus")
	c.Assert(len(paths), Equals, 0)
}

// Helper function to create a simple error for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func NewReloaderErrorForTest(msg string) error {
	return &testError{msg: msg}
}
