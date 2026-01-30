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
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

func (s *ConfigTestSuite) TestManagerOptsAppendPrimaryConfigURL(c *C) {
	opts := &ManagerOpts{}
	err := opts.AppendPrimaryConfigURL("http://example.com/config.yml")
	c.Assert(err, IsNil)
	c.Assert(len(opts.PrimaryConfigsFullURLs), Equals, 1)
	c.Assert(opts.PrimaryConfigsFullURLs[0], Equals, "http://example.com/config.yml")

	// Append another
	err = opts.AppendPrimaryConfigURL("http://example.com/config2.yml")
	c.Assert(err, IsNil)
	c.Assert(len(opts.PrimaryConfigsFullURLs), Equals, 2)
}

func (s *ConfigTestSuite) TestManagerOptsAppendPrimaryConfigFile(c *C) {
	opts := &ManagerOpts{}
	err := opts.AppendPrimaryConfigFile("/opt/prometheus/prometheus.yml")
	c.Assert(err, IsNil)
	c.Assert(len(opts.PrimaryConfigsFullLocalPaths), Equals, 1)
	c.Assert(opts.PrimaryConfigsFullLocalPaths[0], Equals, "/opt/prometheus/prometheus.yml")
}

func (s *ConfigTestSuite) TestManagerOptsAppendAdditionalConfigURL(c *C) {
	opts := &ManagerOpts{}
	err := opts.AppendAdditionalConfigURL("http://example.com/alerts.yml")
	c.Assert(err, IsNil)
	c.Assert(len(opts.AdditionalConfigsFullURLs), Equals, 1)
	c.Assert(opts.AdditionalConfigsFullURLs[0], Equals, "http://example.com/alerts.yml")
}

func (s *ConfigTestSuite) TestManagerOptsAppendAdditionalConfigFile(c *C) {
	opts := &ManagerOpts{}
	err := opts.AppendAdditionalConfigFile("/opt/prometheus/alerts/alert1.yml")
	c.Assert(err, IsNil)
	c.Assert(len(opts.AdditionalConfigsFullLocalPaths), Equals, 1)
	c.Assert(opts.AdditionalConfigsFullLocalPaths[0], Equals, "/opt/prometheus/alerts/alert1.yml")
}

func (s *ConfigTestSuite) TestManagerOptsSetParentManager(c *C) {
	opts := &ManagerOpts{}
	err := opts.SetParentManager("prometheus")
	c.Assert(err, IsNil)
	c.Assert(opts.parentManager, Equals, "prometheus")
}

func (s *ConfigTestSuite) TestManagerOptsGetPrimaryConfigURLs(c *C) {
	opts := &ManagerOpts{
		PrimaryConfigsFullURLs: []string{"http://a.com/1.yml", "http://b.com/2.yml"},
	}
	urls := opts.GetPrimaryConfigURLs()
	c.Assert(len(urls), Equals, 2)
	c.Assert(urls[0], Equals, "http://a.com/1.yml")
}

func (s *ConfigTestSuite) TestManagerOptsGetPrimaryLocalConfigFiles(c *C) {
	opts := &ManagerOpts{
		PrimaryConfigsFullLocalPaths: []string{"/opt/a.yml", "/opt/b.yml"},
	}
	files := opts.GetPrimaryLocalConfigFiles()
	c.Assert(len(files), Equals, 2)
	c.Assert(files[0], Equals, "/opt/a.yml")
}

func (s *ConfigTestSuite) TestManagerOptsGetPrimaryRemoteConfigFiles(c *C) {
	opts := &ManagerOpts{
		PrimaryConfig: []string{"config1.yml", "config2.yml"},
	}
	files := opts.GetPrimaryRemoteConfigFiles()
	c.Assert(len(files), Equals, 2)
	c.Assert(files[0], Equals, "config1.yml")
}

func (s *ConfigTestSuite) TestManagerOptsGetAdditionalConfigURLs(c *C) {
	opts := &ManagerOpts{
		AdditionalConfigsFullURLs: []string{"http://a.com/alerts.yml"},
	}
	urls := opts.GetAdditionalConfigURLs()
	c.Assert(len(urls), Equals, 1)
}

func (s *ConfigTestSuite) TestManagerOptsGetAdditionalLocalConfigFiles(c *C) {
	opts := &ManagerOpts{
		AdditionalConfigsFullLocalPaths: []string{"/opt/alerts/a.yml"},
	}
	files := opts.GetAdditionalLocalConfigFiles()
	c.Assert(len(files), Equals, 1)
}

func (s *ConfigTestSuite) TestManagerOptsGetAdditionalRemoteConfigFiles(c *C) {
	opts := &ManagerOpts{
		AdditionalConfig: []string{"alerts/alert1.yml", "rules/rule1.yml"},
	}
	files := opts.GetAdditionalRemoteConfigFiles()
	c.Assert(len(files), Equals, 2)
}

func (s *ConfigTestSuite) TestManagerGetAllLocalPaths(c *C) {
	mgr := &Manager{
		ManagerOpts: map[string]*ManagerOpts{
			"mgr.repo1": {
				PrimaryConfigsFullLocalPaths:    []string{"/opt/primary1.yml"},
				AdditionalConfigsFullLocalPaths: []string{"/opt/add1.yml", "/opt/add2.yml"},
			},
			"mgr.repo2": {
				PrimaryConfigsFullLocalPaths:    []string{"/opt/primary2.yml"},
				AdditionalConfigsFullLocalPaths: []string{"/opt/add3.yml"},
			},
		},
	}

	paths := mgr.GetAllLocalPaths()
	c.Assert(len(paths), Equals, 5)
}

func (s *ConfigTestSuite) TestManagerGetAllLocalPathsEmpty(c *C) {
	mgr := &Manager{
		ManagerOpts: map[string]*ManagerOpts{},
	}

	paths := mgr.GetAllLocalPaths()
	c.Assert(len(paths), Equals, 0)
}

func (s *ConfigTestSuite) TestManagerPathCleanupDirectory(c *C) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "butler-test-cleanup-*")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpDir)

	mgr := &Manager{
		ManagerOpts: map[string]*ManagerOpts{},
	}

	// Test with a directory - should return nil
	dirInfo, err := os.Stat(tmpDir)
	c.Assert(err, IsNil)

	err = mgr.PathCleanup(tmpDir, dirInfo, nil)
	c.Assert(err, IsNil)
}

func (s *ConfigTestSuite) TestManagerPathCleanupKnownFile(c *C) {
	// Create a temporary directory and file
	tmpDir, err := os.MkdirTemp("", "butler-test-cleanup-*")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpDir)

	knownFile := filepath.Join(tmpDir, "known.yml")
	err = os.WriteFile(knownFile, []byte("test"), 0644)
	c.Assert(err, IsNil)

	mgr := &Manager{
		ManagerOpts: map[string]*ManagerOpts{
			"mgr.repo1": {
				PrimaryConfigsFullLocalPaths:    []string{knownFile},
				AdditionalConfigsFullLocalPaths: []string{},
			},
		},
	}

	fileInfo, err := os.Stat(knownFile)
	c.Assert(err, IsNil)

	// Known file should not be deleted
	err = mgr.PathCleanup(knownFile, fileInfo, nil)
	c.Assert(err, IsNil)

	// Verify file still exists
	_, err = os.Stat(knownFile)
	c.Assert(err, IsNil)
}

func (s *ConfigTestSuite) TestManagerPathCleanupUnknownFile(c *C) {
	// Create a temporary directory and file
	tmpDir, err := os.MkdirTemp("", "butler-test-cleanup-*")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpDir)

	unknownFile := filepath.Join(tmpDir, "unknown.yml")
	err = os.WriteFile(unknownFile, []byte("test"), 0644)
	c.Assert(err, IsNil)

	mgr := &Manager{
		ManagerOpts: map[string]*ManagerOpts{
			"mgr.repo1": {
				PrimaryConfigsFullLocalPaths:    []string{filepath.Join(tmpDir, "known.yml")},
				AdditionalConfigsFullLocalPaths: []string{},
			},
		},
	}

	fileInfo, err := os.Stat(unknownFile)
	c.Assert(err, IsNil)

	// Unknown file should be deleted
	err = mgr.PathCleanup(unknownFile, fileInfo, nil)
	c.Assert(err, NotNil) // Returns error with message about deletion

	// Verify file was deleted
	_, err = os.Stat(unknownFile)
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *ConfigTestSuite) TestManagerReloadNoReloader(c *C) {
	mgr := &Manager{
		Name:     "test-manager",
		Reloader: nil,
	}

	// Should return nil when no reloader is defined
	err := mgr.Reload()
	c.Assert(err, IsNil)
}
