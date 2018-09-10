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
	. "gopkg.in/check.v1"
	"net/http"
)

var _ = Suite(&HTTPTestSuite{})

type HTTPTestSuite struct {
}

func (s *HTTPTestSuite) TestgetBasicAuthorization(c *C) {
	username := "testing"
	password := "testing"
	result := "Basic dGVzdGluZzp0ZXN0aW5n"
	r := getBasicAuthorization(username, password)
	c.Assert(r, Equals, result)
}

func (s *HTTPTestSuite) TestgetMD5(c *C) {
	result := "e2c50ded5d3990bdabeb4b44c4411f18"
	res := getMD5("hiya")
	c.Assert(result, Equals, res)
}

func (s *HTTPTestSuite) TestgetCnonce(c *C) {
	c.Assert(len(getCnonce()), Equals, 16)
}

func (s *HTTPTestSuite) TestgetDigestAuthorization(c *C) {
	resp := &http.Response{}
	header := http.Header{}
	header.Add("Date", "Mon, 04 Jun 2018 14:33:09 GMT")
	header.Add("Content-Type", "text/html")
	header.Add("Www-Authenticate", `Digest algorithm="MD5", qop="auth", realm="testing", nonce="5b25940d5b154da5"`)
	resp.Header = header
	parts := digestDigestParts(resp)
	res := getDigestAuthorization(parts)
	c.Assert(res, Matches, `.*realm="testing", nonce="5b25940d5b154da5".*qop="auth".*`)
}

func (s *HTTPTestSuite) TestdigestDigestParts(c *C) {
	resp := &http.Response{}
	header := http.Header{}
	header.Add("Date", "Mon, 04 Jun 2018 14:33:09 GMT")
	header.Add("Content-Type", "text/html")
	header.Add("Www-Authenticate", `Digest algorithm="MD5", qop="auth", realm="testing", nonce="5b25940d5b154da5"`)
	resp.Header = header
	res := digestDigestParts(resp)
	c.Assert(res["realm"], Equals, "testing")
	c.Assert(res["qop"], Equals, "auth")
	c.Assert(res["nonce"], Equals, "5b25940d5b154da5")
	c.Assert(len(res), Equals, 3)
}
