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
	"errors"
	"fmt"

	//log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Reloader interface {
	Reload() error
	GetMethod() string
	GetOpts() ReloaderOpts
	SetOpts(ReloaderOpts) bool
	SetCounter(int) Reloader
}

type ReloaderOpts interface {
}

func New(entry string) (Reloader, error) {
	var (
		err    error
		result map[string]interface{}
	)

	key := fmt.Sprintf("%s.reloader", entry)

	err = viper.UnmarshalKey(key, &result)
	if err != nil {
		return NewGenericReloader(entry, "error", []byte(entry))
	}

	// No reloader has been defined. We'll assume that is OK
	// but will let the upstream know and they can handle it
	if result == nil {
		return NewGenericReloaderWithCustomError(entry, "error", errors.New("no reloader has been defined for manager"))
	}

	// reloader is defined, but there's no method
	if result["method"] == nil {
		return NewGenericReloaderWithCustomError(entry, "error", errors.New("no reloader has been defined for manager"))
	}

	method := result["method"].(string)
	jsonRes, err := json.Marshal(result[method])
	if err != nil {
		return NewGenericReloader(entry, method, []byte(entry))
	}

	// there are no reloader configuration options for the specified method
	if _, ok := result[method]; !ok {
		return NewGenericReloaderWithCustomError(entry, "error", errors.New("no reloader configuration has been defined for manager"))
	}

	switch method {
	case "http", "https":
		return NewHttpReloader(entry, method, jsonRes)
	default:
		return NewGenericReloader(entry, method, jsonRes)
	}
}

func NewReloaderError() *ReloaderError {
	return &ReloaderError{}
}

func (r *ReloaderError) WithCode(c int) *ReloaderError {
	r.Code = c
	return r
}

func (r *ReloaderError) WithMessage(m string) *ReloaderError {
	r.Message = m
	return r
}

func (r *ReloaderError) Error() string {
	msg := fmt.Sprintf("%v. code=%v", r.Message, r.Code)
	return msg
}

type ReloaderError struct {
	Code    int
	Message string
}
