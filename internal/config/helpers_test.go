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
	"os"

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

func (s *ConfigTestSuite) TestComputeDataHash(c *C) {
	// Test that same data produces same hash
	data1 := []byte("test data for hashing")
	data2 := []byte("test data for hashing")
	hash1 := ComputeDataHash(data1)
	hash2 := ComputeDataHash(data2)
	c.Assert(hash1, Equals, hash2)

	// Test that different data produces different hash
	data3 := []byte("different test data")
	hash3 := ComputeDataHash(data3)
	c.Assert(hash1, Not(Equals), hash3)

	// Test that hash is a valid hex string (64 chars for SHA256)
	c.Assert(len(hash1), Equals, 64)
}

func (s *ConfigTestSuite) TestComputeFileHash(c *C) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "butler-test-hash-*")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile.Name())

	testContent := []byte("test content for file hashing")
	_, err = tmpFile.Write(testContent)
	c.Assert(err, IsNil)
	tmpFile.Close()

	// Compute hash of the file
	hash, err := ComputeFileHash(tmpFile.Name())
	c.Assert(err, IsNil)
	c.Assert(len(hash), Equals, 64)

	// Verify it matches the data hash
	expectedHash := ComputeDataHash(testContent)
	c.Assert(hash, Equals, expectedHash)

	// Test error case - non-existent file
	_, err = ComputeFileHash("/nonexistent/file/path")
	c.Assert(err, NotNil)
}

func (s *ConfigTestSuite) TestCompareHashOnly(c *C) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "butler-test-compare-*")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile.Name())

	testContent := []byte("test content for comparison")
	_, err = tmpFile.Write(testContent)
	c.Assert(err, IsNil)
	tmpFile.Close()

	// Test first run (empty stored hash) - should return changed=true
	changed, newHash, err := CompareHashOnly(tmpFile.Name(), "", "test-manager")
	c.Assert(err, IsNil)
	c.Assert(changed, Equals, true)
	c.Assert(len(newHash), Equals, 64)

	// Test same hash - should return changed=false
	changed, newHash2, err := CompareHashOnly(tmpFile.Name(), newHash, "test-manager")
	c.Assert(err, IsNil)
	c.Assert(changed, Equals, false)
	c.Assert(newHash2, Equals, newHash)

	// Test different hash - should return changed=true
	changed, newHash3, err := CompareHashOnly(tmpFile.Name(), "differenthash", "test-manager")
	c.Assert(err, IsNil)
	c.Assert(changed, Equals, true)
	c.Assert(newHash3, Equals, newHash)

	// Test error case - non-existent file
	_, _, err = CompareHashOnly("/nonexistent/file/path", "", "test-manager")
	c.Assert(err, NotNil)
}

func (s *ConfigTestSuite) TestComparePrimaryConfigHashes(c *C) {
	// Create temporary files for testing
	tmpFile1, err := os.CreateTemp("", "butler-test-primary1-*")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile1.Name())

	tmpFile2, err := os.CreateTemp("", "butler-test-primary2-*")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile2.Name())

	// Write test content
	_, err = tmpFile1.Write([]byte("primary config 1 content"))
	c.Assert(err, IsNil)
	tmpFile1.Close()

	_, err = tmpFile2.Write([]byte("primary config 2 content"))
	c.Assert(err, IsNil)
	tmpFile2.Close()

	// Create a ConfigChanEvent with test data
	chanEvent := NewConfigChanEvent()
	chanEvent.Manager = "test-manager"
	chanEvent.SetSuccess("test-repo", "config1.yml", nil)
	chanEvent.SetSuccess("test-repo", "config2.yml", nil)
	chanEvent.SetTmpFile("test-repo", "config1.yml", tmpFile1.Name())
	chanEvent.SetTmpFile("test-repo", "config2.yml", tmpFile2.Name())

	// Create manager opts
	opts := make(map[string]*ManagerOpts)
	opts["test-manager.test-repo"] = &ManagerOpts{
		PrimaryConfig: []string{"config1.yml", "config2.yml"},
	}

	// Test first run (empty stored hashes) - should return changed=true
	storedHashes := make(map[string]string)
	changed, newHashes := chanEvent.ComparePrimaryConfigHashes(opts, storedHashes)
	c.Assert(changed, Equals, true)
	c.Assert(len(newHashes), Equals, 2)

	// Test same hashes - should return changed=false
	changed, newHashes2 := chanEvent.ComparePrimaryConfigHashes(opts, newHashes)
	c.Assert(changed, Equals, false)
	c.Assert(newHashes2["primary:config1.yml"], Equals, newHashes["primary:config1.yml"])
	c.Assert(newHashes2["primary:config2.yml"], Equals, newHashes["primary:config2.yml"])
}

func (s *ConfigTestSuite) TestCompareAdditionalConfigHashes(c *C) {
	// Create temporary files for testing
	tmpFile1, err := os.CreateTemp("", "butler-test-additional1-*")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile1.Name())

	tmpFile2, err := os.CreateTemp("", "butler-test-additional2-*")
	c.Assert(err, IsNil)
	defer os.Remove(tmpFile2.Name())

	// Write test content
	_, err = tmpFile1.Write([]byte("additional config 1 content"))
	c.Assert(err, IsNil)
	tmpFile1.Close()

	_, err = tmpFile2.Write([]byte("additional config 2 content"))
	c.Assert(err, IsNil)
	tmpFile2.Close()

	// Create a ConfigChanEvent with test data
	chanEvent := NewConfigChanEvent()
	chanEvent.Manager = "test-manager"
	chanEvent.SetSuccess("test-repo", "additional1.yml", nil)
	chanEvent.SetSuccess("test-repo", "additional2.yml", nil)
	chanEvent.SetTmpFile("test-repo", "additional1.yml", tmpFile1.Name())
	chanEvent.SetTmpFile("test-repo", "additional2.yml", tmpFile2.Name())

	// Test first run (empty stored hashes) - should return changed=true
	storedHashes := make(map[string]string)
	changed, newHashes := chanEvent.CompareAdditionalConfigHashes(storedHashes)
	c.Assert(changed, Equals, true)
	c.Assert(len(newHashes), Equals, 2)

	// Test same hashes - should return changed=false
	changed, newHashes2 := chanEvent.CompareAdditionalConfigHashes(newHashes)
	c.Assert(changed, Equals, false)
	c.Assert(newHashes2["additional:additional1.yml"], Equals, newHashes["additional:additional1.yml"])
	c.Assert(newHashes2["additional:additional2.yml"], Equals, newHashes["additional:additional2.yml"])

	// Test with modified file - should return changed=true
	// Reopen and modify one file
	tmpFile1Modified, err := os.OpenFile(tmpFile1.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	c.Assert(err, IsNil)
	_, err = tmpFile1Modified.Write([]byte("modified additional config 1 content"))
	c.Assert(err, IsNil)
	tmpFile1Modified.Close()

	changed, newHashes3 := chanEvent.CompareAdditionalConfigHashes(newHashes2)
	c.Assert(changed, Equals, true)
	// The modified file should have a different hash
	c.Assert(newHashes3["additional:additional1.yml"], Not(Equals), newHashes2["additional:additional1.yml"])
	// The unmodified file should have the same hash
	c.Assert(newHashes3["additional:additional2.yml"], Equals, newHashes2["additional:additional2.yml"])
}
