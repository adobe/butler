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
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/adobe/butler/internal/environment"

	"github.com/coreos/etcd/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type EtcdMethod struct {
	Endpoints             []string       `mapstructure:"endpoints" json:"endpoints"`
	CfgInsecureSkipVerify string         `mapstructure:"insecure-skip-verify" json:"-"`
	InsecureSkipVerify    bool           `json:"insecure-skip-verify"`
	KeysAPI               client.KeysAPI `json:"-"`
	Manager               *string        `json:"-"`
}

type EtcdMethodOpts struct {
	Endpoints []string
	Scheme    string
}

func getTransport(insecureSkipVerify bool) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}
}

func NewEtcdMethod(manager *string, entry *string) (Method, error) {
	var (
		err    error
		result EtcdMethod
	)
	if (manager != nil) && (entry != nil) {
		err = viper.UnmarshalKey(*entry, &result)
		if err != nil {
			return result, err
		}
		endpointsString := environment.GetVar(strings.Join(result.Endpoints, ","))
		if endpointsString == "" {
			return result, errors.New("endpoints is not defined for etcd")
		}
		result.Endpoints = strings.Split(endpointsString, ",")

		result.InsecureSkipVerify = strings.ToLower(environment.GetVar(result.CfgInsecureSkipVerify)) == "true"
		cfg := client.Config{
			Endpoints: result.Endpoints,
			Transport: getTransport(result.InsecureSkipVerify),
			// set timeout per request to fail fast when the target endpoint is unavailable
			HeaderTimeoutPerRequest: time.Second,
		}
		c, err := client.New(cfg)
		if err != nil {
			log.Fatal(err)
			return EtcdMethod{}, errors.New("could not start etcd client")
		}
		result.Manager = manager
		log.Debug("NewsKeyAPI configured for etcd")
		result.KeysAPI = client.NewKeysAPI(c)
	}

	return result, err
}

func NewEtcdMethodWithEndpoints(endpoints []string, insecureSkipVerify bool) (Method, error) {
	var (
		err    error
		result EtcdMethod
	)
	cfg := client.Config{
		Endpoints: endpoints,
		Transport: getTransport(insecureSkipVerify),
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
		return EtcdMethod{}, errors.New("could not start etcd client")
	}
	log.Debugf("NewsKeyAPI configured with Endpoints %v", endpoints)
	result.KeysAPI = client.NewKeysAPI(c)
	result.Endpoints = endpoints
	result.InsecureSkipVerify = insecureSkipVerify
	return result, err
}

func (e EtcdMethod) Get(u *url.URL) (*Response, error) {
	var (
		err      error
		response Response
	)
	// get path key's value
	log.Debugf("Getting file at %v", u)
	resp, err := GetEtcdKey(context.Background(), e, u.Path, nil)
	if err != nil {
		log.Warnf("Error getting key %s from etcd at %s", u.Path, e.Endpoints)
		return &Response{statusCode: 404}, err
	}
	response.statusCode = 200
	response.body = ioutil.NopCloser(bytes.NewReader([]byte(resp.Node.Value)))

	return &response, nil
}

func GetEtcdKey(ctx context.Context, e EtcdMethod, key string, opts *client.GetOptions) (*client.Response, error) {
	return e.KeysAPI.Get(ctx, key, opts)
}

func (o EtcdMethodOpts) GetScheme() string {
	return o.Scheme
}
