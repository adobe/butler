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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/adobe/butler/internal/methods"
	"github.com/adobe/butler/internal/metrics"
	"github.com/adobe/butler/internal/reloaders"

	"strings"

	log "github.com/sirupsen/logrus"
)

type Manager struct {
	Name                   string                  `json:"name"`
	Repos                  []string                `mapstructure:"repos" json:"repos"`
	CfgCleanFiles          string                  `mapstructure:"clean-files" json:"-"`
	CleanFiles             bool                    `json:"clean-files"`
	GoodCache              bool                    `json:"good-cache"`
	LastRun                time.Time               `json:"last-run"`
	MustacheSubsArray      []string                `mapstructure:"mustache-subs" json:"-"`
	MustacheSubs           map[string]string       `json:"mustache-subs"`
	CfgEnableCache         string                  `mapstructure:"enable-cache" json:"-"`
	EnableCache            bool                    `json:"enable-cache"`
	CachePath              string                  `mapstructure:"cache-path" json:"cache-path"`
	DestPath               string                  `mapstructure:"dest-path" json:"dest-path"`
	PrimaryConfigName      string                  `mapstructure:"primary-config-name" json:"primary-config-name"`
	CfgManagerTimeoutOk    string                  `mapstructure:"manager-timeout-ok" json:"-"`
	ManagerTimeoutOk       bool                    `json:"manager-timeout-ok"`
	CfgSkipButlerHeader    string                  `mapstructure:"skip-butler-header" json:"-"`
	SkipButlerHeader       bool                    `json:"skip-butler-header"`
	CfgWatchOnly           string                  `mapstructure:"watch-only" json:"-"`
	WatchOnly              bool                    `json:"watch-only"`
	FileHashes             map[string]string       `json:"-"` // In-memory hash storage for watch-only mode
	ManagerOpts            map[string]*ManagerOpts `json:"opts"`
	Reloader               reloaders.Reloader      `mapstructure:"-" json:"reloader,omitempty"`
	ReloadManager          bool                    `json:"-"`
}

type ManagerOpts struct {
	Method                          string         `mapstructure:"method" json:"method"`
	RepoPath                        string         `mapstructure:"repo-path" json:"repo-path"`
	Repo                            string         `json:"repo"`
	PrimaryConfig                   []string       `mapstructure:"primary-config" json:"primary-config"`
	AdditionalConfig                []string       `mapstructure:"additional-config" json:"additional-config"`
	PrimaryConfigsFullURLs          []string       `json:"-"`
	AdditionalConfigsFullURLs       []string       `json:"-"`
	PrimaryConfigsFullLocalPaths    []string       `json:"-"`
	AdditionalConfigsFullLocalPaths []string       `json:"-"`
	ContentType                     string         `mapstructure:"content-type" json:"content-type"`
	Opts                            methods.Method `json:"opts"`
	parentManager                   string
}

func (bm *Manager) Reload() error {
	log.Debugf("Manager::Reload(): reloading %s manager...", bm.Name)
	if bm.Reloader == nil {
		log.Warnf("Manager::Reload(): No reloader defined for %s manager. Moving on...", bm.Name)
		return nil
	} else {
		return bm.Reloader.SetCounter(cmHandlerCounter).Reload()
	}
}

func (bm *Manager) DownloadPrimaryConfigFiles(c chan ChanEvent) error {
	var (
		Chan              *ConfigChanEvent
		PrimaryConfigName string
	)

	Chan = NewConfigChanEvent()
	Chan.Manager = bm.Name
	PrimaryConfigName = fmt.Sprintf("%s/%s", bm.DestPath, bm.PrimaryConfigName)
	Chan.ConfigFile = &PrimaryConfigName

	// Create a temporary file for the merged prometheus configurations.
	tmpFile, err := ioutil.TempFile("/tmp", "bcmsfile")
	if err != nil {
		msg := fmt.Sprintf("Manager::DownloadPrimaryConfigFiles(): Could not create temporary file . err=%s", err.Error())
		log.Fatal(msg)
	}
	Chan.TmpFile = tmpFile

	// Process the prometheus.yml configuration files
	// We are going to iterate through each of the potential managers configured
	for _, opts := range bm.ManagerOpts {
		for i, u := range opts.GetPrimaryConfigURLs() {
			log.Debugf("Manager::DownloadPrimaryConfigFiles(): i=%v, u=%v", i, u)
			log.Debugf("Manager::DownloadPrimaryConfigFiles(): f=%s", opts.GetPrimaryRemoteConfigFiles()[i])
			f := opts.DownloadConfigFile(u)
			if f == nil {
				metrics.SetButlerContactVal(metrics.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])

				// Set this metrics global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				metrics.SetButlerRemoteRepoUp(metrics.FAILURE, bm.Name)

				log.Debugf("Manager::DownloadPrimaryConfigFiles(): download for %s is nil.", u)
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not download file"))
				continue
			} else {
				metrics.SetButlerContactVal(metrics.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}
			Chan.SetTmpFile(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], f.Name())

			// For the prometheus.yml we have to do some mustache replacement on downloaded file
			// We are doing this before the header/footer check because YAML parsing doesn't like
			// the mustache entries... so we shuffled this around.
			if err := RenderConfigMustache(f, bm.MustacheSubs); err != nil {
				log.Errorf("%s for %s.", err.Error(), u)
				metrics.SetButlerRenderVal(metrics.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				metrics.SetButlerConfigVal(metrics.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				log.Debugf("Manager::DownloadPrimaryConfigFiles(): render for %s is nil.", opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not render file"))
				continue
			} else {
				// metrics
				metrics.SetButlerRenderVal(metrics.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				metrics.SetButlerConfigVal(metrics.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}

			// Let's ensure that the files starts with #butlerstart and
			// ends with #butlerend. If they do not, then we will assume
			// we did not get a correct configuration, or that there is an
			// issue with the upstream
			filename := opts.GetPrimaryRemoteConfigFiles()[i]
			if err := ValidateConfig(NewValidateOpts().WithContentType(opts.ContentType).WithFileName(filename).WithData(f).WithManager(bm.Name).WithSkipButlerHeader(bm.SkipButlerHeader)); err != nil {
				log.Errorf("%s for %s.", err.Error(), u)
				metrics.SetButlerConfigVal(metrics.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])

				// Set this metrics global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				metrics.SetButlerRemoteRepoSanity(metrics.FAILURE, bm.Name)

				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not validate file"))
				continue
			} else {
				metrics.SetButlerConfigVal(metrics.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}
		}
	}

	// Update the channel
	c <- Chan

	return nil
}

func (bm *Manager) DownloadAdditionalConfigFiles(c chan ChanEvent) error {
	var (
		Chan       *ConfigChanEvent
		IsModified bool
	)

	Chan = NewConfigChanEvent()
	Chan.Manager = bm.Name
	IsModified = false
	_ = IsModified

	// Process the additional configuration files
	for _, opts := range bm.ManagerOpts {
		for i, u := range opts.GetAdditionalConfigURLs() {
			log.Debugf("Manager::DownloadAdditionalConfigFiles(): i=%v, u=%v", i, u)
			f := opts.DownloadConfigFile(u)
			if f == nil {
				log.Debugf("Manager::DownloadAdditionalConfigFiles(): download for %s is nil.", u)
				metrics.SetButlerContactVal(metrics.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])

				// Set this metrics global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				metrics.SetButlerRemoteRepoUp(metrics.FAILURE, bm.Name)

				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not download file"))
				continue
			} else {
				metrics.SetButlerContactVal(metrics.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
				Chan.SetTmpFile(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], f.Name())
			}

			// Let's process some mustache ...
			// NOTE: We USED to do this only for the primary configuration. Unsure how this will
			// affect the additional configurations. we can remove this if there are adverse
			// effects.
			// We are doing this before the header/footer check because YAML parsing doesn't like
			// the mustache entries... so we shuffled this around.
			if err := RenderConfigMustache(f, bm.MustacheSubs); err != nil {
				metrics.SetButlerRenderVal(metrics.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				metrics.SetButlerConfigVal(metrics.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not render file"))
				continue
			} else {
				metrics.SetButlerRenderVal(metrics.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				metrics.SetButlerConfigVal(metrics.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
			}

			// Let's ensure that the files starts with #butlerstart and
			// ends with #butlerend. IF they do not, then we will assume
			// we did not get a correct configuration, or that there is an
			// issue with the upstream
			filename := opts.GetAdditionalRemoteConfigFiles()[i]
			if err := ValidateConfig(NewValidateOpts().WithContentType(opts.ContentType).WithFileName(filename).WithData(f).WithManager(bm.Name).WithSkipButlerHeader(bm.SkipButlerHeader)); err != nil {
				metrics.SetButlerConfigVal(metrics.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])

				// Set this metrics global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				metrics.SetButlerRemoteRepoSanity(metrics.FAILURE, bm.Name)

				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not validate file"))
				continue
			} else {
				metrics.SetButlerConfigVal(metrics.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
			}
		}
	}

	// Update the channel
	c <- Chan
	return nil
}

// PathCleanup
func (bm *Manager) PathCleanup(path string, f os.FileInfo, err error) error {
	var (
		Found bool
	)
	Found = false

	// We don't have to do anything with a directory
	if f.Mode().IsDir() {
		log.Debugf("Manager::PathCleanup(): %s is a directory... returning nil", f.Name())
		return nil
	}

	for _, file := range bm.GetAllLocalPaths() {
		if path == file {
			Found = true
		}
	}

	if !Found {
		message := fmt.Sprintf("Found unknown file \"%s\". deleting...", path)
		log.Debugf("Manager::PathCleanup(): Found unknown file \"%s\". deleting...", path)
		os.Remove(path)
		return errors.New(message)
	}
	return nil
}

func (bm *Manager) GetAllLocalPaths() []string {
	var result []string

	for _, opt := range bm.ManagerOpts {
		for _, f := range opt.PrimaryConfigsFullLocalPaths {
			result = append(result, f)
		}
		for _, f := range opt.AdditionalConfigsFullLocalPaths {
			result = append(result, f)
		}
	}
	return result
}

func (bmo *ManagerOpts) AppendPrimaryConfigURL(c string) error {
	log.Debugf("ManagerOpts::AppendPrimaryConfigURL(): adding %s to PrimaryConfigsURLs...", c)
	bmo.PrimaryConfigsFullURLs = append(bmo.PrimaryConfigsFullURLs, c)
	return nil
}

func (bmo *ManagerOpts) AppendPrimaryConfigFile(c string) error {
	log.Debugf("ManagerOpts::AppendPrimaryConfigFile(): adding %s to PrimaryConfigsFullLocalPaths...", c)
	bmo.PrimaryConfigsFullLocalPaths = append(bmo.PrimaryConfigsFullLocalPaths, c)
	return nil
}

func (bmo *ManagerOpts) AppendAdditionalConfigURL(c string) error {
	log.Debugf("ManagerOpts::AppendAdditionalConfigURL(): adding %s to AdditionalConfigsURLs...", c)
	bmo.AdditionalConfigsFullURLs = append(bmo.AdditionalConfigsFullURLs, c)
	return nil
}

func (bmo *ManagerOpts) AppendAdditionalConfigFile(c string) error {
	log.Debugf("ManagerOpts::AppendAdditionalConfigFile(): adding %s to AdditionalConfigsFullLocalPaths...", c)
	bmo.AdditionalConfigsFullLocalPaths = append(bmo.AdditionalConfigsFullLocalPaths, c)
	return nil
}

func (bmo *ManagerOpts) SetParentManager(c string) error {
	bmo.parentManager = c
	return nil
}

func (bmo *ManagerOpts) GetPrimaryConfigURLs() []string {
	return bmo.PrimaryConfigsFullURLs
}

func (bmo *ManagerOpts) GetPrimaryLocalConfigFiles() []string {
	return bmo.PrimaryConfigsFullLocalPaths
}

func (bmo *ManagerOpts) GetPrimaryRemoteConfigFiles() []string {
	return bmo.PrimaryConfig
}

func (bmo *ManagerOpts) GetAdditionalConfigURLs() []string {
	return bmo.AdditionalConfigsFullURLs
}

func (bmo *ManagerOpts) GetAdditionalLocalConfigFiles() []string {
	return bmo.AdditionalConfigsFullLocalPaths
}

func (bmo *ManagerOpts) GetAdditionalRemoteConfigFiles() []string {
	return bmo.AdditionalConfig
}

// Really need to come up with a better method for this.
func (bmo *ManagerOpts) DownloadConfigFile(file string) *os.File {
	if IsValidScheme(bmo.Method) {
		tmpFile, err := ioutil.TempFile("/tmp", "bcmsfile")
		if err != nil {
			msg := fmt.Sprintf("ManagerOpts::DownloadConfigFile()[count=%v][manager=%v]: could not create temporary file. err=%v", cmHandlerCounter, bmo.parentManager, err)
			log.Fatal(msg)
		}

		if (bmo.Method == "file") || (bmo.Method == "s3") {
			// the file argument for the Get()'ing configs are passed in like:
			// file://repo/full/path/to/file. We need to strip out file:// and
			// repo to get the actual path on the filesystem. So that is what
			// we are doing here.
			file = fmt.Sprintf("/%s", strings.Join(strings.Split(strings.Split(file, "://")[1], "/")[1:], "/"))
		}

		if bmo.Method == "blob" {
			// the file argument for the Get()'ing configs are passed in like:
			// blob://storageaccount/container/file. We need to strip out blob://
			file = strings.Split(file, "://")[1]
		}

		url, err := url.Parse(file)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile()[count=%v][manager=%v]: Could not parse file %s to *url.URL, err=%s", cmHandlerCounter, bmo.parentManager, file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		response, err := bmo.Opts.Get(url)

		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile()[count=%v][manager=%v]: Could not download from %s, err=%s", cmHandlerCounter, bmo.parentManager, file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		defer response.GetResponseBody().Close()
		defer tmpFile.Close()

		if response.GetResponseStatusCode() != 200 {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile()[count=%v][manager=%v]: Did not receive 200 response code for %s. code=%v", cmHandlerCounter, bmo.parentManager, file, response.GetResponseStatusCode())
			tmpFile = nil
			return tmpFile
		}

		_, err = io.Copy(tmpFile, response.GetResponseBody())
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile()[count=%v][manager=%v]: Could not copy to %s, err=%s", cmHandlerCounter, bmo.parentManager, file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		return tmpFile
	} else {
		return nil
	}
}
