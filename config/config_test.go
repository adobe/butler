package config

import (
	"fmt"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	urlSplit := strings.Split(s.TestServer.URL, "://")
	log.Debugf("TestConfigConfigHandler_InternalServerError(): urlSplit=%#v", urlSplit)
	s.Config.SetScheme(urlSplit[0])
	s.Config.SetPath(urlSplit[1])
	s.Config.SetUrl(s.TestServer.URL)
	s.Config.Init()
	log.Debugf("TestConfigConfigHandler_InternalServerError(): s.Config.Config=%#v", s.Config.Config)
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
	s.Config.SetUrl(s.TestServer.URL)
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
	c.Assert(err, IsNil)
	c.Assert(ExitTest, Equals, 5)
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
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseConfig(TestConfigEmptyHandlersExit)
	c.Assert(err, IsNil)
	c.Assert(ExitTest, Equals, 5)
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
	c.Assert(err, IsNil)
	c.Assert(ExitTest, Equals, 5)
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
		err error
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
		err error
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
