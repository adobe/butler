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
	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

type ButlerConfig struct {
	Url                     string
	Client                  *ConfigClient
	Config                  *ConfigSettings
	FirstRun                bool
	LogLevel                log.Level
	PrevCMSchedulerInterval int
	Interval                int
	Path                    string
	Scheme                  string
	Timeout                 int
	RawConfig               []byte
	Retries                 int
	RetryWaitMin            int
	RetryWaitMax            int
	Scheduler               *gocron.Scheduler
	// some s3 specific stuff
	S3Region string
	S3Bucket string
}

func (bc *ButlerConfig) SetScheme(s string) error {
	scheme := strings.ToLower(s)
	if !IsValidScheme(scheme) {
		errMsg := fmt.Sprintf("%s is an invalid scheme", scheme)
		log.Debugf("Config::SetScheme(): %s is an invalid scheme", scheme)
		return errors.New(errMsg)
	} else {
		log.Debugf("Config::SetScheme(): setting bc.Scheme=%s", scheme)
		bc.Scheme = scheme
	}
	return nil
}

func (bc *ButlerConfig) SetPath(p string) error {
	log.Debugf("Config::SetPath(): setting bc.Path=%s", p)
	bc.Path = p
	return nil
}

func (bc *ButlerConfig) GetPath() string {
	log.Debugf("Config::GetPath(): getting bc.Path=%s", bc.Path)
	return bc.Path
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

func (bc *ButlerConfig) SetRegion(r string) error {
	bc.S3Region = r
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

func (bc *ButlerConfig) SetUrl(u string) error {
	log.Debugf("Config::SetUrl(): setting bc.Url=%s", u)
	bc.Url = u
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
	log.Debugf("Config::Init(): initializing butler config.")
	var err error

	method, err := methods.New(nil, bc.Scheme, nil)
	if err != nil {
		log.Debugf("Config::Init(): could not initialize butler config. err=%s", err.Error())
		return err
	}

	bc.Client, err = NewConfigClient(bc.Scheme)
	if err != nil {
		log.Debugf("Config::Init(): could not initialize butler config. err=%s", err.Error())
		return err
	}
	bc.Client.Method = method

	switch bc.Scheme {
	case "http", "https":
		if bc.Url == "" {
			ConfigUrl := fmt.Sprintf("%s://%s", bc.Scheme, bc.Path)
			if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
				log.Debugf("Config::Init(): could not initialize butler config. err=%s", err.Error())
				return err
			}
			bc.Url = ConfigUrl
		}

		bc.Client.SetTimeout(bc.Timeout)
		bc.Client.SetRetryMax(bc.Retries)
		bc.Client.SetRetryWaitMin(bc.RetryWaitMin)
		bc.Client.SetRetryWaitMax(bc.RetryWaitMax)
	case "s3", "S3":
		pathSplit := strings.Split(bc.Path, "/")
		bucket := pathSplit[0]
		path := strings.Join(pathSplit[1:len(pathSplit)], "/")

		bc.Url = path
		bc.SetRegionAndBucket(bc.S3Region, bucket)
	}
	log.Debugf("Config::Init(): bc.Client.Method=%#v", bc.Client.Method)

	bc.Config = NewConfigSettings()

	log.Debugf("Config::Init(): butler config initialized.")
	return nil
}

func (bc *ButlerConfig) Handler() error {
	log.Debugf("Config::Handler(): entering.")
	response, err := bc.Client.Get(bc.Url)
	if err != nil {
		return err
	}
	defer response.GetResponseBody().Close()
	log.Debugf("Config::Handler(): test 1")

	if response.GetResponseStatusCode() != 200 {
		errMsg := fmt.Sprintf("Did not receive 200 response code for %s. code=%d", bc.Url, response.GetResponseStatusCode())
		return errors.New(errMsg)
	}
	log.Debugf("Config::Handler(): test 2")

	body, err := ioutil.ReadAll(response.GetResponseBody())
	if err != nil {
		errMsg := fmt.Sprintf("Could not read response body for %s. err=%s", bc.Url, err)
		return errors.New(errMsg)
	}
	log.Debugf("Config::Handler(): test 3")

	err = ValidateConfig(body)
	if err != nil {
		return err
	}
	log.Debugf("Config::Handler(): test 4")

	if bc.RawConfig == nil {
		err := bc.Config.ParseConfig(body)
		if err != nil {
			if bc.Config.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				return err
			}
		} else {
			log.Debugf("Config::Handler(): bc.RawConfig is nil. Filling it up.")
			bc.RawConfig = body
		}
	}
	log.Debugf("Config::Handler(): test 5")

	if !bytes.Equal(bc.RawConfig, body) {
		err := bc.Config.ParseConfig(body)
		if err != nil {
			log.Debugf("Config::Handler(): test bc.Config.Globals=%#v", bc.Config.Globals)
			if bc.Config.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				return err
			}
		} else {
			log.Debugf("Config::Handler(): butler config has changed. updating.")
			bc.RawConfig = body
		}
	} else {
		log.Debugf("Config::Handler(): butler config unchanged.")
	}
	log.Debugf("Config::Handler(): test 6")

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
	log.Debugf("Config::RunCMHandler(): entering")

	c1 := make(chan ChanEvent)
	c2 := make(chan ChanEvent)

	bc.CheckPaths()

	for _, m := range bc.GetManagers() {
		go m.DownloadPrimaryConfigFiles(c1)
		go m.DownloadAdditionalConfigFiles(c2)
		PrimaryChan, AdditionalChan := <-c1, <-c2

		if PrimaryChan.CanCopyFiles() && AdditionalChan.CanCopyFiles() {
			log.Debugf("Config::RunCMHandler(): successfully retrieved files. processing...")
			p := PrimaryChan.CopyPrimaryConfigFiles()
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
		log.Debugf("Config::RunCMHandler(): CM files unchanged... continuing.")
		// We are going to run through the managers and ensure that the status file
		// is in an OK state for the manager. If it is not, then we will attempt a reload
		for _, m := range bc.GetManagers() {
			stats.SetButlerRepoInSync(stats.SUCCESS, m.Name)
			if !GetManagerStatus(bc.GetStatusFile(), m.Name) {
				log.Debugf("Config::RunCMHandler(): Could not find manager status. Going to reload to get in sync.")
				err := m.Reload()
				log.Debugf("Config::RunCMHandler(): err=%#v", err)
				if err != nil {
					err := SetManagerStatus(bc.GetStatusFile(), m.Name, false)
					if err != nil {
						log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
					}
					stats.SetButlerReloadVal(stats.FAILURE, m.Name)
					if m.EnableCache {
						RestoreCachedConfigs(m.Name, bc.Config.GetAllConfigLocalPaths())
					}
				} else {
					err := SetManagerStatus(bc.GetStatusFile(), m.Name, true)
					if err != nil {
						log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
					}
					stats.SetButlerReloadVal(stats.SUCCESS, m.Name)
					if m.EnableCache {
						CacheConfigs(m.Name, bc.Config.GetAllConfigLocalPaths())
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
			log.Debugf("Config::RunCMHandler(): err=%#v", err)
			if err != nil {
				err := SetManagerStatus(bc.GetStatusFile(), m, false)
				if err != nil {
					log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
				}
				stats.SetButlerReloadVal(stats.FAILURE, m)
				if mgr.EnableCache {
					RestoreCachedConfigs(m, bc.Config.GetAllConfigLocalPaths())
				}
			} else {
				err := SetManagerStatus(bc.GetStatusFile(), m, true)
				if err != nil {
					log.Fatalf("Config::RunCMHandler(): could not write to %v err=%v", bc.GetStatusFile(), err.Error())
				}
				stats.SetButlerReloadVal(stats.SUCCESS, m)
				if mgr.EnableCache {
					CacheConfigs(m, bc.Config.GetAllConfigLocalPaths())
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
