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

func (s *ConfigTestSuite) TestReadManagerStatusFileNotExist(c *C) {
	// Test reading a non-existent file
	_, err := ReadManagerStatusFile("/nonexistent/path/status.json")
	c.Assert(err, NotNil)
}

func (s *ConfigTestSuite) TestWriteAndReadManagerStatusFile(c *C) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "butler-status-test-*.json")
	c.Assert(err, IsNil)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create a status object
	status := Status{
		Manager: map[string]bool{
			"prometheus":   true,
			"alertmanager": false,
		},
	}

	// Write the status file
	err = WriteManagerStatusFile(tmpFile.Name(), status)
	c.Assert(err, IsNil)

	// Read it back
	readStatus, err := ReadManagerStatusFile(tmpFile.Name())
	c.Assert(err, IsNil)
	c.Assert(readStatus.Manager["prometheus"], Equals, true)
	c.Assert(readStatus.Manager["alertmanager"], Equals, false)
}

func (s *ConfigTestSuite) TestWriteManagerStatusFileInvalidPath(c *C) {
	status := Status{
		Manager: map[string]bool{"test": true},
	}
	err := WriteManagerStatusFile("/nonexistent/directory/status.json", status)
	c.Assert(err, NotNil)
}

func (s *ConfigTestSuite) TestReadManagerStatusFileInvalidJSON(c *C) {
	// Create a temporary file with invalid JSON
	tmpFile, err := os.CreateTemp("", "butler-status-invalid-*.json")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("this is not valid json"))
	c.Assert(err, IsNil)
	tmpFile.Close()

	// Try to read it
	_, err = ReadManagerStatusFile(tmpFile.Name())
	c.Assert(err, NotNil)
}

func (s *ConfigTestSuite) TestGetManagerStatus(c *C) {
	// Create a temporary status file
	tmpFile, err := os.CreateTemp("", "butler-status-get-*.json")
	c.Assert(err, IsNil)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Write initial status
	status := Status{
		Manager: map[string]bool{
			"prometheus":   true,
			"alertmanager": false,
		},
	}
	err = WriteManagerStatusFile(tmpFile.Name(), status)
	c.Assert(err, IsNil)

	// Test getting existing manager status
	result := GetManagerStatus(tmpFile.Name(), "prometheus")
	c.Assert(result, Equals, true)

	result = GetManagerStatus(tmpFile.Name(), "alertmanager")
	c.Assert(result, Equals, false)

	// Test getting non-existent manager status
	result = GetManagerStatus(tmpFile.Name(), "nonexistent")
	c.Assert(result, Equals, false)

	// Test with non-existent file
	result = GetManagerStatus("/nonexistent/status.json", "prometheus")
	c.Assert(result, Equals, false)
}

func (s *ConfigTestSuite) TestSetManagerStatus(c *C) {
	// Create a temporary status file
	tmpFile, err := os.CreateTemp("", "butler-status-set-*.json")
	c.Assert(err, IsNil)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Set a manager status (file doesn't exist yet with valid content)
	err = SetManagerStatus(tmpFile.Name(), "prometheus", true)
	c.Assert(err, IsNil)

	// Verify it was set
	result := GetManagerStatus(tmpFile.Name(), "prometheus")
	c.Assert(result, Equals, true)

	// Update the status
	err = SetManagerStatus(tmpFile.Name(), "prometheus", false)
	c.Assert(err, IsNil)

	// Verify it was updated
	result = GetManagerStatus(tmpFile.Name(), "prometheus")
	c.Assert(result, Equals, false)

	// Add another manager
	err = SetManagerStatus(tmpFile.Name(), "alertmanager", true)
	c.Assert(err, IsNil)

	// Verify both exist
	result = GetManagerStatus(tmpFile.Name(), "prometheus")
	c.Assert(result, Equals, false)
	result = GetManagerStatus(tmpFile.Name(), "alertmanager")
	c.Assert(result, Equals, true)
}

func (s *ConfigTestSuite) TestSetManagerStatusNewFile(c *C) {
	// Create a path for a new file
	tmpFile, err := os.CreateTemp("", "butler-status-new-*.json")
	c.Assert(err, IsNil)
	tmpFile.Close()
	os.Remove(tmpFile.Name()) // Remove it so SetManagerStatus creates it

	// Set a manager status on a new file
	err = SetManagerStatus(tmpFile.Name(), "newmanager", true)
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile.Name())

	// Verify it was created and set
	result := GetManagerStatus(tmpFile.Name(), "newmanager")
	c.Assert(result, Equals, true)
}
