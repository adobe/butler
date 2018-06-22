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

package config

import (
	"fmt"
)

var (
	ConfigCache map[string]map[string][]byte
)

type TmpFile struct {
	Name string
	File string
}

type RepoFileEvent struct {
	Success map[string]bool
	Error   map[string]error
	TmpFile map[string]string
}

func (r *RepoFileEvent) SetSuccess(file string, err error) error {
	r.Success[file] = true
	r.Error[file] = err
	return nil
}

func (r *RepoFileEvent) SetFailure(file string, err error) error {
	r.Success[file] = false
	r.Error[file] = err
	return nil
}

func (r *RepoFileEvent) SetTmpFile(file string, tmpfile string) error {
	r.TmpFile[file] = tmpfile
	return nil
}

type ConfigFileMap struct {
	TmpFile string
	Success bool
}

type ConfigSettings struct {
	Managers map[string]*Manager `json:"managers"`
	Globals  ConfigGlobals       `json:"globals"`
}

func (b *ConfigSettings) GetAllConfigLocalPaths(mgr string) []string {
	var result []string
	if _, ok := b.Managers[mgr]; !ok {
		return result
	}

	mopts := b.Managers[mgr]
	result = append(result, fmt.Sprintf("%s/%s", mopts.DestPath, mopts.PrimaryConfigName))
	for _, o := range mopts.ManagerOpts {
		for _, f := range o.AdditionalConfigsFullLocalPaths {
			result = append(result, f)
		}
	}
	return result
}

type ConfigGlobals struct {
	Managers             []string `mapstructure:"config-managers" json:"-"`
	SchedulerInterval    int      `json:"scheduler-interval"`
	CfgEnableHttpLog     string   `mapstructure:"enable-http-log" json:"-"`
	EnableHttpLog        bool     `json:"enable-http-log"`
	CfgSchedulerInterval string   `mapstructure:"scheduler-interval" json:"-"`
	CfgExitOnFailure     string   `mapstructure:"exit-on-config-failure" json:"-"`
	ExitOnFailure        bool     `json:"exit-on-failure"`
	CfgStatusFile        string   `mapstructure:"status-file" json:"-"`
	StatusFile           string   `json:"status-file"`
}

type ValidateOpts struct {
	ContentType string
	Data        interface{}
	FileName    string
	Manager     string
}

func NewValidateOpts() *ValidateOpts {
	return &ValidateOpts{ContentType: "text"}
}

func (o *ValidateOpts) WithContentType(t string) *ValidateOpts {
	o.ContentType = t
	return o
}

func (o *ValidateOpts) WithData(d interface{}) *ValidateOpts {
	o.Data = d
	return o
}

func (o *ValidateOpts) WithFileName(f string) *ValidateOpts {
	o.FileName = f
	return o
}

func (o *ValidateOpts) WithManager(m string) *ValidateOpts {
	o.Manager = m
	return o
}
