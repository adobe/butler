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

package reloaders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "gopkg.in/check.v1"
)

func (s *ReloaderTestSuite) TestNewHTTPReloader(c *C) {
	opts := HTTPReloaderOpts{
		Host:        "localhost",
		Port:        "9090",
		URI:         "/-/reload",
		Method:      "post",
		ContentType: "application/json",
		Payload:     "{}",
		Timeout:     "10",
		Retries:     "3",
		RetryWaitMin: "1",
		RetryWaitMax: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewHTTPReloader("test-manager", "http", jsonOpts)
	c.Assert(err, IsNil)

	httpReloader := reloader.(HTTPReloader)
	c.Assert(httpReloader.Method, Equals, "http")
	c.Assert(httpReloader.Manager, Equals, "test-manager")
	c.Assert(httpReloader.Opts.Host, Equals, "localhost")
	c.Assert(httpReloader.Opts.URI, Equals, "/-/reload")
}

func (s *ReloaderTestSuite) TestNewHTTPReloaderInvalidJSON(c *C) {
	_, err := NewHTTPReloader("test-manager", "http", []byte("invalid json"))
	c.Assert(err, NotNil)
}

func (s *ReloaderTestSuite) TestHTTPReloaderGetMethod(c *C) {
	reloader := HTTPReloader{Method: "https"}
	c.Assert(reloader.GetMethod(), Equals, "https")
}

func (s *ReloaderTestSuite) TestHTTPReloaderGetOpts(c *C) {
	opts := HTTPReloaderOpts{Host: "testhost"}
	reloader := HTTPReloader{Opts: opts}
	result := reloader.GetOpts().(HTTPReloaderOpts)
	c.Assert(result.Host, Equals, "testhost")
}

func (s *ReloaderTestSuite) TestHTTPReloaderSetOpts(c *C) {
	reloader := HTTPReloader{}
	newOpts := HTTPReloaderOpts{Host: "newhost"}
	result := reloader.SetOpts(newOpts)
	c.Assert(result, Equals, true)
}

func (s *ReloaderTestSuite) TestHTTPReloaderSetCounter(c *C) {
	reloader := HTTPReloader{Counter: 0}
	result := reloader.SetCounter(5)
	httpResult := result.(HTTPReloader)
	c.Assert(httpResult.Counter, Equals, 5)
}

func (s *ReloaderTestSuite) TestHTTPReloaderOptsGetClient(c *C) {
	opts := HTTPReloaderOpts{}
	c.Assert(opts.GetClient(), IsNil)
}

func (s *ReloaderTestSuite) TestHTTPReloaderReloadSuccess(c *C) {
	// Create a test server that returns 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := HTTPReloaderOpts{
		Host:        "127.0.0.1",
		Port:        fmt.Sprintf("%d", getPortFromURL(server.URL)),
		URI:         "/",
		Method:      "post",
		ContentType: "application/json",
		Payload:     "{}",
		Timeout:     "10",
		Retries:     "1",
		RetryWaitMin: "1",
		RetryWaitMax: "2",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewHTTPReloader("test-manager", "http", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, IsNil)
}

func (s *ReloaderTestSuite) TestHTTPReloaderReloadBadResponse(c *C) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	opts := HTTPReloaderOpts{
		Host:        "127.0.0.1",
		Port:        fmt.Sprintf("%d", getPortFromURL(server.URL)),
		URI:         "/",
		Method:      "get",
		Timeout:     "5",
		Retries:     "0",
		RetryWaitMin: "1",
		RetryWaitMax: "2",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewHTTPReloader("test-manager", "http", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, NotNil)
	reloaderErr := err.(*ReloaderError)
	c.Assert(reloaderErr.Code, Equals, 500)
}

func (s *ReloaderTestSuite) TestHTTPReloaderReloadMethods(c *C) {
	// Test different HTTP methods
	methods := []string{"post", "put", "patch", "get"}

	for _, method := range methods {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		opts := HTTPReloaderOpts{
			Host:        "127.0.0.1",
			Port:        fmt.Sprintf("%d", getPortFromURL(server.URL)),
			URI:         "/",
			Method:      method,
			ContentType: "application/json",
			Payload:     "{}",
			Timeout:     "5",
			Retries:     "0",
			RetryWaitMin: "1",
			RetryWaitMax: "2",
		}

		jsonOpts, _ := json.Marshal(opts)
		reloader, _ := NewHTTPReloader("test-manager", "http", jsonOpts)
		err := reloader.Reload()
		c.Assert(err, IsNil, Commentf("Method %s failed", method))

		server.Close()
	}
}

func (s *ReloaderTestSuite) TestHTTPReloaderRetryPolicy(c *C) {
	reloader := HTTPReloader{Manager: "test-manager"}

	// Test with error
	shouldRetry, err := reloader.ReloaderRetryPolicy(context.Background(), nil, fmt.Errorf("test error"))
	c.Assert(shouldRetry, Equals, true)
	c.Assert(err, NotNil)

	// Test with status code 0
	resp := &http.Response{StatusCode: 0}
	shouldRetry, err = reloader.ReloaderRetryPolicy(context.Background(), resp, nil)
	c.Assert(shouldRetry, Equals, true)
	c.Assert(err, IsNil)

	// Test with status code >= 600
	resp = &http.Response{StatusCode: 600}
	shouldRetry, err = reloader.ReloaderRetryPolicy(context.Background(), resp, nil)
	c.Assert(shouldRetry, Equals, true)
	c.Assert(err, IsNil)

	// Test with normal status code (should not retry)
	resp = &http.Response{StatusCode: 200}
	shouldRetry, err = reloader.ReloaderRetryPolicy(context.Background(), resp, nil)
	c.Assert(shouldRetry, Equals, false)
	c.Assert(err, IsNil)

	// Test with 500 status code (should not retry by our policy)
	resp = &http.Response{StatusCode: 500}
	shouldRetry, err = reloader.ReloaderRetryPolicy(context.Background(), resp, nil)
	c.Assert(shouldRetry, Equals, false)
	c.Assert(err, IsNil)
}

// Helper function to extract port from URL
func getPortFromURL(urlStr string) int {
	// Parse URL like "http://127.0.0.1:12345"
	var port int
	fmt.Sscanf(urlStr, "http://127.0.0.1:%d", &port)
	return port
}
