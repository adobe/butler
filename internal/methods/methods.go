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
	//log "github.com/sirupsen/logrus"
	"io"
	"net/url"
	"strings"
)

type Method interface {
	Get(*url.URL) (*Response, error)
}

type MethodOpts interface {
	GetScheme() string
}

type Response struct {
	body       io.ReadCloser
	statusCode int
}

func (r Response) GetResponseBody() io.ReadCloser {
	return r.body
}

func (r Response) GetResponseStatusCode() int {
	return r.statusCode
}

func New(manager *string, method string, entry *string) (Method, error) {
	method = strings.ToLower(method)
	switch method {
	case "http", "https":
		return NewHTTPMethod(manager, entry)
	case "s3":
		return NewS3Method(manager, entry)
	case "file":
		return NewFileMethod(manager, entry)
	case "blob":
		return NewBlobMethod(manager, entry)
	case "etcd":
		return NewEtcdMethod(manager, entry)
	default:
		return NewGenericMethod(manager, entry)
	}
}
