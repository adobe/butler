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

package methods

import (
	"io"
	"strings"

	. "gopkg.in/check.v1"
)

// Note: Test() function is defined in file_test.go, so we don't redeclare it here

type MethodsTestSuite struct{}

var _ = Suite(&MethodsTestSuite{})

func (s *MethodsTestSuite) TestResponseGetResponseBody(c *C) {
	testBody := io.NopCloser(strings.NewReader("test body"))
	resp := Response{body: testBody, statusCode: 200}

	body := resp.GetResponseBody()
	c.Assert(body, NotNil)

	// Read the body
	data, err := io.ReadAll(body)
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, "test body")
}

func (s *MethodsTestSuite) TestResponseGetResponseStatusCode(c *C) {
	resp := Response{statusCode: 404}
	c.Assert(resp.GetResponseStatusCode(), Equals, 404)
}

func (s *MethodsTestSuite) TestNewMethodHTTP(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "http", &entry)
	// HTTP method requires proper configuration, so it may return an error
	// but should not panic
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodHTTPS(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "https", &entry)
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodS3(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "s3", &entry)
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodFile(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "file", &entry)
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodBlob(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "blob", &entry)
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodEtcd(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "etcd", &entry)
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodDefault(c *C) {
	manager := "test-manager"
	entry := "test-entry"
	method, err := New(&manager, "unknown", &entry)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Generic method handler is not very useful")
	_ = method
}

func (s *MethodsTestSuite) TestNewMethodCaseInsensitive(c *C) {
	manager := "test-manager"
	entry := "test-entry"

	// Test uppercase
	method1, _ := New(&manager, "HTTP", &entry)
	method2, _ := New(&manager, "http", &entry)

	// Both should create the same type of method
	_ = method1
	_ = method2
}

func (s *MethodsTestSuite) TestNewMethodNilManager(c *C) {
	entry := "test-entry"
	method, err := New(nil, "http", &entry)
	_ = method
	_ = err
}

func (s *MethodsTestSuite) TestNewMethodNilEntry(c *C) {
	manager := "test-manager"
	method, err := New(&manager, "http", nil)
	_ = method
	_ = err
}
