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
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/storage"
	//log "github.com/sirupsen/logrus"
	"github.com/bouk/monkey"
	"github.com/spf13/viper"
	. "gopkg.in/check.v1"
)

var _ = Suite(&BlobTestSuite{})

type BlobTestSuite struct {
}

var TestViperConfigBlob = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "blob"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.blob]
      storage-account-name = "stegentestblobva7"
      storage-account-key = "aGl5YWhpeWFoaXlh"
`)

var TestViperConfigBlobEnv = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "blob"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    [test-manager.repo.blob]
      storage-account-name = "env:STORAGE_ACCOUNT"
      storage-account-key = "env:STORAGE_KEY"
`)

var TestViperConfigBlobNoAccount = []byte(`[test-manager]
  repos = ["repo"]
  clean-files = "true"
  mustache-subs = ["ethos-cluster-id=ethos01-dev-or1", "endpoint=external", "authkey=env:AUTH_KEY"]
  enable-cache = "true"
  cache-path = "/opt/cache/prometheus"
  dest-path = "/opt/prometheus"
  primary-config-name = "prometheus.yml"
  [test-manager.repo]
    method = "blob"
    repo-path = "/var/www/html/butler/configs/prometheus"
    primary-config = ["prometheus.yml", "prometheus-other.yml"]
    additional-config = ["alerts/commonalerts.yml", "butler/butler.yml"]
    #[test-manager.repo.blob]
    #  storage-account-name = "stegentestblobva7"
`)

func (s *BlobTestSuite) SetUpSuite(c *C) {
	viper.SetConfigType("toml")
}

func (s *BlobTestSuite) TearDownSuite(c *C) {
}

func (s *BlobTestSuite) TestNewBlobMethod(c *C) {
	// load config
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigBlob))
	c.Assert(err, IsNil)

	// Reset some environment
	os.Unsetenv("ACCOUNT_NAME")
	os.Unsetenv("ACCOUNT_KEY")

	// setup some stuff
	manager := "test-manager"
	entry := "test-manager.repo.blob"

	method, err := NewBlobMethod(&manager, &entry)
	m := method.(BlobMethod)
	c.Assert(m.StorageAccount, Equals, "stegentestblobva7")
	c.Assert(m.StorageKey, Equals, "aGl5YWhpeWFoaXlh")
	c.Assert(m.AzureClient.HTTPClient, NotNil)
	c.Assert(err, IsNil)

	// Let's reset the viper config
	err = viper.ReadConfig(bytes.NewBuffer(TestViperConfigBlobNoAccount))
	c.Assert(err, IsNil)

	// Let's override the storage account
	os.Setenv("ACCOUNT_NAME", "newblob")
	method, err = NewBlobMethod(&manager, &entry)
	m = method.(BlobMethod)
	c.Assert(m.StorageAccount, Equals, "")
	c.Assert(m.AzureClient.HTTPClient, IsNil)
	c.Assert(err, NotNil)

	os.Unsetenv("ACCOUNT_NAME")
	os.Unsetenv("ACCOUNT_KEY")

	// test out the environment stuff
	err = viper.ReadConfig(bytes.NewBuffer(TestViperConfigBlobEnv))
	c.Assert(err, IsNil)
	method, err = NewBlobMethod(&manager, &entry)
	c.Assert(err, NotNil)

	os.Setenv("ACCOUNT_NAME", "boombam")
	os.Setenv("ACCOUNT_KEY", "hiya")
	method, err = NewBlobMethod(&manager, &entry)
	c.Assert(err, IsNil)
	m = method.(BlobMethod)
	c.Assert(m.StorageAccount, Equals, "boombam")
	os.Unsetenv("ACCOUNT_NAME")
	os.Unsetenv("ACCOUNT_KEY")
}

func (s *BlobTestSuite) TestNewBlobMethodWithAccountAndKey(c *C) {
	// load config
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigBlob))
	c.Assert(err, IsNil)

	// setup some stuff
	//manager := "test-manager"
	//entry := "test-manager.repo.blob"

	// Let's setup a fake token
	os.Setenv("BUTLER_STORAGE_TOKEN", "hiya")

	method, err := NewBlobMethodWithAccountAndKey("stegentestblobva7", "aGl5YWhpeWFoaXlh")
	m := method.(BlobMethod)
	c.Assert(m.StorageAccount, Equals, "stegentestblobva7")
	c.Assert(m.StorageKey, Equals, "aGl5YWhpeWFoaXlh")
	c.Assert(err, IsNil)

	method, err = NewBlobMethodWithAccountAndKey("", "")
	m = method.(BlobMethod)
	c.Assert(m.StorageAccount, Equals, "")
	c.Assert(m.StorageKey, Equals, "")
	c.Assert(err, NotNil)

	os.Setenv("STORAGE_ACCOUNT", "boombam")
	os.Setenv("STORAGE_KEY", "aGl5YWhpeWFoaXlh")
	method, err = NewBlobMethodWithAccountAndKey("env:STORAGE_ACCOUNT", "env:STORAGE_KEY")
	m = method.(BlobMethod)
	c.Assert(m.StorageAccount, Equals, "boombam")
	c.Assert(m.StorageKey, Equals, "aGl5YWhpeWFoaXlh")
	c.Assert(err, IsNil)

	os.Unsetenv("STORAGE_ACCOUNT")
	os.Unsetenv("STORAGE_KEY")
}

func (s *BlobTestSuite) TestGet(c *C) {
	// load config
	err := viper.ReadConfig(bytes.NewBuffer(TestViperConfigBlob))
	c.Assert(err, IsNil)

	// setup some stuff
	manager := "test-manager"
	entry := "test-manager.repo.blob"
	// Let's setup a fake token
	os.Setenv("BUTLER_STORAGE_TOKEN", "hiya")

	// This will error due to no BUTLER_STORAGE_TOKEN
	method, err := NewBlobMethod(&manager, &entry)
	c.Assert(err, IsNil)

	u, err := url.Parse("none")
	c.Assert(err, IsNil)
	resp, err := method.Get(u)
	c.Assert(err, NotNil)

	var b *storage.Blob
	patch := monkey.PatchInstanceMethod(reflect.TypeOf(b), "Get", func(*storage.Blob, *storage.GetBlobOptions) (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader([]byte("hiya"))), nil
	})
	defer patch.Unpatch()

	u, err = url.Parse("/foo/bar")
	c.Assert(err, IsNil)
	resp, err = method.Get(u)
	c.Assert(err, IsNil)
	c.Assert(resp, NotNil)
	out, err := ioutil.ReadAll(resp.GetResponseBody())
	c.Assert(err, IsNil)
	c.Assert(string(out), Equals, "hiya")
	c.Assert(resp.GetResponseStatusCode(), Equals, 200)

	patch = monkey.PatchInstanceMethod(reflect.TypeOf(b), "Get", func(*storage.Blob, *storage.GetBlobOptions) (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewReader([]byte("boom"))), errors.New("some error")
	})
	u, err = url.Parse("/foo/bar")
	c.Assert(err, IsNil)
	resp, err = method.Get(u)
	c.Assert(err, NotNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "some error")
	c.Assert(resp.GetResponseStatusCode(), Equals, 504)

	os.Unsetenv("BUTLER_STORAGE_TOKEN")
	os.Unsetenv("BUTLER_STORAGE_ACCOUNT")
}
