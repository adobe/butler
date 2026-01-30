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

	. "gopkg.in/check.v1"
)

func (s *ConfigTestSuite) TestNewConfigChanEvent(c *C) {
	event := NewConfigChanEvent()
	c.Assert(event, NotNil)
	c.Assert(event.Repo, NotNil)
	c.Assert(event.HasChanged, Equals, false)
}

func (s *ConfigTestSuite) TestConfigChanEventCanCopyFilesAllSuccess(c *C) {
	event := NewConfigChanEvent()
	event.SetSuccess("repo1", "file1.yml", nil)
	event.SetSuccess("repo1", "file2.yml", nil)
	event.SetSuccess("repo2", "file3.yml", nil)

	c.Assert(event.CanCopyFiles(), Equals, true)
}

func (s *ConfigTestSuite) TestConfigChanEventCanCopyFilesWithFailure(c *C) {
	event := NewConfigChanEvent()
	event.SetSuccess("repo1", "file1.yml", nil)
	event.SetFailure("repo1", "file2.yml", nil)

	c.Assert(event.CanCopyFiles(), Equals, false)
}

func (s *ConfigTestSuite) TestConfigChanEventCanCopyFilesEmpty(c *C) {
	event := NewConfigChanEvent()
	c.Assert(event.CanCopyFiles(), Equals, true)
}

func (s *ConfigTestSuite) TestConfigChanEventSetSuccessInitializesRepo(c *C) {
	event := &ConfigChanEvent{}
	c.Assert(event.Repo, IsNil)

	event.SetSuccess("newrepo", "file.yml", nil)
	c.Assert(event.Repo, NotNil)
	c.Assert(event.Repo["newrepo"], NotNil)
	c.Assert(event.Repo["newrepo"].Success["file.yml"], Equals, true)
}

func (s *ConfigTestSuite) TestConfigChanEventSetFailureInitializesRepo(c *C) {
	event := &ConfigChanEvent{}
	c.Assert(event.Repo, IsNil)

	event.SetFailure("newrepo", "file.yml", nil)
	c.Assert(event.Repo, NotNil)
	c.Assert(event.Repo["newrepo"], NotNil)
	c.Assert(event.Repo["newrepo"].Success["file.yml"], Equals, false)
}

func (s *ConfigTestSuite) TestConfigChanEventSetTmpFile(c *C) {
	event := NewConfigChanEvent()
	event.SetSuccess("repo1", "file.yml", nil)
	event.SetTmpFile("repo1", "file.yml", "/tmp/butler-12345")

	c.Assert(event.Repo["repo1"].TmpFile["file.yml"], Equals, "/tmp/butler-12345")
}

func (s *ConfigTestSuite) TestConfigChanEventSetTmpFileNoRepo(c *C) {
	event := NewConfigChanEvent()
	// Setting tmp file for non-existent repo should not panic
	err := event.SetTmpFile("nonexistent", "file.yml", "/tmp/butler-12345")
	c.Assert(err, IsNil)
}

func (s *ConfigTestSuite) TestConfigChanEventGetTmpFileMap(c *C) {
	event := NewConfigChanEvent()
	event.SetSuccess("repo1", "b_file.yml", nil)
	event.SetSuccess("repo1", "a_file.yml", nil)
	event.SetTmpFile("repo1", "b_file.yml", "/tmp/butler-b")
	event.SetTmpFile("repo1", "a_file.yml", "/tmp/butler-a")

	tmpFiles := event.GetTmpFileMap()
	c.Assert(len(tmpFiles), Equals, 2)
	// Should be sorted alphabetically
	c.Assert(tmpFiles[0].Name, Equals, "a_file.yml")
	c.Assert(tmpFiles[0].File, Equals, "/tmp/butler-a")
	c.Assert(tmpFiles[1].Name, Equals, "b_file.yml")
	c.Assert(tmpFiles[1].File, Equals, "/tmp/butler-b")
}

func (s *ConfigTestSuite) TestConfigChanEventCleanTmpFiles(c *C) {
	// Create actual temp files
	tmpFile1, err := os.CreateTemp("", "butler-test-clean1-*")
	c.Assert(err, IsNil)
	tmpFile1.Close()

	tmpFile2, err := os.CreateTemp("", "butler-test-clean2-*")
	c.Assert(err, IsNil)
	tmpFile2.Close()

	// Verify files exist
	_, err = os.Stat(tmpFile1.Name())
	c.Assert(err, IsNil)
	_, err = os.Stat(tmpFile2.Name())
	c.Assert(err, IsNil)

	event := NewConfigChanEvent()
	event.SetSuccess("repo1", "file1.yml", nil)
	event.SetSuccess("repo1", "file2.yml", nil)
	event.SetTmpFile("repo1", "file1.yml", tmpFile1.Name())
	event.SetTmpFile("repo1", "file2.yml", tmpFile2.Name())

	// Clean up
	err = event.CleanTmpFiles()
	c.Assert(err, IsNil)

	// Verify files are deleted
	_, err = os.Stat(tmpFile1.Name())
	c.Assert(os.IsNotExist(err), Equals, true)
	_, err = os.Stat(tmpFile2.Name())
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *ConfigTestSuite) TestConfigChanEventCleanTmpFilesWithMainTmpFile(c *C) {
	// Create a main temp file
	mainTmpFile, err := os.CreateTemp("", "butler-test-main-*")
	c.Assert(err, IsNil)
	mainTmpFile.Close()

	event := NewConfigChanEvent()
	event.TmpFile = mainTmpFile

	// Clean up
	err = event.CleanTmpFiles()
	c.Assert(err, IsNil)

	// Verify main file is deleted
	_, err = os.Stat(mainTmpFile.Name())
	c.Assert(os.IsNotExist(err), Equals, true)
}

func (s *ConfigTestSuite) TestConfigChanEventMultipleRepos(c *C) {
	event := NewConfigChanEvent()

	// Add files from multiple repos
	event.SetSuccess("repo1", "file1.yml", nil)
	event.SetSuccess("repo2", "file2.yml", nil)
	event.SetSuccess("repo3", "file3.yml", nil)

	c.Assert(len(event.Repo), Equals, 3)
	c.Assert(event.Repo["repo1"].Success["file1.yml"], Equals, true)
	c.Assert(event.Repo["repo2"].Success["file2.yml"], Equals, true)
	c.Assert(event.Repo["repo3"].Success["file3.yml"], Equals, true)
}
