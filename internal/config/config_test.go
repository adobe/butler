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
	"fmt"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/adobe/butler/internal/metrics"
	"github.com/adobe/butler/internal/methods"
	"github.com/adobe/butler/internal/reloaders"

	"bou.ke/monkey"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ConfigTestSuite{})
var TestHTTPCase = 0

type ConfigTestSuite struct {
	TestServer *httptest.Server
	Config     *ButlerConfig
}

type TestHTTPHandler struct {
}

func (h *TestHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch TestHTTPCase {
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

// Test config for watch-only mode
var TestConfigWatchOnly = []byte(`[globals]
  config-managers = ["test-handler"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  status-file = "/var/tmp/butler.status"
  [test-handler]
    repos = ["localhost"]
    clean-files = "false"
    watch-only = "true"
    skip-butler-header = "true"
    enable-cache = "false"
    primary-config-name = "config.yml"
    [test-handler.localhost]
      method = "file"
      repo-path = "/tmp/butler-test"
      primary-config = ["test.yml"]
      [test-handler.localhost.file]
        path = "/tmp/butler-test"
    [test-handler.reloader]
      method = "http"
      [test-handler.reloader.http]
        host = "localhost"
        port = "8080"
        uri = "/reload"
        method = "post"
        payload = "{}"
        content-type = "application/json"
        retries = "3"
        retry-wait-min = "1"
        retry-wait-max = "5"
        timeout = "10"
`)

// Test config for watch-only mode with dest-path (should still work)
var TestConfigWatchOnlyWithDestPath = []byte(`[globals]
  config-managers = ["test-handler"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  status-file = "/var/tmp/butler.status"
  [test-handler]
    repos = ["localhost"]
    clean-files = "false"
    watch-only = "true"
    skip-butler-header = "true"
    enable-cache = "false"
    dest-path = "/tmp/butler-dest"
    primary-config-name = "config.yml"
    [test-handler.localhost]
      method = "file"
      repo-path = "/tmp/butler-test"
      primary-config = ["test.yml"]
      [test-handler.localhost.file]
        path = "/tmp/butler-test"
`)

// Fixtures for exercising ConfigSettings.ParseConfig's metric-cleanup
// behavior: two managers, each with a distinct repo/file, so that removing
// one manager (or one repo from a surviving manager) between successive
// ParseConfig calls lets us assert the corresponding metrics were deleted.
var TestConfigMetricsTwoManagers = []byte(`[globals]
  config-managers = ["test-metrics-a", "test-metrics-b"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  [test-metrics-a]
    repos = ["repo-a"]
    clean-files = "false"
    enable-cache = "false"
    dest-path = "/tmp/butler-metrics-test-a"
    primary-config-name = "config-a.yml"
    [test-metrics-a.repo-a]
      method = "file"
      repo-path = "/tmp/butler-metrics-test-a"
      primary-config = ["primary-a.yml"]
      [test-metrics-a.repo-a.file]
        path = "/tmp/butler-metrics-test-a"
  [test-metrics-b]
    repos = ["repo-b"]
    clean-files = "false"
    enable-cache = "false"
    dest-path = "/tmp/butler-metrics-test-b"
    primary-config-name = "config-b.yml"
    [test-metrics-b.repo-b]
      method = "file"
      repo-path = "/tmp/butler-metrics-test-b"
      primary-config = ["primary-b.yml"]
      [test-metrics-b.repo-b.file]
        path = "/tmp/butler-metrics-test-b"
`)

var TestConfigMetricsOneManager = []byte(`[globals]
  config-managers = ["test-metrics-a"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  [test-metrics-a]
    repos = ["repo-a"]
    clean-files = "false"
    enable-cache = "false"
    dest-path = "/tmp/butler-metrics-test-a"
    primary-config-name = "config-a.yml"
    [test-metrics-a.repo-a]
      method = "file"
      repo-path = "/tmp/butler-metrics-test-a"
      primary-config = ["primary-a.yml"]
      [test-metrics-a.repo-a.file]
        path = "/tmp/butler-metrics-test-a"
`)

// TestConfigMetricsManagerTwoRepos and TestConfigMetricsManagerOneRepo cover
// the narrower case from the issue: the manager itself survives, but one of
// its two repos is dropped. Metrics for the removed repo/file should be
// deleted while the manager-level metrics (still in use by the surviving
// repo) must NOT be deleted.
var TestConfigMetricsManagerTwoRepos = []byte(`[globals]
  config-managers = ["test-metrics-c"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  [test-metrics-c]
    repos = ["repo-c1", "repo-c2"]
    clean-files = "false"
    enable-cache = "false"
    dest-path = "/tmp/butler-metrics-test-c"
    primary-config-name = "config-c.yml"
    [test-metrics-c.repo-c1]
      method = "file"
      repo-path = "/tmp/butler-metrics-test-c1"
      primary-config = ["primary-c1.yml"]
      [test-metrics-c.repo-c1.file]
        path = "/tmp/butler-metrics-test-c1"
    [test-metrics-c.repo-c2]
      method = "file"
      repo-path = "/tmp/butler-metrics-test-c2"
      primary-config = ["primary-c2.yml"]
      [test-metrics-c.repo-c2.file]
        path = "/tmp/butler-metrics-test-c2"
`)

var TestConfigMetricsManagerOneRepo = []byte(`[globals]
  config-managers = ["test-metrics-c"]
  scheduler-interval = 300
  exit-on-config-failure = "false"
  [test-metrics-c]
    repos = ["repo-c1"]
    clean-files = "false"
    enable-cache = "false"
    dest-path = "/tmp/butler-metrics-test-c"
    primary-config-name = "config-c.yml"
    [test-metrics-c.repo-c1]
      method = "file"
      repo-path = "/tmp/butler-metrics-test-c1"
      primary-config = ["primary-c1.yml"]
      [test-metrics-c.repo-c1.file]
        path = "/tmp/butler-metrics-test-c1"
`)

var TestManagerNoURLs = []byte(`[testing]
`)

var TestManagerURLs = []byte(`[testing]
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
	s.TestServer = httptest.NewServer(&TestHTTPHandler{})
	u, err := url.Parse(s.TestServer.URL)
	opts := &ButlerConfigOpts{
		InsecureSkipVerify: false,
		LogLevel:           log.DebugLevel,
		URL:                u}
	s.Config, err = NewButlerConfig(opts)
	c.Assert(err, IsNil)
}

func (s *ConfigTestSuite) TearDownSuite(c *C) {
	s.TestServer.Close()
}

func (s *ConfigTestSuite) TestConfigSchedulerInterval(c *C) {
	c.Assert(ConfigSchedulerInterval, Equals, 300)
}

func (s *ConfigTestSuite) TestNewConfigClientHttp(c *C) {
	u1, err := url.Parse("http://localhost")
	c.Assert(err, IsNil)
	u2, err := url.Parse("https://localhost")
	c.Assert(err, IsNil)
	opts1 := &ButlerConfigOpts{
		InsecureSkipVerify: false,
		LogLevel:           log.DebugLevel,
		URL:                u1}
	bc1, err := NewButlerConfig(opts1)
	bc1.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u1.Scheme})

	c.Assert(err, IsNil)
	opts2 := &ButlerConfigOpts{
		InsecureSkipVerify: false,
		LogLevel:           log.DebugLevel,
		URL:                u2}
	bc2, err := NewButlerConfig(opts2)
	bc2.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u2.Scheme})
	c.Assert(err, IsNil)
	c1, err1 := NewConfigClient(bc1)
	c2, err2 := NewConfigClient(bc2)
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)
	c.Assert(c1.Scheme, Equals, "http")
	c.Assert(c2.Scheme, Equals, "https")
	log.Debugf("c1=%#v c2=%#v\n", c1, c2)
}

func (s *ConfigTestSuite) TestNewConfigClientDefault(c *C) {
	u, err := url.Parse("hiya://localhost")
	c.Assert(err, IsNil)
	opts := &ButlerConfigOpts{
		InsecureSkipVerify: false,
		LogLevel:           log.DebugLevel,
		URL:                u}
	bc, err := NewButlerConfig(opts)
	_ = bc
	c.Assert(err, NotNil)
	//c1, err1 := NewConfigClient(bc)
	//c.Assert(err1, NotNil)
	//c.Assert(c1.Scheme, Equals, "")
	//_ = err1
	// = c1
}

func (s *ConfigTestSuite) TestConfigConfigHandler_InternalServerError(c *C) {
	var err error
	TestHTTPCase = 0
	u, err := url.Parse(s.TestServer.URL)
	c.Assert(err, IsNil)
	opts := &ButlerConfigOpts{
		InsecureSkipVerify: false,
		LogLevel:           log.DebugLevel,
		URL:                u}
	bc, err := NewButlerConfig(opts)
	c.Assert(err, IsNil)
	bc.SetMethodOpts(methods.HTTPMethodOpts{Scheme: u.Scheme})
	s.Config = bc
	c.Logf("bc=%#v", bc)
	s.Config.Init()
	err = s.Config.Handler()
	//log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "GET.*attempt.*")
}

func (s *ConfigTestSuite) TestConfigConfigHandler_NotFound(c *C) {
	var err error
	TestHTTPCase = 1
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

func (s *ConfigTestSuite) TestGetConfigManagerNoURLs(c *C) {
	var err error

	// Load the config initially
	err = ParseConfig(TestManagerNoURLs)
	c.Assert(err, NotNil)
	err = GetConfigManager("testing", &ConfigSettings{})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "No repos configured for manager testing.*")
}

func (s *ConfigTestSuite) TestGetConfigManagerURLs(c *C) {
	var err error

	// Load the config initially
	err = ParseConfig(TestManagerURLs)
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
	mgrReloaderOpts := config.Managers["test-handler"].Reloader.(reloaders.HTTPReloader)
	c.Assert(mgrReloaderOpts.Opts.Host, Equals, reloaderHost)

	// Cleanup env
	os.Unsetenv("RELOADER_HOST")
	os.Unsetenv("MSUB")
}

func (s *ConfigTestSuite) TestConfigWatchOnlyMode(c *C) {
	var (
		err    error
		config ConfigSettings
	)

	// Load the watch-only config
	err = ParseConfig(TestConfigWatchOnly)
	c.Assert(err, IsNil)

	// Get the configuration
	err = GetConfigManager("test-handler", &config)
	c.Assert(err, IsNil)

	// Verify watch-only mode is enabled
	mgr := config.Managers["test-handler"]
	c.Assert(mgr.WatchOnly, Equals, true)
	c.Assert(mgr.SkipButlerHeader, Equals, true)
	c.Assert(mgr.CleanFiles, Equals, false)

	// Verify FileHashes map is initialized
	c.Assert(mgr.FileHashes, NotNil)

	// Verify dest-path is empty (optional in watch-only mode)
	// Note: filepath.Clean("") returns "." so we check for that
	c.Assert(mgr.DestPath, Equals, "")
}

func (s *ConfigTestSuite) TestConfigWatchOnlyModeWithDestPath(c *C) {
	var (
		err    error
		config ConfigSettings
	)

	// Load the watch-only config with dest-path
	err = ParseConfig(TestConfigWatchOnlyWithDestPath)
	c.Assert(err, IsNil)

	// Get the configuration
	err = GetConfigManager("test-handler", &config)
	c.Assert(err, IsNil)

	// Verify watch-only mode is enabled
	mgr := config.Managers["test-handler"]
	c.Assert(mgr.WatchOnly, Equals, true)
	c.Assert(mgr.SkipButlerHeader, Equals, true)

	// Verify dest-path is set even in watch-only mode (it's optional but allowed)
	c.Assert(mgr.DestPath, Equals, "/tmp/butler-dest")

	// Verify FileHashes map is initialized
	c.Assert(mgr.FileHashes, NotNil)
}

func (s *ConfigTestSuite) TestConfigWatchOnlyModeDisabled(c *C) {
	var (
		err    error
		config ConfigSettings
	)

	// Load the standard config (watch-only not set)
	err = ParseConfig(TestConfigCompleteEnvironment)
	c.Assert(err, IsNil)

	// setup environment for this test
	os.Setenv("RELOADER_HOST", "localhost")
	os.Setenv("MSUB", "test")
	defer os.Unsetenv("RELOADER_HOST")
	defer os.Unsetenv("MSUB")

	// Get the configuration
	err = GetConfigManager("test-handler", &config)
	c.Assert(err, IsNil)

	// Verify watch-only mode is disabled by default
	mgr := config.Managers["test-handler"]
	c.Assert(mgr.WatchOnly, Equals, false)

	// Verify FileHashes map is nil when watch-only is disabled
	c.Assert(mgr.FileHashes, IsNil)
}

// metricHasLabelValue reports whether any series of the named metric family,
// as currently known to the default Prometheus registry, carries the given
// label value. Used to check for a specific manager/repo/file's presence or
// absence without depending on the total series count, which is shared
// (and thus polluted by other tests) across the whole test binary.
func metricHasLabelValue(c *C, metricName string, labelName string, labelValue string) bool {
	families, err := prometheus.DefaultGatherer.Gather()
	c.Assert(err, IsNil)
	for _, family := range families {
		if family.GetName() != metricName {
			continue
		}
		for _, m := range family.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == labelName && lp.GetValue() == labelValue {
					return true
				}
			}
		}
	}
	return false
}

// TestParseConfigDeletesStaleMetricsOnManagerRemoval covers
// https://github.com/adobe/butler/issues/5: when a manager disappears from
// the butler configuration between two ParseConfig calls, its Prometheus
// series must be deleted rather than left reporting stale values forever.
func (s *ConfigTestSuite) TestParseConfigDeletesStaleMetricsOnManagerRemoval(c *C) {
	var settings ConfigSettings

	err := settings.ParseConfig(TestConfigMetricsTwoManagers)
	c.Assert(err, IsNil)

	// Populate metrics for both managers/repos as butler normally would
	// while processing them.
	metrics.SetButlerReloadVal(metrics.SUCCESS, "test-metrics-a")
	metrics.SetButlerReloadVal(metrics.SUCCESS, "test-metrics-b")
	metrics.SetButlerRemoteRepoUp(metrics.SUCCESS, "test-metrics-a")
	metrics.SetButlerRemoteRepoUp(metrics.SUCCESS, "test-metrics-b")
	metrics.SetButlerContactVal(metrics.SUCCESS, "repo-a", "primary-a.yml")
	metrics.SetButlerContactVal(metrics.SUCCESS, "repo-b", "primary-b.yml")

	c.Assert(metricHasLabelValue(c, "butler_localconfig_reload_success", "manager", "test-metrics-b"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_up", "manager", "test-metrics-b"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-b"), Equals, true)

	// Now reparse with test-metrics-b removed from the configuration,
	// simulating an operator deleting that manager/repo from butler.toml.
	err = settings.ParseConfig(TestConfigMetricsOneManager)
	c.Assert(err, IsNil)

	// test-metrics-b's series must be gone...
	c.Assert(metricHasLabelValue(c, "butler_localconfig_reload_success", "manager", "test-metrics-b"), Equals, false)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_up", "manager", "test-metrics-b"), Equals, false)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-b"), Equals, false)

	// ...while test-metrics-a's must remain untouched.
	c.Assert(metricHasLabelValue(c, "butler_localconfig_reload_success", "manager", "test-metrics-a"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_up", "manager", "test-metrics-a"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-a"), Equals, true)
}

// TestParseConfigDeletesStaleMetricsOnRepoRemoval covers the narrower case
// where the manager itself survives a reparse but one of its repos is
// dropped: only the removed repo's (repo, config_file) series should be
// deleted, and the manager-level series (still backed by the surviving
// repo) must be left alone.
func (s *ConfigTestSuite) TestParseConfigDeletesStaleMetricsOnRepoRemoval(c *C) {
	var settings ConfigSettings

	err := settings.ParseConfig(TestConfigMetricsManagerTwoRepos)
	c.Assert(err, IsNil)

	metrics.SetButlerReloadVal(metrics.SUCCESS, "test-metrics-c")
	metrics.SetButlerRemoteRepoUp(metrics.SUCCESS, "test-metrics-c")
	metrics.SetButlerContactVal(metrics.SUCCESS, "repo-c1", "primary-c1.yml")
	metrics.SetButlerContactVal(metrics.SUCCESS, "repo-c2", "primary-c2.yml")

	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-c1"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-c2"), Equals, true)

	// Reparse with repo-c2 dropped from test-metrics-c, but the manager itself survives.
	err = settings.ParseConfig(TestConfigMetricsManagerOneRepo)
	c.Assert(err, IsNil)

	// repo-c2's series must be gone...
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-c2"), Equals, false)

	// ...while repo-c1's and the manager-level series must remain, since the
	// manager is still configured (just with fewer repos).
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_contact_success", "repo", "repo-c1"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_localconfig_reload_success", "manager", "test-metrics-c"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_remoterepo_up", "manager", "test-metrics-c"), Equals, true)
}

// TestParseConfigFailedReparseDoesNotDeleteMetrics ensures that if a
// subsequent ParseConfig call fails (e.g. malformed butler.toml), the
// previous, still-live managers' metrics are left untouched rather than
// being wiped out by a botched reparse.
func (s *ConfigTestSuite) TestParseConfigFailedReparseDoesNotDeleteMetrics(c *C) {
	var settings ConfigSettings

	err := settings.ParseConfig(TestConfigMetricsTwoManagers)
	c.Assert(err, IsNil)

	metrics.SetButlerReloadVal(metrics.SUCCESS, "test-metrics-a")
	metrics.SetButlerReloadVal(metrics.SUCCESS, "test-metrics-b")

	// Attempt to reparse with a broken config (references a manager that
	// isn't defined as a TOML table at all).
	err = settings.ParseConfig(TestConfigBrokenIncompleteHandler)
	c.Assert(err, NotNil)

	// Neither manager's metrics should have been touched by the failed parse.
	c.Assert(metricHasLabelValue(c, "butler_localconfig_reload_success", "manager", "test-metrics-a"), Equals, true)
	c.Assert(metricHasLabelValue(c, "butler_localconfig_reload_success", "manager", "test-metrics-b"), Equals, true)

	// And the live manager set should still be the last successfully parsed one.
	c.Assert(settings.Managers["test-metrics-a"], NotNil)
	c.Assert(settings.Managers["test-metrics-b"], NotNil)
}

// TestParseConfigFirstRunDoesNotPanic ensures collectMetricIdentities and
// deleteStaleMetrics handle a nil/empty prior manager map (the very first
// ParseConfig call on a fresh ConfigSettings) without panicking or deleting
// anything spuriously.
func (s *ConfigTestSuite) TestParseConfigFirstRunDoesNotPanic(c *C) {
	var settings ConfigSettings
	c.Assert(settings.Managers, IsNil)

	err := settings.ParseConfig(TestConfigMetricsTwoManagers)
	c.Assert(err, IsNil)
	c.Assert(settings.Managers["test-metrics-a"], NotNil)
	c.Assert(settings.Managers["test-metrics-b"], NotNil)
}
