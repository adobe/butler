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
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/adobe/butler/internal/environment"
	"github.com/adobe/butler/internal/stats"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	defaultRetryWaitMin = 5
	defaultRetryWaitMax = 15
	defaultRetries      = 5
	defaultTimeout      = 10
)

func NewHTTPMethod(manager *string, entry *string) (Method, error) {
	var (
		err    error
		result HTTPMethod
	)

	if (manager != nil) && (entry != nil) {
		err = viper.UnmarshalKey(*entry, &result)
		if err != nil {
			return result, err
		}
	}

	newTimeout, _ := strconv.Atoi(environment.GetVar(result.Timeout))
	if newTimeout == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for timeout, defaulting to %v. This is probably undesired.", result.Timeout, defaultTimeout)
		newTimeout = defaultTimeout
	}

	newRetries, _ := strconv.Atoi(environment.GetVar(result.Retries))
	if newRetries == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for retries, defaulting to %v. This is probably undesired.", result.Retries, defaultRetries)
		newRetries = defaultRetries
	}

	newRetryWaitMax, _ := strconv.Atoi(environment.GetVar(result.RetryWaitMax))
	if newRetryWaitMax == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for retry-wait-max, defaulting to %v. This is probably undesired.", result.RetryWaitMax, defaultRetryWaitMax)
		newRetryWaitMax = defaultRetryWaitMax
	}

	newRetryWaitMin, _ := strconv.Atoi(environment.GetVar(result.RetryWaitMin))
	if newRetryWaitMin == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for retry-wait-min, defaulting to %v. This is probably undesired.", result.RetryWaitMin, defaultRetryWaitMin)
		newRetryWaitMin = defaultRetryWaitMin
	}

	result.Client = retryablehttp.NewClient()
	result.Client.Logger.SetFlags(0)
	result.Client.Logger.SetOutput(ioutil.Discard)
	result.Client.HTTPClient.Timeout = time.Duration(newTimeout) * time.Second
	result.Client.RetryMax = newRetries
	result.Client.RetryWaitMax = time.Duration(newRetryWaitMax) * time.Second
	result.Client.RetryWaitMin = time.Duration(newRetryWaitMin) * time.Second
	result.Client.CheckRetry = result.MethodRetryPolicy
	result.Manager = manager
	return result, err
}

type HTTPMethod struct {
	Client       *retryablehttp.Client `json:"-"`
	Manager      *string               `json:"-"`
	Host         string                `mapstruecture:"host" json:"host,omitempty"`
	Retries      string                `mapstructure:"retries" json:"retries"`
	RetryWaitMax string                `mapstructure:"retry-wait-max" json:"retry-wait-max"`
	RetryWaitMin string                `mapstructure:"retry-wait-min" json:"retry-wait-min"`
	Timeout      string                `mapstructure:"timeout" json:"timeout"`
	AuthType     string                `mapstructure:"auth-type" json:"auth-type,omitempty"`
	AuthToken    string                `mapstructure:"auth-token" json:"-"`
	AuthUser     string                `mapstructure:"auth-user" json:"auth-user,omitempty"`
}

func (h HTTPMethod) Get(u *url.URL) (*Response, error) {
	var (
		err       error
		r         *http.Response
		res       Response
		authToken string
		authType  string
		authUser  string
	)

	req, err := retryablehttp.NewRequest("GET", u.String(), nil)
	if err != nil {
		return &Response{}, err
	}

	if h.AuthUser != "" && h.AuthToken != "" {
		authType = strings.ToLower(environment.GetVar(h.AuthType))
		if authType == "" {
			log.Debugf("HttpMethod::Get(): found authentication tokens but auth-type is empty. Setting auth-type to basic.")
			authType = "basic"
		}
		authUser = environment.GetVar(h.AuthUser)
		authToken = environment.GetVar(h.AuthToken)
	}

	switch authType {
	case "basic":
		req.Header.Set("Authorization", getBasicAuthorization(authUser, authToken))
	case "digest":
		r, err = h.Client.Do(req)
		if err != nil {
			return &Response{}, err
		}
		if r.StatusCode == http.StatusUnauthorized {
			digestParts := digestDigestParts(r)
			digestParts["uri"] = u.Path
			digestParts["method"] = "GET"
			digestParts["username"] = authUser
			digestParts["password"] = authToken
			req.Header.Set("Authorization", getDigestAuthorization(digestParts))
		} else {
			res.body = r.Body
			res.statusCode = r.StatusCode
			return &res, err
		}
	case "token-key":
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s, key=%s", authToken, authUser))
	default:
		break
	}

	r, err = h.Client.Do(req)
	if err != nil {
		return &Response{}, err
	}
	res.body = r.Body
	res.statusCode = r.StatusCode
	return &res, err
}

func (h *HTTPMethod) MethodRetryPolicy(resp *http.Response, err error) (bool, error) {
	// This is actually the default RetryPolicy from the go-retryablehttp library. The only
	// change is the stats monitor. We want to keep track of all the reload failures.
	if (err != nil) && (h.Manager != nil) {
		opErr := err.(*url.Error)
		stats.SetButlerContactRetryVal(stats.SUCCESS, *h.Manager, stats.GetStatsLabel(opErr.URL))
		return true, err
	}

	// resp can be nil if the remote host is unreachble of otherwise does not exist
	if resp != nil {
		// Check the response code. We retry on 500-range responses to allow
		// the server time to recover, as 500's are typically not permanent
		// errors and may relate to outages on the server side. This will catch
		// invalid response codes as well, like 0 and 999.
		if resp.StatusCode == 0 || resp.StatusCode >= 500 {
			if h.Manager != nil {
				stats.SetButlerContactRetryVal(stats.SUCCESS, *h.Manager, stats.GetStatsLabel(resp.Request.RequestURI))
			}
			return true, nil
		}
	}

	return false, nil
}

func getBasicAuthorization(authUser string, authToken string) string {
	auth := authUser + ":" + authToken
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func getMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCnonce() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)[:16]
}

func getDigestAuthorization(digestParts map[string]string) string {
	ha1 := getMD5(digestParts["username"] + ":" + digestParts["realm"] + ":" + digestParts["password"])
	ha2 := getMD5(digestParts["method"] + ":" + digestParts["uri"])
	nonceCount := 00000001
	cnonce := getCnonce()
	response := getMD5(fmt.Sprintf("%s:%s:%v:%s:%s:%s", ha1, digestParts["nonce"], nonceCount, cnonce, digestParts["qop"], ha2))
	authorization := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%v", qop="%s", response="%s"`,
		digestParts["username"], digestParts["realm"], digestParts["nonce"], digestParts["uri"], cnonce, nonceCount, digestParts["qop"],
		response)
	return authorization
}

func digestDigestParts(resp *http.Response) map[string]string {
	result := map[string]string{}
	if len(resp.Header["Www-Authenticate"]) > 0 {
		wantedHeaders := []string{"nonce", "realm", "qop"}
		responseHeaders := strings.Split(resp.Header["Www-Authenticate"][0], ",")
		for _, r := range responseHeaders {
			for _, w := range wantedHeaders {
				if strings.Contains(r, w) {
					result[w] = strings.Split(r, `"`)[1]
				}
			}
		}
	}
	return result
}
