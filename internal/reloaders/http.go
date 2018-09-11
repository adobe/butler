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

package reloaders

import (
	"encoding/json"
	//"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/adobe/butler/internal/environment"
	"github.com/adobe/butler/internal/stats"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

func NewHTTPReloader(manager string, method string, entry []byte) (Reloader, error) {
	var (
		err    error
		result HTTPReloader
		opts   HTTPReloaderOpts
	)

	err = json.Unmarshal(entry, &opts)
	if err != nil {
		return result, err
	}

	newTimeout, _ := strconv.Atoi(environment.GetVar(opts.Timeout))
	if newTimeout == 0 {
		log.Warnf("NewHttpReloader(): could not convert %v to integer for timeout, defaulting to 0. This is probably undesired.", opts.Timeout)
	}

	newRetries, _ := strconv.Atoi(environment.GetVar(opts.Retries))
	if newRetries == 0 {
		log.Warnf("NewHttpReloader(): could not convert %v to integer for retries, defaulting to 0. This is probably undesired.", opts.Retries)
	}

	newRetryWaitMax, _ := strconv.Atoi(environment.GetVar(opts.RetryWaitMax))
	if newRetryWaitMax == 0 {
		log.Warnf("NewHttpReloader(): could not convert %v to integer for retry-wait-max, defaulting to 0. This is probably undesired.", opts.RetryWaitMax)
	}

	newRetryWaitMin, _ := strconv.Atoi(environment.GetVar(opts.RetryWaitMin))
	if newRetryWaitMin == 0 {
		log.Warnf("NewHttpReloader(): could not convert %v to integer for retry-wait-min, defaulting to 0. This is probably undesired.", opts.RetryWaitMin)
	}

	opts.Client = retryablehttp.NewClient()
	opts.Client.Logger.SetFlags(0)
	opts.Client.Logger.SetOutput(ioutil.Discard)
	opts.Client.HTTPClient.Timeout = time.Duration(newTimeout) * time.Second
	opts.Client.RetryMax = newRetries
	opts.Client.RetryWaitMax = time.Duration(newRetryWaitMax) * time.Second
	opts.Client.RetryWaitMin = time.Duration(newRetryWaitMin) * time.Second

	// Let's populate some environment variables
	opts.Host = environment.GetVar(opts.Host)
	opts.ContentType = environment.GetVar(opts.ContentType)
	// we cannot do ints yet!
	//opts.Port
	opts.URI = environment.GetVar(opts.URI)
	opts.Method = environment.GetVar(opts.Method)
	opts.Payload = environment.GetVar(opts.Payload)

	result.Method = method
	result.Opts = opts
	result.Manager = manager

	//log.Debugf("NewHttpReloader(): opts=%#v", opts)
	return result, err
}

type HTTPReloader struct {
	Manager string           `json:"-"`
	Counter int              `json:"-"`
	Method  string           `mapstructure:"method" json:"method"`
	Opts    HTTPReloaderOpts `json:"opts"`
}

type HTTPReloaderOpts struct {
	Client       *retryablehttp.Client `json:"-"`
	ContentType  string                `json:"content-type"`
	Host         string                `json:"host"`
	Port         string                `mapstructure:"port" json:"port"`
	URI          string                `json:"uri"`
	Method       string                `json:"method"`
	Payload      string                `json:"payload"`
	Retries      string                `json:"retries"`
	RetryWaitMax string                `json:"retry-wait-max"`
	RetryWaitMin string                `json:"retry-wait-min"`
	Timeout      string                `json:"timeout"`
}

func (h *HTTPReloaderOpts) GetClient() *retryablehttp.Client {
	return h.Client
}

func (h HTTPReloader) Reload() error {
	var (
		err  error
		req  *retryablehttp.Request
		resp *http.Response
	)

	log.Debugf("HTTPReloader::Reload()[count=%v][manager=%v]: reloading manager using http", h.Counter, h.Manager)
	o := h.GetOpts().(HTTPReloaderOpts)
	c := o.GetClient()
	// Set the reloader retry policy
	c.CheckRetry = h.ReloaderRetryPolicy
	newPort, _ := strconv.Atoi(environment.GetVar(o.Port))
	if newPort == 0 {
		log.Warnf("HTTPReloader::Reload()[count=%v][manager=%v]: could not convert %v to integer for port, defaulting to 0. This is probably undesired.", h.Counter, h.Manager, o.Port)
	}
	reloadURL := fmt.Sprintf("%s://%s:%d%s", h.Method, o.Host, newPort, o.URI)

	switch o.Method {
	case "post", "put", "patch":
		req, err = retryablehttp.NewRequest(strings.ToUpper(o.Method), reloadURL, strings.NewReader(o.Payload))
		req.Header.Add("Content-Type", o.ContentType)
	default:
		req, err = retryablehttp.NewRequest(strings.ToUpper(o.Method), reloadURL, nil)
	}

	if err != nil {
		msg := fmt.Sprintf("HTTPReloader::Reload()[count=%v][manager=%v]: err=%v", h.Counter, h.Manager, err.Error())
		log.Errorf(msg)
		return NewReloaderError().WithMessage(err.Error()).WithCode(1)
	}

	log.Debugf("HTTPReloader::Reload()[count=%v][manager=%v]: %v'ing up!", h.Counter, h.Manager, o.Method)
	resp, err = c.Do(req)
	if err != nil {
		msg := fmt.Sprintf("HTTPReloader::Reload()[count=%v][manager=%v]: err=%v", h.Counter, h.Manager, err.Error())
		log.Errorf(msg)
		return NewReloaderError().WithMessage(err.Error()).WithCode(1)
	}
	if resp.StatusCode == 200 {
		log.Infof("HTTPReloader::Reload()[count=%v][manager=%v]: successfully reloaded config. http_code=%d", h.Counter, h.Manager, int(resp.StatusCode))
		// at this point error should be nil, so things are OK
	} else {
		msg := fmt.Sprintf("HTTPReloader::Reload()[count=%v][manager=%v]: received bad response from server. http_code=%d", h.Counter, h.Manager, int(resp.StatusCode))
		log.Errorf(msg)
		// at this point we should raise an error
		return NewReloaderError().WithMessage("received bad response from server").WithCode(resp.StatusCode)
	}

	return err
}

func (h *HTTPReloader) ReloaderRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		stats.SetButlerReloaderRetry(stats.SUCCESS, h.Manager)
		return true, err
	}

	// Here is our policy override. By default it looks for
	// res.StatusCode >= 500 ...
	if resp.StatusCode == 0 || resp.StatusCode >= 600 {
		stats.SetButlerReloaderRetry(stats.SUCCESS, h.Manager)
		return true, nil
	}
	return false, nil
}

func (h HTTPReloader) GetMethod() string {
	return h.Method
}
func (h HTTPReloader) GetOpts() ReloaderOpts {
	return h.Opts
}

func (h HTTPReloader) SetOpts(opts ReloaderOpts) bool {
	h.Opts = opts.(HTTPReloaderOpts)
	return true
}

func (h HTTPReloader) SetCounter(c int) Reloader {
	h.Counter = c
	return h
}
