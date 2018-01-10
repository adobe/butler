package methods

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	//log "github.com/sirupsen/logrus"
	"github.com/bouk/monkey"
	"github.com/spf13/viper"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&FileTestSuite{})

type FileTestSuite struct {
}

var TestViperConfig = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "file"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.file]
      path = "/var/www/html/butler/configs/prometheus"
`)

var TestViperConfigEnv = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "file"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.file]
      path = "env:BUTLER_PATH"
`)

var TestViperConfigGet = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "file"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.file]
      #path = "/var/www/html/butler/configs/prometheus"
`)

func (s *FileTestSuite) SetUpSuite(c *C) {
	viper.SetConfigType("toml")
}

func (s *FileTestSuite) TearDownSuite(c *C) {
}

func (s *FileTestSuite) TestNewFileMethod(c *C) {
	// Load config
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfig))
	c.Assert(err, IsNil)

	manager := "test-manager"
	entry := "test-manager.repo.file"
	method, err := NewFileMethod(&manager, &entry)
	m := method.(FileMethod)
	c.Assert(m.Path, Equals, "/var/www/html/butler/configs/prometheus")
	c.Assert(err, IsNil)
}

func (s *FileTestSuite) TestNewFileMethodEnv(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigEnv))
	c.Assert(err, IsNil)

	path := "/var/www/html/butler/configs/hiya"
	os.Setenv("BUTLER_PATH", path)
	manager := "test-manager"
	entry := "test-manager.repo.file"
	method, err := NewFileMethod(&manager, &entry)
	m := method.(FileMethod)
	c.Assert(err, IsNil)
	c.Assert(m.Path, Equals, path)
	os.Unsetenv("BUTLER_PATH")
}

func (s *FileTestSuite) TestNewFileMethodWithPath(c *C) {
	path := "/var/www/html/butler/configs/hiya"
	method, err := NewFileMethodWithPath(path)
	c.Assert(err, IsNil)
	m := method.(FileMethod)
	c.Assert(m.Path, Equals, path)
}

func (s *FileTestSuite) TestGetPass(c *C) {
	manager := "test-manager"
	entry := "test-manager.repo.file"
	method1, err1 := NewFileMethodWithPath("none")
	method2, err2 := NewFileMethod(&manager, &entry)
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)

	patch := monkey.Patch(ioutil.ReadFile, func(string) ([]byte, error) {
		return []byte(`hiya`), nil
	})
	defer patch.Unpatch()

	resp1, err1 := method1.Get("none")
	resp2, err2 := method2.Get("none")
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)
	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	buf1.ReadFrom(resp1.GetResponseBody())
	buf2.ReadFrom(resp2.GetResponseBody())
	c.Assert(resp1.GetResponseStatusCode(), Equals, 200)
	c.Assert(buf1.String(), Equals, "hiya")

	c.Assert(resp2.GetResponseStatusCode(), Equals, 200)
	c.Assert(buf2.String(), Equals, "hiya")
}

func (s *FileTestSuite) TestGetFail(c *C) {
	manager := "test-manager"
	entry := "test-manager.repo.file"
	method1, err1 := NewFileMethodWithPath("none")
	method2, err2 := NewFileMethod(&manager, &entry)
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)

	resp1, err1 := method1.Get("none")
	resp2, err2 := method2.Get("none")
	c.Assert(err1, NotNil)
	c.Assert(err2, NotNil)

	c.Assert(resp1.GetResponseStatusCode(), Equals, 504)
	c.Assert(resp1.GetResponseBody(), IsNil)

	c.Assert(resp2.GetResponseStatusCode(), Equals, 504)
	c.Assert(resp2.GetResponseBody(), IsNil)
}
