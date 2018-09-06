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

package methods

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"strings"

	//log "github.com/sirupsen/logrus"

	"github.com/bouk/monkey"
	"github.com/coreos/etcd/client"
	"github.com/spf13/viper"
	. "gopkg.in/check.v1"
)

//func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&EtcdTestSuite{})

type EtcdTestSuite struct {
}

var TestViperConfigEtcd = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "etcd"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.etcd]
      endpoints = "http://127.0.0.1:2379"
`)

var TestViperConfigEnvEtcd = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "etcd"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.etcd]
      endpoints = "env:ENDPOINTS"
`)

var TestViperConfigGetEtcd = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "etcd"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.etcd]
	  endpoints = "http://127.0.0.1:2379"
`)

func (s *EtcdTestSuite) SetUpSuite(c *C) {
	viper.SetConfigType("toml")
}

func (s *EtcdTestSuite) TearDownSuite(c *C) {
}

func (s *EtcdTestSuite) TestNewEtcdMethod(c *C) {
	// Load config
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigEtcd))
	c.Assert(err, IsNil)

	manager := "test-manager"
	entry := "test-manager.repo.etcd"
	method, err := NewEtcdMethod(&manager, &entry)
	m := method.(EtcdMethod)
	c.Assert(m.Endpoints, DeepEquals, []string{"http://127.0.0.1:2379"})
	c.Assert(err, IsNil)
}

func (s *EtcdTestSuite) TestNewEtcdMethodEnv(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigEnvEtcd))
	c.Assert(err, IsNil)

	endpoints := []string{"http://127.0.0.1:2379", "http://127.0.0.2:2379"}

	os.Setenv("ENDPOINTS", strings.Join(endpoints, ","))
	manager := "test-manager"
	entry := "test-manager.repo.etcd"
	method, err := NewEtcdMethod(&manager, &entry)
	m := method.(EtcdMethod)
	c.Assert(err, IsNil)
	c.Assert(m.Endpoints, DeepEquals, endpoints)
	os.Unsetenv("ENDPOINTS")
}

func (s *EtcdTestSuite) TestNewEtcdMethodWithUrl(c *C) {
	endpoints := []string{"http://127.0.0.2:2379", "http://127.0.0.1:2379"}
	method, err := NewEtcdMethodWithEndpoints(endpoints)
	c.Assert(err, IsNil)
	m := method.(EtcdMethod)
	c.Assert(m.Endpoints, DeepEquals, endpoints)
}

func (s *EtcdTestSuite) TestGetPass(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigGetEtcd))
	c.Assert(err, IsNil)
	manager := "test-manager"
	entry := "test-manager.repo.etcd"
	u, err := url.Parse("none")
	c.Assert(err, IsNil)
	endpoints := []string{"http://127.0.0.2:2379"}

	patch := monkey.Patch(GetEtcdKey, func(_ EtcdMethod, _ context.Context, _ string, _ *client.GetOptions) (*client.Response, error) {
		return &client.Response{
			Node: &client.Node{
				Value: "hiya",
			},
		}, nil
	})
	defer patch.Unpatch()

	method1, err1 := NewEtcdMethodWithEndpoints(endpoints)
	method2, err2 := NewEtcdMethod(&manager, &entry)
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)

	resp1, err1 := method1.Get(u)
	resp2, err2 := method2.Get(u)
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

func (s *EtcdTestSuite) TestGetFail(c *C) {
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigGetEtcd))
	c.Assert(err, IsNil)
	manager := "test-manager"
	entry := "test-manager.repo.etcd"
	u, err := url.Parse("none")
	endpoints := []string{"http://127.0.0.3:2379"}
	c.Assert(err, IsNil)
	method1, err1 := NewEtcdMethodWithEndpoints(endpoints)
	method2, err2 := NewEtcdMethod(&manager, &entry)
	c.Assert(err1, IsNil)
	c.Assert(err2, IsNil)

	resp1, err1 := method1.Get(u)
	resp2, err2 := method2.Get(u)
	c.Assert(err1, NotNil)
	c.Assert(err2, NotNil)

	c.Assert(resp1.GetResponseStatusCode(), Equals, 404)
	c.Assert(resp1.GetResponseBody(), IsNil)

	c.Assert(resp2.GetResponseStatusCode(), Equals, 404)
	c.Assert(resp2.GetResponseBody(), IsNil)
}
