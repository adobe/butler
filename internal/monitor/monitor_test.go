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

package monitor

import (
	. "gopkg.in/check.v1"

	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	//"net/http/httptest"
	"net/url"
	"os"
	"testing"
	//"time"

	"github.com/adobe/butler/internal/config"
	"github.com/adobe/butler/internal/methods"
	//log "github.com/sirupsen/logrus"
)

func Test(t *testing.T) { TestingT(t) }

type ButlerTestSuite struct {
	bm *Monitor
}

var _ = Suite(&ButlerTestSuite{})

var TestSSLCert = []byte(`-----BEGIN CERTIFICATE-----
MIICMDCCAbcCCQCPiqEX92or1DAKBggqhkjOPQQDAjCBgTELMAkGA1UEBhMCVVMx
CzAJBgNVBAgMAkNBMREwDwYDVQQHDAhTYW4gSm9zZTEOMAwGA1UECgwFQWRvYmUx
CzAJBgNVBAsMAklUMRIwEAYDVQQDDAlsb2NhbGhvc3QxITAfBgkqhkiG9w0BCQEW
Em1hdHRoc21pQGFkb2JlLmNvbTAeFw0xODA5MDUyMDE3MjhaFw0yODA5MDIyMDE3
MjhaMIGBMQswCQYDVQQGEwJVUzELMAkGA1UECAwCQ0ExETAPBgNVBAcMCFNhbiBK
b3NlMQ4wDAYDVQQKDAVBZG9iZTELMAkGA1UECwwCSVQxEjAQBgNVBAMMCWxvY2Fs
aG9zdDEhMB8GCSqGSIb3DQEJARYSbWF0dGhzbWlAYWRvYmUuY29tMHYwEAYHKoZI
zj0CAQYFK4EEACIDYgAEVtt75B8bC133CO0BNsMoeC8pgL9hLYNcINRwyBi430tX
arE04Kyqh5o6K00vbUzVVrgbLCq//UUWRZ8tRFN70oJAP/ywNW60qehjLP21yi2o
+qOksK1I6nej/HtLLn60MAoGCCqGSM49BAMCA2cAMGQCMDyXmvtN7D06uprMfRcN
GEgtPRPP9w3KUn9RmHPGago9oSmsN6pHf949NBCQmDyQHQIwEtE2jQiAHlxHL5vg
7CCRHRQGsnNQa3HeDHptrpaKpaSuRr2rirApd6RnkkN9OLd4
-----END CERTIFICATE-----
`)

var TestSSLKey = []byte(`-----BEGIN EC PARAMETERS-----
BgUrgQQAIg==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDD+3gb9CJH+7yQhd+mvwcvPR6fquStZrp33fe2Po2K4lRGJ05Xo3Ncy
MYmQvdTVex+gBwYFK4EEACKhZANiAARW23vkHxsLXfcI7QE2wyh4LymAv2Etg1wg
1HDIGLjfS1dqsTTgrKqHmjorTS9tTNVWuBssKr/9RRZFny1EU3vSgkA//LA1brSp
6GMs/bXKLaj6o6SwrUjqd6P8e0sufrQ=
-----END EC PRIVATE KEY-----
`)

func (s *ButlerTestSuite) SetUpSuite(c *C) {
	//ParseConfigFiles(&Files, FileList)
}

func (s *ButlerTestSuite) TestNewMonitor(c *C) {
	u, err := url.Parse("https://localhost")
	c.Assert(err, IsNil)
	opts := &config.ButlerConfigOpts{
		InsecureSkipVerify: true,
		URL:                u}
	bc, err := config.NewButlerConfig(opts)
	c.Assert(err, IsNil)
	bc.SetMethodOpts(methods.HTTPMethodOpts{Scheme: bc.Scheme()})
	m := NewMonitor().WithOpts(&Opts{Config: bc, Version: "1.2.3"})
	c.Assert(bc, Equals, m.config)
}

func (s *ButlerTestSuite) TestStartHTTP(c *C) {
	// have to set some stuff up first
	u, err := url.Parse("https://localhost:58532")
	c.Assert(err, IsNil)
	opts := &config.ButlerConfigOpts{
		InsecureSkipVerify: true,
		URL:                u}
	bc, err := config.NewButlerConfig(opts)
	c.Assert(err, IsNil)
	bc.Config = config.NewConfigSettings()
	bc.Config.Globals.HTTPProto = "http"
	bc.Config.Globals.HTTPPort = 58532
	bc.Config.Globals.EnableHTTPLog = true
	m := NewMonitor().WithOpts(&Opts{Config: bc, Version: "1.2.3"})
	s.bm = m
	c.Assert(bc, Equals, m.config)
	// WOW
	s.bm.Start()
	defer s.bm.Stop()
	host := fmt.Sprintf("%v://127.0.0.1:%v/health-check", bc.Config.Globals.HTTPProto, bc.Config.Globals.HTTPPort)
	resp, err := http.Get(host)
	c.Assert(err, IsNil)
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	c.Assert(buf.String(), Matches, `.*"http-proto\":\"http\",\"http-port\":58532,.*`)
}

func (s *ButlerTestSuite) TestStartHTTPs(c *C) {
	// have to set some stuff up first
	tmpCert, err := ioutil.TempFile("", "tmpCert")
	c.Assert(err, IsNil)
	tmpKey, err := ioutil.TempFile("", "tmpCert")
	c.Assert(err, IsNil)
	defer os.Remove(tmpCert.Name())
	defer os.Remove(tmpKey.Name())
	_, err = tmpCert.Write(TestSSLCert)
	c.Assert(err, IsNil)
	_, err = tmpKey.Write(TestSSLKey)
	c.Assert(err, IsNil)
	err = tmpCert.Close()
	c.Assert(err, IsNil)
	err = tmpKey.Close()
	c.Assert(err, IsNil)

	u, err := url.Parse("https://localhost")
	c.Assert(err, IsNil)
	opts := &config.ButlerConfigOpts{
		InsecureSkipVerify: true,
		URL:                u}
	bc, err := config.NewButlerConfig(opts)
	bc.Config = config.NewConfigSettings()
	bc.Config.Globals.HTTPProto = "https"
	bc.Config.Globals.HTTPPort = 58532
	bc.Config.Globals.EnableHTTPLog = false
	bc.Config.Globals.HTTPTLSCert = tmpCert.Name()
	bc.Config.Globals.HTTPTLSKey = tmpKey.Name()
	s.bm.Update(bc)
	c.Assert(bc, Equals, s.bm.config)

	// WOWER
	host := fmt.Sprintf("%v://127.0.0.1:%v/health-check", bc.Config.Globals.HTTPProto, bc.Config.Globals.HTTPPort)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(host)
	c.Assert(err, IsNil)
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	c.Assert(buf.String(), Matches, `.*"http-proto\":\"https\",\"http-port\":58532,.*`)
}
