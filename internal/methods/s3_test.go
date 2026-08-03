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
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	. "gopkg.in/check.v1"
)

var _ = Suite(&S3TestSuite{})

type S3TestSuite struct {
}

var TestViperConfigS3 = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "s3"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.s3]
      bucket = "test-bucket"
      region = "us-east-1"
      retries = "7"
`)

var TestViperConfigS3NoRetries = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "s3"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.s3]
      bucket = "test-bucket"
      region = "us-east-1"
`)

var TestViperConfigS3NoBucket = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "s3"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.s3]
      region = "us-east-1"
`)

func (s *S3TestSuite) SetUpSuite(c *C) {
	viper.SetConfigType("toml")
}

func (s *S3TestSuite) TearDownSuite(c *C) {
}

// newTestS3Method builds an S3Method whose Downloader talks to a local
// httptest.Server instead of real AWS, so Get() can be exercised without
// network access. maxRetries mirrors what NewS3Method/NewS3MethodWithOpts
// would set on the session from the configured retries value.
func newTestS3Method(endpoint string, maxRetries int) S3Method {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(true),
		MaxRetries:       aws.Int(maxRetries),
		Credentials:      credentials.NewStaticCredentials("id", "secret", ""),
		DisableSSL:       aws.Bool(true),
	}))
	return S3Method{
		Bucket:     "test-bucket",
		Region:     "us-east-1",
		Retries:    fmt.Sprintf("%v", maxRetries),
		Downloader: s3manager.NewDownloader(sess),
	}
}

func (s *S3TestSuite) TestNewS3MethodRetriesFromConfig(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigS3))
	c.Assert(err, IsNil)

	manager := "test-manager"
	entry := "test-manager.repo.s3"

	method, err := NewS3Method(&manager, &entry)
	c.Assert(err, IsNil)
	m := method.(S3Method)
	c.Assert(m.Retries, Equals, "7")
	c.Assert(m.Bucket, Equals, "test-bucket")
	c.Assert(m.Region, Equals, "us-east-1")
	c.Assert(m.Downloader, NotNil)
}

func (s *S3TestSuite) TestNewS3MethodRetriesDefaultsWhenUnset(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigS3NoRetries))
	c.Assert(err, IsNil)

	manager := "test-manager"
	entry := "test-manager.repo.s3"

	method, err := NewS3Method(&manager, &entry)
	c.Assert(err, IsNil)
	m := method.(S3Method)
	c.Assert(m.Retries, Equals, "")
}

func (s *S3TestSuite) TestNewS3MethodMissingBucketOrRegion(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigS3NoBucket))
	c.Assert(err, IsNil)

	manager := "test-manager"
	entry := "test-manager.repo.s3"

	_, err = NewS3Method(&manager, &entry)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "s3 bucket or region is not defined in config")
}

func (s *S3TestSuite) TestNewS3MethodWithOptsRetries(c *C) {
	opts := S3MethodOpts{
		AccessKeyID:     "id",
		SecretAccessKey: "secret",
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		Retries:         9,
	}
	method, err := NewS3MethodWithOpts(opts)
	c.Assert(err, IsNil)
	m := method.(S3Method)
	c.Assert(m.Retries, Equals, "9")
	c.Assert(m.Bucket, Equals, "test-bucket")
}

func (s *S3TestSuite) TestNewS3MethodWithOptsRetriesDefault(c *C) {
	opts := S3MethodOpts{
		AccessKeyID:     "id",
		SecretAccessKey: "secret",
		Bucket:          "test-bucket",
		Region:          "us-east-1",
	}
	method, err := NewS3MethodWithOpts(opts)
	c.Assert(err, IsNil)
	m := method.(S3Method)
	c.Assert(m.Retries, Equals, fmt.Sprintf("%v", defaultS3Retries))
}

// TestGetSucceedsFirstTry verifies the happy path still works: a single
// successful GetObject call downloads and returns the body unchanged.
func (s *S3TestSuite) TestGetSucceedsFirstTry(c *C) {
	var reqCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hiya"))
	}))
	defer ts.Close()

	method := newTestS3Method(ts.URL, 3)
	u, err := url.Parse("/foo/bar")
	c.Assert(err, IsNil)

	resp, err := method.Get(u)
	c.Assert(err, IsNil)
	c.Assert(resp.GetResponseStatusCode(), Equals, 200)
	out, err := ioutil.ReadAll(resp.GetResponseBody())
	c.Assert(err, IsNil)
	c.Assert(string(out), Equals, "hiya")
	c.Assert(atomic.LoadInt32(&reqCount), Equals, int32(1))
}

// TestGetRetriesOnTransientFailureThenSucceeds is the core retry-behavior
// test for issue #4: the fake S3 endpoint returns 500 twice, then 200. With
// MaxRetries=3 the SDK's default retryer should retry past the transient
// failures and Get() should return the eventual successful body.
func (s *S3TestSuite) TestGetRetriesOnTransientFailureThenSucceeds(c *C) {
	var reqCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&reqCount, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("<Error><Code>InternalError</Code></Error>"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("recovered"))
	}))
	defer ts.Close()

	method := newTestS3Method(ts.URL, 3)
	u, err := url.Parse("/foo/bar")
	c.Assert(err, IsNil)

	resp, err := method.Get(u)
	c.Assert(err, IsNil)
	c.Assert(resp.GetResponseStatusCode(), Equals, 200)
	out, err := ioutil.ReadAll(resp.GetResponseBody())
	c.Assert(err, IsNil)
	c.Assert(string(out), Equals, "recovered")
	// 2 failures + 1 success = 3 requests total, proving the SDK retried
	// past the transient 500s using the MaxRetries we configured.
	c.Assert(atomic.LoadInt32(&reqCount), Equals, int32(3))
}

// TestGetExhaustsRetriesAndFails configures MaxRetries=1 (2 attempts total)
// against an endpoint that always 500s, so Get() should still fail overall,
// but only after making the expected number of attempts.
func (s *S3TestSuite) TestGetExhaustsRetriesAndFails(c *C) {
	var reqCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<Error><Code>InternalError</Code></Error>"))
	}))
	defer ts.Close()

	method := newTestS3Method(ts.URL, 1)
	u, err := url.Parse("/foo/bar")
	c.Assert(err, IsNil)

	resp, err := method.Get(u)
	c.Assert(err, NotNil)
	c.Assert(resp.GetResponseStatusCode(), Equals, 500)
	// MaxRetries=1 means 1 initial attempt + 1 retry = 2 requests.
	c.Assert(atomic.LoadInt32(&reqCount), Equals, int32(2))
}

// TestGetDoesNotRetryOnNotFound ensures a permanent 404 (no such key) is
// not retried at all -- only 500-range/transient errors should trigger
// retries, matching S3's own retry semantics.
func (s *S3TestSuite) TestGetDoesNotRetryOnNotFound(c *C) {
	var reqCount int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<Error><Code>NoSuchKey</Code></Error>`))
	}))
	defer ts.Close()

	method := newTestS3Method(ts.URL, 3)
	u, err := url.Parse("/foo/bar")
	c.Assert(err, IsNil)

	resp, err := method.Get(u)
	c.Assert(err, NotNil)
	c.Assert(resp.GetResponseStatusCode(), Equals, 404)
	c.Assert(atomic.LoadInt32(&reqCount), Equals, int32(1))
}

// TestGetNetworkFailureFallsBackTo504 covers the case where the request
// never reaches an S3-like server at all (connection refused / DNS
// failure) -- this is a pure network-level awserr.Error, not an
// awserr.RequestFailure, so Get() should fall back to the hardcoded 504
// rather than a real HTTP status (there isn't one).
func (s *S3TestSuite) TestGetNetworkFailureFallsBackTo504(c *C) {
	// Port 0 combined with a closed listener guarantees connection refused
	// without depending on any real network access.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	c.Assert(err, IsNil)
	closedAddr := ln.Addr().String()
	ln.Close()

	method := newTestS3Method(fmt.Sprintf("http://%s", closedAddr), 0)
	u, err := url.Parse("/foo/bar")
	c.Assert(err, IsNil)

	resp, err := method.Get(u)
	c.Assert(err, NotNil)
	c.Assert(resp.GetResponseStatusCode(), Equals, 504)
}

func (s *S3TestSuite) TestGetScheme(c *C) {
	opts := S3MethodOpts{Scheme: "s3"}
	c.Assert(opts.GetScheme(), Equals, "s3")
}

func (s *S3TestSuite) TestNewS3MethodEnvCredentials(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigS3))
	c.Assert(err, IsNil)

	os.Setenv("AWS_ACCESS_KEY_ID", "env-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret-key")
	defer os.Unsetenv("AWS_ACCESS_KEY_ID")
	defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

	manager := "test-manager"
	entry := "test-manager.repo.s3"

	method, err := NewS3Method(&manager, &entry)
	c.Assert(err, IsNil)
	m := method.(S3Method)
	c.Assert(m.AccessKeyID, Equals, "env-access-key")
	c.Assert(m.SecretAccessKey, Equals, "env-secret-key")
}
