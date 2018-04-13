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
	"errors"
)

func NewGenericReloader(manager string, method string, entry []byte) (Reloader, error) {
	return GenericReloader{}, errors.New("Generic reloader is not very useful")
}

func NewGenericReloaderWithCustomError(manager string, method string, err error) (Reloader, error) {
	return GenericReloader{}, err
}

type GenericReloader struct {
	Opts GenericReloaderOpts
}

type GenericReloaderOpts struct {
}

func (r GenericReloader) Reload() error {
	var (
		res error
	)
	return res
}
func (r GenericReloader) GetMethod() string {
	return "none"
}
func (r GenericReloader) GetOpts() ReloaderOpts {
	return r.Opts
}

func (r GenericReloader) SetOpts(opts ReloaderOpts) bool {
	r.Opts = opts.(GenericReloaderOpts)
	return true
}
