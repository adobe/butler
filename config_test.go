package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	. "gopkg.in/check.v1"
	log "github.com/sirupsen/logrus"
)

var _ = Suite(&ButlerConfigTestSuite{})
var TestHttpCase = 0

type ButlerConfigTestSuite struct {
	TestServer *httptest.Server
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
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)
var TestButlerConfigEmptyHandlers = []byte(`[globals]
config-handlers = []
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)
var TestButlerConfigBroken = []byte(`[globals]
config-handlers = ["test-handler"]
scheduler-interval = 300
exit-on-config-failure = false
clean-files = true
`)

func (s *ButlerConfigTestSuite) SetUpSuite(c *C) {
	s.TestServer = httptest.NewServer(&TestHttpHandler{})
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
	ButlerConfigScheme = urlSplit[0]
	ButlerConfigUrl = s.TestServer.URL
	err = ButlerConfigHandler()
	//log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "GET.*attempts")
}

func (s *ButlerConfigTestSuite) TestConfigButlerConfigHandler_NotFound(c *C) {
	var err error
	TestHttpCase = 1
	urlSplit := strings.Split(s.TestServer.URL, "://")
	ButlerConfigScheme = urlSplit[0]
	ButlerConfigUrl = s.TestServer.URL
	err = ButlerConfigHandler()
	//log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "Did not receive 200.*404")
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigEmpty(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigEmpty)
	log.Infof("err=%#v\n", err.Error())
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "No globals.config-handlers in butler.*")
}

func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenNoHandlersNoExit(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigNoHandlers)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "No globals.config-handlers in butler.*")
}
func (s *ButlerConfigTestSuite) TestParseButlerConfigBrokenEmptyHandlersNoExit(c *C) {
	var err error
	err = ParseButlerConfig(TestButlerConfigEmptyHandlers)
	log.Infof("err=%#v\n", err)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "globals.config-handlers = .*")
}
