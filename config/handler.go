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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/config/methods"
	"git.corp.adobe.com/TechOps-IAO/butler/config/reloaders"
	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

type ButlerConfig struct {
	Url                     *url.URL
	Client                  *ConfigClient
	Config                  *ConfigSettings
	FirstRun                bool
	LogLevel                log.Level
	PrevCMSchedulerInterval int
	Interval                int
	Timeout                 int
	RawConfig               []byte
	Retries                 int
	RetryWaitMin            int
	RetryWaitMax            int
	Scheduler               *gocron.Scheduler
	// some http specific stuff
	HttpAuthType  string
	HttpAuthUser  string
	HttpAuthToken string
	// some s3 specific stuff
	S3Region  string
	S3Bucket  string
	Endpoints []string
}

func (bc *ButlerConfig) SetScheme(s string) error {
	scheme := strings.ToLower(s)
	if !IsValidScheme(scheme) {
		errMsg := fmt.Sprintf("%s is an invalid scheme", scheme)
		log.Errorf("Config::SetScheme(): %s is an invalid scheme", scheme)
		return errors.New(errMsg)
	} else {
		log.Debugf("Config::SetScheme(): setting bc.Scheme=%s", scheme)
		bc.Url.Scheme = scheme
	}
	return nil
}

func (bc *ButlerConfig) SetPath(p string) error {
	newPath := filepath.Clean(p)
	log.Debugf("Config::SetPath(): setting bc.Path=%s", newPath)
	bc.Url.Path = newPath
	return nil
}

func (bc *ButlerConfig) GetPath() string {
	return bc.Url.Path
}

func (bc *ButlerConfig) SetInterval(t int) error {
	log.Debugf("Config::SetInterval(): setting bc.Interval=%v", t)
	bc.Interval = t
	return nil
}

func (bc *ButlerConfig) GetCMInterval() int {
	return bc.Config.Globals.SchedulerInterval
}

func (bc *ButlerConfig) SetCMInterval(i int) error {
	bc.Config.Globals.SchedulerInterval = i
	return nil
}

func (bc *ButlerConfig) GetCMPrevInterval() int {
	return bc.PrevCMSchedulerInterval
}

func (bc *ButlerConfig) SetCMPrevInterval(i int) error {
	bc.PrevCMSchedulerInterval = i
	return nil
}

func (bc *ButlerConfig) GetInterval() int {
	return bc.Interval
}

func (bc *ButlerConfig) SetTimeout(t int) error {
	log.Debugf("Config::SetTimeout(): setting bc.Timeout=%v", t)
	bc.Timeout = t
	return nil
}

func (bc *ButlerConfig) SetHttpAuthType(t string) error {
	bc.HttpAuthType = t
	return nil
}

func (bc *ButlerConfig) SetHttpAuthToken(t string) error {
	bc.HttpAuthToken = t
	return nil
}

func (bc *ButlerConfig) SetHttpAuthUser(t string) error {
	bc.HttpAuthUser = t
	return nil
}

func (bc *ButlerConfig) SetRegion(r string) error {
	bc.S3Region = r
	return nil
}

func (bc *ButlerConfig) SetEndpoints(e []string) error {
	bc.Endpoints = e
	return nil
}

func (bc *ButlerConfig) SetRegionAndBucket(r string, b string) error {
	if bc.Client != nil {
		method, err := methods.NewS3MethodWithRegionAndBucket(r, b)
		if err != nil {
			return err
		}
		log.Debugf("ButlerConfig::SetRegionAndBucket(): method=%#v", method)
		bc.Client.Method = method

	} else {
		return errors.New("ButlerConfig::SetRegionAndBucket(): bc.Client does not exist")
	}

	bc.S3Region = r
	bc.S3Bucket = b
	return nil
}

func (bc *ButlerConfig) SetRetries(t int) error {
	log.Debugf("Config::SetRetries(): setting bc.Retries=%v", t)
	bc.Retries = t
	return nil
}

func (bc *ButlerConfig) SetRetryWaitMin(t int) error {
	log.Debugf("Config::SetRetryWaitMin(): setting bc.RetryWaitMin=%v", t)
	bc.RetryWaitMin = t
	return nil
}

func (bc *ButlerConfig) SetRetryWaitMax(t int) error {
	log.Debugf("Config::SetRetryWaitMax(): setting bc.RetryWaitMax=%v", t)
	bc.RetryWaitMax = t
	return nil
}

func (bc *ButlerConfig) SetLogLevel(level log.Level) {
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	bc.LogLevel = level
	log.Debugf("Config::SetLogLevel(): setting log level to %s", level)
}

func (bc *ButlerConfig) GetLogLevel() log.Level {
	return bc.LogLevel
}

func (bc *ButlerConfig) Init() error {
	log.Infof("Config::Init(): initializing butler config.")
	var err error

	method, err := methods.New(nil, bc.Url.Scheme, nil)
	if err != nil {
		if err.Error() == "Generic method handler is not very useful" {
			log.Errorf("Config::Init(): could not initialize butler config (check if using valid scheme). err=%s", err.Error())
			return errors.New(fmt.Sprintf("\"%s\" is an invalid config retrieval method.", bc.Url.Scheme))

		} else {
			log.Errorf("Config::Init(): could not initialize butler config. err=%s", err.Error())
			return err
		}
	}
	log.Warnf("ButlerConfig::Init() Above \"NewHttpMethod(): could not convert\" warnings may be safely disregarded.")

	client, err := NewConfigClient(bc.Url.Scheme)
	if err != nil {
		log.Errorf("Config::Init(): could not initialize butler config. err=%s", err.Error())
		return err
	}
	client.Method = method

	switch bc.Url.Scheme {
	case "http", "https":
		client.SetTimeout(bc.Timeout)
		client.SetRetryMax(bc.Retries)
		client.SetRetryWaitMin(bc.RetryWaitMin)
		client.SetRetryWaitMax(bc.RetryWaitMax)
		// this is a bit hokey
		m := method.(methods.HttpMethod)
		m.AuthType = bc.HttpAuthType
		m.AuthToken = bc.HttpAuthToken
		m.AuthUser = bc.HttpAuthUser
		client.Method = m
	case "s3", "S3":
		pathSplit := strings.Split(bc.Url.Path, "/")
		bucket := pathSplit[0]
		bc.SetRegionAndBucket(bc.S3Region, bucket)
		client.Method, err = methods.NewS3MethodWithRegionAndBucket(bc.S3Region, bc.Url.Host)
		if err != nil {
			return err
		}
	case "file":
		client.Method, err = methods.NewFileMethodWithUrl(bc.Url)
		if err != nil {
			return err
		}
	case "blob":
		client.Method, err = methods.NewBlobMethodWithAccount(bc.Url.Host)
		if err != nil {
			return err
		}
	case "etcd":
		client.Method, err = methods.NewEtcdMethodWithEndpoints(bc.Endpoints)
		if err != nil {
			return err
		}
	}

	bc.Client = client
	bc.Config = NewConfigSettings()

	log.Infof("Config::Init(): butler config initialized.")
	return nil
}

func (bc *ButlerConfig) Handler() error {
	log.Infof("ButlerConfig::Handler(): entering.")
	response, err := bc.Client.Get(bc.Url)

	if err != nil {
		stats.SetButlerContactVal(stats.FAILURE, bc.Url.Host, bc.Url.Path)
		return err
	}
	defer response.GetResponseBody().Close()

	if response.GetResponseStatusCode() != 200 {
		stats.SetButlerContactVal(stats.FAILURE, bc.Url.Host, bc.Url.Path)
		errMsg := fmt.Sprintf("Did not receive 200 response code for %s. code=%d", bc.Url.String(), response.GetResponseStatusCode())
		return errors.New(errMsg)
	}

	body, err := ioutil.ReadAll(response.GetResponseBody())
	if err != nil {
		stats.SetButlerContactVal(stats.FAILURE, bc.Url.Host, bc.Url.Path)
		errMsg := fmt.Sprintf("Could not read response body for %s. err=%s", bc.Url.String(), err)
		return errors.New(errMsg)
	}

	err = ValidateConfig(NewValidateOpts().WithData(body).WithFileName("butler.toml"))
	if err != nil {
		stats.SetButlerContactVal(stats.FAILURE, bc.Url.Host, bc.Url.Path)
		return err
	}

	if bc.RawConfig == nil {
		err := bc.Config.ParseConfig(body)
		if err != nil {
			if bc.Config.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				stats.SetButlerContactVal(stats.FAILURE, bc.Url.Host, bc.Url.Path)
				return err
			}
		} else {
			log.Debugf("Config::Handler(): bc.RawConfig is nil. Filling it up.")
			bc.RawConfig = body
		}
	}

	if !bytes.Equal(bc.RawConfig, body) {
		err := bc.Config.ParseConfig(body)
		if err != nil {
			if bc.Config.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				stats.SetButlerContactVal(stats.FAILURE, bc.Url.Host, bc.Url.Path)
				return err
			}
		} else {
			log.Debugf("Config::Handler(): butler config has changed. updating.")
			bc.RawConfig = body
		}
	} else {
		if !bc.FirstRun {
			log.Infof("ButlerConfig::Handler(): butler config unchanged.")
		}
	}

	// We don't want to handle the scheduler stuff on the first run. The scheduler doesn't yet exist
	log.Debugf("Config::Handler(): CM PrevSchedulerInterval=%v SchedulerInterval=%v", bc.GetCMPrevInterval(), bc.GetCMInterval())

	// This is going to manage the CM scheduler. If it changes in the butler configuration, we should be aware of it.
	if bc.FirstRun {
		bc.FirstRun = false
	} else {
		// If we need to start the scheduler, then let's do that
		// If PrevInterval == 0, then no scheduler has been started
		if bc.GetCMPrevInterval() == 0 {
			log.Debugf("Config::Handler(): starting scheduler for RunCMHandler each %v seconds", bc.GetCMInterval())
			bc.Scheduler.Every(uint64(bc.GetCMInterval())).Seconds().Do(bc.RunCMHandler)
			bc.SetCMPrevInterval(bc.GetCMInterval())
		}
		// If PrevInterval is > 0 and the Intervals differ, then the configuration has changed.
		// We should restart the scheduler
		if (bc.GetCMPrevInterval() != 0) && (bc.GetCMPrevInterval() != bc.GetCMInterval()) {
			log.Debugf("Config::Handler(): butler CM interval has changed from %v to %v", bc.GetCMPrevInterval(), bc.GetCMInterval())
			log.Debugf("Config::Handler(): stopping current butler scheduler for RunCMHandler")
			bc.Scheduler.Remove(bc.RunCMHandler)
			log.Debugf("Config::Handler(): re-starting scheduler for RunCMHandler each %v seconds", bc.GetCMInterval())
			bc.Scheduler.Every(uint64(bc.GetCMInterval())).Seconds().Do(bc.RunCMHandler)
			bc.SetCMPrevInterval(bc.GetCMInterval())
		}
	}
	stats.SetButlerContactVal(stats.SUCCESS, bc.Url.Host, bc.Url.Path)
	return nil
}

func (bc *ButlerConfig) SetScheduler(s *gocron.Scheduler) error {
	log.Debugf("Config::SetScheduler(): entering")
	bc.Scheduler = s
	return nil
}

func (bc *ButlerConfig) RunCMHandler() error {
	var (
		ReloadManager []string
	)
	log.Infof("Config::RunCMHandler(): entering")

	c1 := make(chan ChanEvent)
	c2 := make(chan ChanEvent)

	bc.CheckPaths()

	for _, m := range bc.GetManagers() {
		go m.DownloadPrimaryConfigFiles(c1)
		go m.DownloadAdditionalConfigFiles(c2)
		PrimaryChan, AdditionalChan := <-c1, <-c2

		if PrimaryChan.CanCopyFiles() && AdditionalChan.CanCopyFiles() {
			log.Debugf("Config::RunCMHandler(): successfully retrieved files. processing...")
			p := PrimaryChan.CopyPrimaryConfigFiles(m.ManagerOpts)
			a := AdditionalChan.CopyAdditionalConfigFiles(m.DestPath)
			if p || a {
				ReloadManager = append(ReloadManager, m.Name)
			}
			PrimaryChan.CleanTmpFiles()
			AdditionalChan.CleanTmpFiles()
			stats.SetButlerRemoteRepoUp(stats.SUCCESS, m.Name)
			stats.SetButlerRemoteRepoSanity(stats.SUCCESS, m.Name)
		} else {
			log.Debugf("Config::RunCMHandler(): cannot copy files. cleaning up...")
			// Failure statistics for RemoteRepoUp and RemoteRepoSanity
			// happen in DownloadPrimaryConfigFiles // DownloadAdditionalConfigFiles
			PrimaryChan.CleanTmpFiles()
			AdditionalChan.CleanTmpFiles()
		}
		m.LastRun = time.Now()
	}

	if len(ReloadManager) == 0 {
		log.Infof("Config::RunCMHandler(): CM files unchanged... continuing.")
		// We are going to run through the managers and ensure that the status file
		// is in an OK state for the manager. If it is not, then we will attempt a reload
		for _, m := range bc.GetManagers() {
			stats.SetButlerRepoInSync(stats.SUCCESS, m.Name)
			if !GetManagerStatus(bc.GetStatusFile(), m.Name) {
				log.Debugf("Config::RunCMHandler(): Could not find manager status. Going to reload to get in sync.")
				err := m.Reload()
				if err != nil {
					switch e := err.(type) {
					case *reloaders.ReloaderError:
						// an http timeout is 1
						log.Debugf("Config::RunCMHandler(): e.Code=%#v, m.ManagerTimeoutOk=%#v", e.Code, m.ManagerTimeoutOk)
						if e.Code == 1 && m.ManagerTimeoutOk == true {
							// we really don't care about here
							// let's make sure we at least delete our metrics
							stats.DeleteButlerReloadVal(m.Name)
						} else {
							log.Errorf("Config::RunCMHandler(): err=%#v", err)
							err := SetManagerStatus(bc.GetStatusFile(), m.Name, false)
							if err != nil {
								log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
							}
							stats.SetButlerReloadVal(stats.FAILURE, m.Name)
							if m.EnableCache && m.GoodCache {
								RestoreCachedConfigs(m.Name, bc.Config.GetAllConfigLocalPaths(m.Name), m.CleanFiles)
							}
						}
					}
				} else {
					err := SetManagerStatus(bc.GetStatusFile(), m.Name, true)
					if err != nil {
						log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
					}
					stats.SetButlerReloadVal(stats.SUCCESS, m.Name)
					if m.EnableCache {
						CacheConfigs(m.Name, bc.Config.GetAllConfigLocalPaths(m.Name))
						m.GoodCache = true
					}
				}
			}
		}
	} else {
		log.Debugf("Config::RunCMHandler(): CM files changed... reloading.")
		for _, m := range ReloadManager {
			log.Debugf("Config::RunCMHandler(): m=%#v", m)
			mgr := bc.GetManager(m)
			err := mgr.Reload()
			if err != nil {
				switch e := err.(type) {
				case *reloaders.ReloaderError:
					log.Debugf("Config::RunCMHandler(): e.Code=%#v, mgr.ManagerTimeoutOk=%#v", e.Code, mgr.ManagerTimeoutOk)
					if e.Code == 1 && mgr.ManagerTimeoutOk == true {
						// we really don't care about here, but
						// let's make sure we at least delete our metrics
						stats.DeleteButlerReloadVal(mgr.Name)
					} else {
						log.Errorf("Config::RunCMHandler(): Could not reload manager \"%v\" err=%#v", mgr.Name, err)
						err := SetManagerStatus(bc.GetStatusFile(), m, false)
						if err != nil {
							log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
						}
						stats.SetButlerReloadVal(stats.FAILURE, m)
						if mgr.EnableCache && mgr.GoodCache {
							RestoreCachedConfigs(m, bc.Config.GetAllConfigLocalPaths(mgr.Name), mgr.CleanFiles)
						}
					}
				}
			} else {
				err := SetManagerStatus(bc.GetStatusFile(), m, true)
				if err != nil {
					log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
				}
				stats.SetButlerReloadVal(stats.SUCCESS, m)
				if mgr.EnableCache {
					CacheConfigs(m, bc.Config.GetAllConfigLocalPaths(mgr.Name))
					mgr.GoodCache = true
				}
			}
		}
	}

	return nil
}

func (bc *ButlerConfig) GetManagers() map[string]*Manager {
	return bc.Config.Managers
}

func (bc *ButlerConfig) GetManager(m string) *Manager {
	return bc.Config.Managers[m]
}

func (bc *ButlerConfig) GetStatusFile() string {
	return bc.Config.Globals.StatusFile
}

func (bc *ButlerConfig) CheckPaths() error {
	log.Debugf("Config::CheckPaths(): entering")
	for _, m := range bc.Config.Managers {
		for _, f := range m.GetAllLocalPaths() {
			dir := filepath.Dir(f)
			if _, err := os.Stat(dir); err != nil {
				err = os.MkdirAll(dir, 0755)
				if err != nil {
					msg := fmt.Sprintf("Config::CheckPaths(): err=%s", err.Error())
					log.Fatal(msg)
				}
				log.Infof("Config::CheckPaths(): Created directory \"%s\"", dir)
				log.Debugf("Config::CheckPaths(): setting m.ReloadManager=true")
				m.ReloadManager = true
			}
		}

		if m.CleanFiles {
			err := filepath.Walk(m.DestPath, m.PathCleanup)
			if err != nil {
				log.Debugf("Config::CheckPaths(): got err for filepath. setting m.ReloadManager=true")
				m.ReloadManager = true
			}
		}
	}
	return nil
}
