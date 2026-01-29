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

package config

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/adobe/butler/internal/metrics"

	log "github.com/sirupsen/logrus"
)

// ChanEvent is the interface which gets passed along to the different
// retrieval mechanisms which ultimately keeps track of which files
// have been downlaoded, changed, etc.
type ChanEvent interface {
	CanCopyFiles() bool
	CleanTmpFiles() error
	GetTmpFileMap() []TmpFile
	SetSuccess(string, string, error) error
	SetTmpFile(string, string, string) error
	CopyPrimaryConfigFiles(map[string]*ManagerOpts) bool
	CopyAdditionalConfigFiles(string) bool
}

// ConfigChanEvent is the object passed around in the channel which contains
// information on whether the file has changed, the path to the tempfile,
// the config name, and repository event information
type ConfigChanEvent struct {
	HasChanged bool
	TmpFile    *os.File
	ConfigFile *string
	Manager    string
	Repo       map[string]*RepoFileEvent
}

// CanCopyFiles returns a boolean which tells whether or not butler is able to
// copy the files from the repository to the local filesystem. The assumption
// is that ALL files have to pass before butler attempts to copy over.
func (c *ConfigChanEvent) CanCopyFiles() bool {
	var (
		res bool
	)
	res = true

	log.Debugf("ConfigChanEvent::CanCopyFiles(): seeing if we can copy files")
	for _, r := range c.Repo {
		for _, v := range r.Success {
			if v == false {
				res = false
			}
		}
	}
	log.Debugf("ConfigChanEvent::CanCopyFiles(): returning %v", res)
	return res
}

// CleanTmpFiles returns an error, or not, depending on whether butler was able
// to delete all the tempfiles that were created during the config file
// retrieval from the remote repository
func (c *ConfigChanEvent) CleanTmpFiles() error {
	log.Debugf("ConfigChanEvent::CleanTmpFiles(): cleaning up temporary files")
	for _, r := range c.Repo {
		for _, f := range r.TmpFile {
			log.Debugf("ConfigChanEvent::CleanTmpFiles(): removing file %#v", f)
			os.Remove(f)
		}
	}

	if c.TmpFile != nil {
		os.Remove(c.TmpFile.Name())
	}
	return nil
}

// GetTmpFileMap returns a slice of SORTED TmpFile objects which contain the
// names of all the temp files creted during the config file retrieval from the
// remote repository.
func (c *ConfigChanEvent) GetTmpFileMap() []TmpFile {
	var (
		keys   []string
		res    []TmpFile
		tmpRes map[string]string
	)
	tmpRes = make(map[string]string)

	for _, r := range c.Repo {
		for k, v := range r.TmpFile {
			keys = append(keys, k)
			tmpRes[k] = v
		}
	}

	// Due to the way that golang handles the ordering of maps (random), we have to
	// enforce a sorted ordering, otherwise we may write config files differently,
	// but with the same data (eg: the merged primary config file), causing an undesired
	// configuration reload
	sort.Strings(keys)
	for _, v := range keys {
		res = append(res, TmpFile{Name: v, File: tmpRes[v]})
	}
	log.Debugf("ConfigChanEvent::GetTmpFileMap(): res=%#v", res)
	return res
}

// SetSuccess sets the value for the file argument in the repo argument to true
func (c *ConfigChanEvent) SetSuccess(repo string, file string, err error) error {
	// If c.Repo has not been initialized, do so.
	if c.Repo == nil {
		c.Repo = make(map[string]*RepoFileEvent)
	}
	if _, ok := c.Repo[repo]; !ok {
		rfe := &RepoFileEvent{}
		rfe.Success = make(map[string]bool)
		rfe.Error = make(map[string]error)
		rfe.TmpFile = make(map[string]string)
		c.Repo[repo] = rfe
	}
	c.Repo[repo].SetSuccess(file, err)
	return nil
}

// SetFailure sets the value for the file argument in the repo argument to false
func (c *ConfigChanEvent) SetFailure(repo string, file string, err error) error {
	// If c.Repo has not been initialized, do so.
	if c.Repo == nil {
		c.Repo = make(map[string]*RepoFileEvent)
	}
	if _, ok := c.Repo[repo]; !ok {
		rfe := &RepoFileEvent{}
		rfe.Success = make(map[string]bool)
		rfe.Error = make(map[string]error)
		rfe.TmpFile = make(map[string]string)
		c.Repo[repo] = rfe
	}
	c.Repo[repo].SetFailure(file, err)
	return nil
}

func (c *ConfigChanEvent) SetTmpFile(repo string, file string, tmpfile string) error {
	if _, ok := c.Repo[repo]; ok {
		c.Repo[repo].SetTmpFile(file, tmpfile)
	}
	return nil
}

func (c *ConfigChanEvent) CopyPrimaryConfigFiles(opts map[string]*ManagerOpts) bool {
	var (
		primaryConfigs []string
	)

	// need to get a list of the primary config files, in order from first to last, through each repo
	for _, opt := range opts {
		for _, config := range opt.PrimaryConfig {
			primaryConfigs = append(primaryConfigs, config)
		}
	}

	out, err := os.OpenFile(c.TmpFile.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %v err=%s.", c.ConfigFile, err.Error())
		metrics.SetButlerConfigVal(metrics.FAILURE, "local", metrics.GetStatsLabel(*c.ConfigFile))
		c.CleanTmpFiles()
		return false
	} else {
		// we need to go through each fo the primary config files in order, and then find the corresponding tmpMap entry
		for _, f := range primaryConfigs {
			for _, t := range c.GetTmpFileMap() {
				if t.Name == f {
					in, err := os.Open(t.File)
					if err != nil {
						log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %v err=%s.", c.ConfigFile, err.Error())
						metrics.SetButlerConfigVal(metrics.FAILURE, "local", metrics.GetStatsLabel(t.Name))
						c.CleanTmpFiles()
						out.Close()
						return false
					}
					_, err = io.Copy(out, in)
					if err != nil {
						log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %v err=%s.", c.ConfigFile, err.Error())
						metrics.SetButlerConfigVal(metrics.FAILURE, "local", metrics.GetStatsLabel(t.Name))
						c.CleanTmpFiles()
						out.Close()
						return false
					}
					in.Close()
					break
				}
			}
		}
	}
	out.Sync()
	out.Close()
	return CompareAndCopy(c.TmpFile.Name(), *c.ConfigFile, c.Manager)
}

func (c *ConfigChanEvent) CopyAdditionalConfigFiles(destDir string) bool {
	var (
		IsModified bool
	)
	log.Debugf("Manager::CopyAdditionalConfigFiles(): entering")
	IsModified = false

	for _, f := range c.GetTmpFileMap() {
		destFile := fmt.Sprintf("%s/%s", destDir, f.Name)
		if CompareAndCopy(f.File, destFile, c.Manager) {
			IsModified = true
		}
	}
	return IsModified
}
