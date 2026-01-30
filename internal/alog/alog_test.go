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

package alog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/adobe/butler/internal/config"
	"github.com/adobe/butler/internal/methods"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type AlogTestSuite struct{}

var _ = Suite(&AlogTestSuite{})

func (s *AlogTestSuite) TestApacheLogRecordWrite(c *C) {
	// Create a response recorder
	recorder := httptest.NewRecorder()

	// Create an ApacheLogRecord
	record := &ApacheLogRecord{
		ResponseWriter: recorder,
		log:            false,
		ip:             "127.0.0.1",
		time:           time.Now(),
		method:         "GET",
		uri:            "/test",
		protocol:       "HTTP/1.1",
		status:         http.StatusOK,
		responseBytes:  0,
	}

	// Write some data
	testData := []byte("Hello, World!")
	n, err := record.Write(testData)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, len(testData))
	c.Assert(record.responseBytes, Equals, int64(len(testData)))
}

func (s *AlogTestSuite) TestApacheLogRecordWriteHeader(c *C) {
	recorder := httptest.NewRecorder()

	record := &ApacheLogRecord{
		ResponseWriter: recorder,
		status:         http.StatusOK,
	}

	record.WriteHeader(http.StatusNotFound)
	c.Assert(record.status, Equals, http.StatusNotFound)
	c.Assert(recorder.Code, Equals, http.StatusNotFound)
}

func (s *AlogTestSuite) TestApacheLogRecordLog(c *C) {
	recorder := httptest.NewRecorder()

	// Test with logging disabled
	record := &ApacheLogRecord{
		ResponseWriter: recorder,
		log:            false,
		ip:             "127.0.0.1",
		time:           time.Now(),
		method:         "GET",
		uri:            "/test",
		protocol:       "HTTP/1.1",
		status:         http.StatusOK,
		responseBytes:  100,
		elapsedTime:    time.Millisecond * 50,
	}

	// This should not panic even with logging disabled
	record.Log()
}

func (s *AlogTestSuite) TestApacheLogRecordLogEnabled(c *C) {
	recorder := httptest.NewRecorder()

	// Test with logging enabled
	record := &ApacheLogRecord{
		ResponseWriter: recorder,
		log:            true,
		ip:             "192.168.1.1",
		time:           time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		method:         "POST",
		uri:            "/api/reload",
		protocol:       "HTTP/1.1",
		status:         http.StatusOK,
		responseBytes:  256,
		elapsedTime:    time.Millisecond * 100,
	}

	// This should log without panicking
	record.Log()
}

func (s *AlogTestSuite) TestNewApacheLoggingHandler(c *C) {
	// Create a simple handler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create butler config
	u, _ := url.Parse("http://localhost")
	opts := &config.ButlerConfigOpts{
		InsecureSkipVerify: false,
		URL:                u,
	}
	bc, _ := config.NewButlerConfig(opts)
	bc.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u.Scheme})
	bc.Config = config.NewConfigSettings()
	bc.Config.Globals.EnableHTTPLog = true

	// Create the logging handler
	loggingHandler := NewApacheLoggingHandler(innerHandler, bc)
	c.Assert(loggingHandler, NotNil)
}

func (s *AlogTestSuite) TestApacheLoggingHandlerServeHTTP(c *C) {
	// Create a simple handler that writes a response
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	})

	// Create butler config
	u, _ := url.Parse("http://localhost")
	opts := &config.ButlerConfigOpts{
		InsecureSkipVerify: false,
		URL:                u,
	}
	bc, _ := config.NewButlerConfig(opts)
	bc.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u.Scheme})
	bc.Config = config.NewConfigSettings()
	bc.Config.Globals.EnableHTTPLog = false

	// Create the logging handler
	loggingHandler := NewApacheLoggingHandler(innerHandler, bc)

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Create a response recorder
	recorder := httptest.NewRecorder()

	// Serve the request
	loggingHandler.ServeHTTP(recorder, req)

	// Verify the response
	c.Assert(recorder.Code, Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), Equals, "Hello")
}

func (s *AlogTestSuite) TestApacheLoggingHandlerServeHTTPWithLogging(c *C) {
	// Create a simple handler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Created"))
	})

	// Create butler config with logging enabled
	u, _ := url.Parse("http://localhost")
	opts := &config.ButlerConfigOpts{
		InsecureSkipVerify: false,
		URL:                u,
	}
	bc, _ := config.NewButlerConfig(opts)
	bc.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u.Scheme})
	bc.Config = config.NewConfigSettings()
	bc.Config.Globals.EnableHTTPLog = true

	loggingHandler := NewApacheLoggingHandler(innerHandler, bc)

	// Test with IPv6 address
	req := httptest.NewRequest("POST", "/api/create", bytes.NewReader([]byte("{}")))
	req.RemoteAddr = "[::1]:54321"

	recorder := httptest.NewRecorder()
	loggingHandler.ServeHTTP(recorder, req)

	c.Assert(recorder.Code, Equals, http.StatusCreated)
}

func (s *AlogTestSuite) TestApacheLoggingHandlerClientIPExtraction(c *C) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	u, _ := url.Parse("http://localhost")
	opts := &config.ButlerConfigOpts{URL: u}
	bc, _ := config.NewButlerConfig(opts)
	bc.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u.Scheme})
	bc.Config = config.NewConfigSettings()
	bc.Config.Globals.EnableHTTPLog = false

	loggingHandler := NewApacheLoggingHandler(innerHandler, bc)

	// Test with standard IP:port format
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:8080"

	recorder := httptest.NewRecorder()
	loggingHandler.ServeHTTP(recorder, req)
	c.Assert(recorder.Code, Equals, http.StatusOK)
}

func (s *AlogTestSuite) TestApacheFormatPattern(c *C) {
	// Verify the format pattern is correct
	c.Assert(ApacheFormatPattern, Equals, "%s - - [%s] \"%s %d %d\" %f\n")
}
