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
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/adobe/butler/internal/environment"

	//log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewFileMethod(manager *string, entry *string) (Method, error) {
	var (
		err    error
		result FileMethod
		u      *url.URL
	)

	u = &url.URL{}
	if (manager != nil) && (entry != nil) {
		err = viper.UnmarshalKey(*entry, &result)
		if err != nil {
			return result, err
		}
	}

	if environment.GetVar(result.Path) != "" {
		u.Path = environment.GetVar(result.Path)
	}
	result.Path = u.Path
	result.Url = u
	return result, err
}

func NewFileMethodWithUrl(u *url.URL) (Method, error) {
	var (
		err    error
		result FileMethod
	)

	result.Url = u
	result.Path = u.Path
	return result, err
}

type FileMethod struct {
	Url  *url.URL `json:"-"`
	Path string   `mapstructure:"path" json:"path"`
}

func (f FileMethod) Get(u *url.URL) (*Response, error) {
	var (
		err      error
		fileData []byte
		response Response
	)
	fileData, err = ioutil.ReadFile(fmt.Sprintf("%s%s", u.Host, u.Path))

	if err != nil {
		// 504 is hokey, but we need some bogus code.
		return &Response{statusCode: 504}, errors.New(fmt.Sprintf("FileMethod.Get(): caught error read file err=%v", err.Error()))
	}

	response.statusCode = 200
	response.body = ioutil.NopCloser(bytes.NewReader(fileData))
	return &response, nil
}
