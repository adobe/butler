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

var _ = Suite(&ButlerConfigTestSuite{})
var TestHttpCase = 0

type ButlerConfigTestSuite struct {
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
		fmt.Fprintf(w, string(TestButlerConfigBroken))
	default:
		fmt.Fprintf(w, string(TestButlerConfigBroken))
	}
}

var TestButlerConfigEmpty = []byte(``)
var TestButlerConfigNoHandlers = []byte(`[globals]
scheduler-interval2 = 300
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

var TestButlerConfigNoHandlersExit = []byte(`[globals]
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
`)
var TestButlerConfigEmptyHandlers = []byte(`[globals]
config-managers = []
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

var TestButlerConfigEmptyHandlersExit = []byte(`[globals]
config-managers = []
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
`)

var TestButlerConfigBroken = []byte(`[globals]
config-managers = ["test-handler"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

var TestButlerConfigBrokenExit = []byte(`[globals]
config-managers = ["test-handler"]
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
`)

var TestButlerConfigBrokenIncompleteHandler = []byte(`[globals]
config-managers = ["test-handler", "test-handler2"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
[test-handler]
  urls = ["localhost", "localhost"]
`)

var TestButlerConfigBrokenIncompleteHandlerExit = []byte(`[globals]
config-managers = ["test-handler", "test-handler2"]
scheduler-interval = 300
exit-on-config-failure = true
clean-files = true
[test-handler]
  urls = ["localhost", "localhost"]
`)

var TestButlerConfigCompleteNoExit = []byte(`[globals]
config-managers = ["test-handler", "test-handler2"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
[test-handler]
  urls = ["localhost", "localhost"]

[test-handler2]
`)

var TestButlerManagerNoUrls = []byte(`[testing]
`)

var TestButlerManagerUrls = []byte(`[testing]
  urls = ["woden.corp.adobe.com", "localhost"]
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external"]
`)

var TestButlerManagerOptsEmpty = []byte(`[testing.localhost]
`)

var TestButlerManagerOptsFail1 = []byte(`[testing.localhost]
 method = "http"
`)

var TestButlerManagerOptsFail2 = []byte(`[testing.localhost]
 method = "http"
 uri-path = "/foo/bar"
`)

var TestButlerManagerOptsFail3 = []byte(`[testing.localhost]
 method = "http"
 uri-path = "/foo/bar"
 dest-path = ""
`)

var TestButlerManagerOptsFail4 = []byte(`[testing.localhost]
 method = "http"
 uri-path = "/foo/bar"
 dest-path = "/bar/baz"
 primary-config = ["prometheus.yml", "prometheus2.yml"]
 additional-config = ["alerts/butler.yml", "rules/commonrules.yml"]
`)

func (s *ButlerConfigTestSuite) SetUpSuite(c *C) {
	s.TestServer = httptest.NewServer(&TestHttpHandler{})
	s.Config = NewButlerConfig()
	log.SetLevel(log.DebugLevel)
}

func (s *ButlerConfigTestSuite) TearDownSuite(c *C) {
	s.TestServer.Close()
}

func (s *ButlerConfigTestSuite) TestConfigSchedulerInterval(c *C) {
	c.Assert(ConfigSchedulerInterval, Equals, 300)
}

func (s *ButlerConfigTestSuite) TestNewButlerConfigClientHttp(c *C) {
	c1, err1 := NewButlerConfigClient("http")
	c2, err2 := NewButlerConfigClient("https")
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)
	c.Assert(c1.Scheme, Equals, "http")
	c.Assert(c2.Scheme, Equals, "http")
	log.Debugf("c1=%#v c2=%#v\n", c1, c2)
}

func (s *ButlerConfigTestSuite) TestNewButlerConfigClientDefault(c *C) {
	c1, err1 := NewButlerConfigClient("hiya")
	c.Assert(err1, NotNil)
	c.Assert(c1.Scheme, Equals, "")
}

func (s *ButlerConfigTestSuite) TestConfigButlerConfigHandler_InternalServerError(c *C) {
	var err error
	TestHttpCase = 0
	urlSplit := strings.Split(s.TestServer.URL, "://")
	s.Config.SetScheme(urlSplit[0])
	s.Config.SetPath(urlSplit[1])
	s.Config.SetUrl(s.TestServer.URL)
	s.Config.Init()
	err = s.Config.Handler()
	//log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "GET.*attempts")
}

func (s *ButlerConfigTestSuite) TestConfigButlerConfigHandler_NotFound(c *C) {
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

func (s *ButlerConfigTestSuite) TestParseButlerConfigEmpty(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigEmpty)
	log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	//c.Assert(err.Error(), Matches, "No globals.config-managers in butler.*")
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenNoHandlersNoExit(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigNoHandlers)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	//c.Assert(err.Error(), Matches, "No globals.config-managers in butler.*")
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenNoHandlersExit(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseButlerConfig(TestButlerConfigNoHandlersExit)
	c.Assert(err, IsNil)
	c.Assert(ExitTest, Equals, 5)
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenEmptyHandlersNoExit(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigEmptyHandlers)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "globals.config-managers has no entries.*")
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenEmptyHandlersExit(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseButlerConfig(TestButlerConfigEmptyHandlersExit)
	c.Assert(err, IsNil)
	c.Assert(ExitTest, Equals, 5)
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenIncompleteHandlerNoExit(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigBrokenIncompleteHandler)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	// stegen
	//c.Assert(err.Error(), Matches, "Cannot find manager for test-handler2")
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenIncompleteHandlerExit(c *C) {
	var err error
	var ExitTest = 0
	patch := monkey.Patch(log.Fatalf, func(string, ...interface{}) {
		ExitTest = 5
	})
	defer patch.Unpatch()
	err = ParseButlerConfig(TestButlerConfigBrokenIncompleteHandlerExit)
	c.Assert(err, IsNil)
	c.Assert(ExitTest, Equals, 5)
}

/*
func (s *ButlerConfigTestSuite) TestParseButlerConfigCompleteNoExit(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigCompleteNoExit)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "Cannot find handler for test-handler2")
}
*/

func (s *ButlerConfigTestSuite) TestGetButlerConfigManagerNoUrls(c *C) {
	var err error

	// Load the config initially
	err = ParseButlerConfig(TestButlerManagerNoUrls)
	c.Assert(err, NotNil)
	err = GetButlerConfigManager("testing", &ButlerConfigSettings{})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "No urls configured for manager testing.*")
}

func (s *ButlerConfigTestSuite) TestGetButlerConfigManagerUrls(c *C) {
	var err error

	// Load the config initially
	err = ParseButlerConfig(TestButlerManagerUrls)
	c.Assert(err, NotNil)
	err = GetButlerConfigManager("testing", &ButlerConfigSettings{})
	c.Assert(err, NotNil)
	// stegen
	//c.Assert(err.Error(), Matches, "No urls configured for manager testing.*")
}

func (s *ButlerConfigTestSuite) TestGetButlerManagerOptsNoConfig(c *C) {
	var err error

	// Load the config initially
	err = ParseButlerConfig(TestButlerManagerOptsEmpty)
	c.Assert(err, NotNil)
	err = GetButlerManagerOpts("testing.localhost", &ButlerConfigSettings{})
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "unknown manager.method.*")
}

func (s *ButlerConfigTestSuite) TestGetButlerManagerOptsFullCheck(c *C) {
	var err error

	// Load the config initially
	err = ParseButlerConfig(TestButlerManagerOptsFail1)
	c.Assert(err, NotNil)
	err = GetButlerManagerOpts("testing.localhost", &ButlerConfigSettings{})
	c.Assert(err, NotNil)
	// stegen
	c.Assert(err.Error(), Matches, "unknown manager.method.*")
}
