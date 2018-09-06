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

package config

import (
	"fmt"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/adobe/butler/internal/reloaders"

	"github.com/bouk/monkey"
	log "github.com/sirupsen/logrus"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ConfigTestSuite{})
var TestHttpCase = 0

type ConfigTestSuite struct {
	TestServer *httptest.Server
	Config     *ButlerConfig
}

type TestHttpHandler struct {
}

func (h *TestHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch TestHttpCase {
	case 0:
		// Let's throw a 500
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	case 1:
		http.Error(w, http.StatusText(http.StatusNotFound),
			http.StatusNotFound)
	case 2:
		fmt.Fprintf(w, string(TestConfigBroken))
	default:
		fmt.Fprintf(w, string(TestConfigBroken))
	}
}

var TestConfigEmpty = []byte(``)
var TestConfigNoHandlers = []byte(`[globals]
scheduler-interval2 = 300
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

var TestConfigNoHandlersExit = []byte(`[globals]
scheduler-interval = 300
exit-on-config-failure = "true"
clean-files = true
`)

var TestConfigNoHandlersNoExit = []byte(`[globals]
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
`)

var TestConfigEmptyHandlers = []byte(`[globals]
config-managers = []
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

var TestConfigEmptyHandlersExit = []byte(`[globals]
config-managers = []
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
`)

var TestConfigBroken = []byte(`[globals]
config-managers = ["test-handler"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

var TestConfigBrokenExit = []byte(`[globals]
config-managers = ["test-handler"]
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
`)

var TestConfigBrokenIncompleteHandler = []byte(`[globals]
config-managers = ["test-handler", "test-handler2"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
[test-handler]
  urls = ["localhost", "localhost"]
`)

var TestConfigBrokenIncompleteHandlerExit = []byte(`[globals]
config-managers = ["test-handler", "test-handler2"]
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
[test-handler]
  urls = ["localhost", "localhost"]
`)

var TestConfigCompleteNoExit = []byte(`[globals]
config-managers = ["test-handler", "test-handler2"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
[test-handler]
  urls = ["localhost", "localhost"]

[test-handler2]
`)

// stegen finish this test
var TestConfigCompleteEnvironment = []byte(`[globals]
  config-managers = ["test-handler"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  status-file = "/var/tmp/butler.status"
  [test-handler]
    repos = ["localhost"]
    clean-files = "true"
    mustache-subs = ["foo=env:MSUB"]
    enable-cache = "false"
    cache-path = "/opt/cache/prometheus"
    dest-path = "/opt/prometheus"
    primary-config-name = "prometheus.yml"
    [test-handler.localhost]
      method = "http"
      repo-path = "/butler/configs"
      primary-config = ["test.yml"]
      additional-config = ["test-add.yml"]
      [test-handler.localhost.http]
        retries = "5"
        retry-wait-min = "5"
        retry-wait-max = "10"
        timeout = "10"
    [test-handler.reloader]
      method = "http"
      [test-handler.reloader.http]
        host = "env:RELOADER_HOST"
        port = "9090"
        uri = "/-/reload"
        method = "post"
        payload = "{}"
        content-type = "application/json"
        # retry info and timeouts
        retries = "5"
        retry-wait-min = "5"
        retry-wait-max = "10"
        timeout = "10"
`)

var TestManagerNoUrls = []byte(`[testing]
`)

var TestManagerUrls = []byte(`[testing]
  urls = ["woden.corp.adobe.com", "localhost"]
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external"]
`)

var TestManagerOptsEmpty = []byte(`[testing.localhost]
`)

var TestManagerOptsFail1 = []byte(`[testing.localhost]
 method = "http"
`)

var TestManagerOptsFail2 = []byte(`[testing.localhost]
 method = "http"
 uri-path = "/foo/bar"
`)

var TestManagerOptsFail3 = []byte(`[testing.localhost]
 method = "http"
 uri-path = "/foo/bar"
 dest-path = ""
`)

var TestManagerOptsFail4 = []byte(`[testing.localhost]
 method = "http"
 uri-path = "/foo/bar"
 dest-path = "/bar/baz"
 primary-config = ["prometheus.yml", "prometheus2.yml"]
 additional-config = ["alerts/butler.yml", "rules/commonrules.yml"]
`)

func (s *ConfigTestSuite) SetUpSuite(c *C) {
	s.TestServer = httptest.NewServer(&TestHttpHandler{})
	s.Config = NewButlerConfig()
	u, err := url.Parse(s.TestServer.URL)
	c.Assert(err, IsNil)
	s.Config.Url = u
	log.SetLevel(log.DebugLevel)
}

func (s *ConfigTestSuite) TearDownSuite(c *C) {
	s.TestServer.Close()
}

func (s *ConfigTestSuite) TestConfigSchedulerInterval(c *C) {
	c.Assert(ConfigSchedulerInterval, Equals, 300)
}

func (s *ConfigTestSuite) TestNewConfigClientHttp(c *C) {
	c1, err1 := NewConfigClient("http")
	c2, err2 := NewConfigClient("https")
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)
	c.Assert(c1.Scheme, Equals, "http")
	c.Assert(c2.Scheme, Equals, "http")
	log.Debugf("c1=%#v c2=%#v\n", c1, c2)
}

func (s *ConfigTestSuite) TestNewConfigClientDefault(c *C) {
	c1, err1 := NewConfigClient("hiya")
	c.Assert(err1, NotNil)
	c.Assert(c1.Scheme, Equals, "")
}

func (s *ConfigTestSuite) TestConfigConfigHandler_InternalServerError(c *C) {
	var err error
	TestHttpCase = 0
	s.Config.SetScheme(s.Config.Url.Scheme)
	s.Config.SetPath(s.Config.Url.Path)
	s.Config.Init()
	err = s.Config.Handler()
	//log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "GET.*attempts")
}

func (s *ConfigTestSuite) TestConfigConfigHandler_NotFound(c *C) {
	var err error
	TestHttpCase = 1
	urlSplit := strings.Split(s.TestServer.URL, "://")
	s.Config.SetScheme(urlSplit[0])
	s.Config.SetPath(urlSplit[1])
	s.Config.Init()
	err = s.Config.Handler()
	//log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "Did not receive 200.*404")
}

func (s *ConfigTestSuite) TestParseConfigEmpty(c *C) {
	var err error
	err = ParseConfig(TestConfigEmpty)
	log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	//c.Assert(err.Error(), Matches, "No globals.config-managers in butler.*")
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
}

func (s *ConfigTestSuite) TestParseConfigBrokenNoHandlersNoExit(c *C) {
	var err error
	err = ParseConfig(TestConfigNoHandlers)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	//c.Assert(err.Error(), Matches, "No globals.config-managers in butler.*")
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
}

func (s *ConfigTestSuite) TestParseConfigBrokenNoHandlersExit(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseConfig(TestConfigNoHandlersExit)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
	c.Assert(ExitTest, Equals, 0)
}

func (s *ConfigTestSuite) TestParseConfigBrokenNoHandlersNoExit2(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseConfig(TestConfigNoHandlersNoExit)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
	c.Assert(ExitTest, Equals, 0)
}

func (s *ConfigTestSuite) TestParseConfigBrokenEmptyHandlersNoExit(c *C) {
	var err error
	err = ParseConfig(TestConfigEmptyHandlers)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
}

func (s *ConfigTestSuite) TestParseConfigBrokenEmptyHandlersExit(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 0
	})
	defer patch.Unpatch()
	err = ParseConfig(TestConfigEmptyHandlersExit)
	c.Assert(err, NotNil)
	c.Assert(ExitTest, Equals, 0)
}

func (s *ConfigTestSuite) TestParseConfigBrokenIncompleteHandlerNoExit(c *C) {
	var err error
	err = ParseConfig(TestConfigBrokenIncompleteHandler)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	// stegen
	//c.Assert(err.Error(), Matches, "Cannot find manager for test-handler2")
}

func (s *ConfigTestSuite) TestParseConfigBrokenIncompleteHandlerExit(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseConfig(TestConfigBrokenIncompleteHandlerExit)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "could not retrieve config options for test-handler.*")
	c.Assert(ExitTest, Equals, 0)
}

/*
func (s *ConfigTestSuite) TestParseConfigCompleteNoExit(c *C) {
	var err error
	err = ParseConfig(TestConfigCompleteNoExit)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "Cannot find handler for test-handler2")
}
*/

func (s *ConfigTestSuite) TestGetConfigManagerNoUrls(c *C) {
	var err error

	// Load the config initially
	err = ParseConfig(TestManagerNoUrls)
	c.Assert(err, NotNil)
	err = GetConfigManager("testing", &ConfigSettings{})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "No repos configured for manager testing.*")
}

func (s *ConfigTestSuite) TestGetConfigManagerUrls(c *C) {
	var err error

	// Load the config initially
	err = ParseConfig(TestManagerUrls)
	c.Assert(err, NotNil)
	err = GetConfigManager("testing", &ConfigSettings{})
	c.Assert(err, NotNil)
	// stegen
	//c.Assert(err.Error(), Matches, "No urls configured for manager testing.*")
}

func (s *ConfigTestSuite) TestGetManagerOptsNoConfig(c *C) {
	var (
		err  error
		opts *ManagerOpts
	)

	// Load the config initially
	err = ParseConfig(TestManagerOptsEmpty)
	c.Assert(err, NotNil)
	opts, err = GetManagerOpts("testing.localhost", &ConfigSettings{})
	log.Debugf("TestGetManagerOptsNoConfig(): opts=%#v", opts)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "unknown manager.method.*")
}

func (s *ConfigTestSuite) TestGetManagerOptsFullCheck(c *C) {
	var (
		err  error
		opts *ManagerOpts
	)

	// Load the config initially
	err = ParseConfig(TestManagerOptsFail1)
	c.Assert(err, NotNil)
	opts, err = GetManagerOpts("testing.localhost", &ConfigSettings{})
	log.Debugf("TestGetManagerOptsNoConfig(): opts=%#v", opts)
	c.Assert(err, NotNil)
	// stegen
	c.Assert(err.Error(), Matches, "no manager.primary-config defined")
}

func (s *ConfigTestSuite) TestConfigCompleteEnvironment(c *C) {
	var (
		err    error
		config ConfigSettings
	)

	// setup some environment
	reloaderHost := "testing.com"
	mustacheSub := "holla"
	os.Setenv("RELOADER_HOST", reloaderHost)
	os.Setenv("MSUB", mustacheSub)

	// Load the config initially
	err = ParseConfig(TestConfigCompleteEnvironment)
	c.Assert(err, IsNil)

	// Get the configuration
	err = GetConfigManager("test-handler", &config)
	c.Assert(err, IsNil)

	// Let's spot test some entries from the config
	mgr := config.Managers["test-handler"]
	c.Assert(mgr.CleanFiles, Equals, true)
	c.Assert(mgr.EnableCache, Equals, false)
	c.Assert(mgr.MustacheSubs["foo"], Equals, mustacheSub)
	mgrReloaderOpts := config.Managers["test-handler"].Reloader.(reloaders.HttpReloader)
	c.Assert(mgrReloaderOpts.Opts.Host, Equals, reloaderHost)

	// Cleanup env
	os.Unsetenv("RELOADER_HOST")
	os.Unsetenv("MSUB")
}
