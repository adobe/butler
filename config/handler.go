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

	if bc.Url == "" {
		ConfigUrl := fmt.Sprintf("%s://%s", bc.Scheme, bc.Path)
		if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
			log.Debugf("Config::Init(): could not initialize butler config. err=%s", err.Error())
			return err
		}
		bc.Url = ConfigUrl
	}

	c, err := NewConfigClient(bc.Scheme)
	if err != nil {
		log.Debugf("Config::Init(): could not initialize butler config. err=%s", err.Error())
		return err
	}

	bc.Client = c
	bc.Client.SetTimeout(bc.Timeout)
	bc.Client.SetRetryMax(bc.Retries)
	bc.Client.SetRetryWaitMin(bc.RetryWaitMin)
	bc.Client.SetRetryWaitMax(bc.RetryWaitMax)

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
	defer response.Body.Close()

	if response.StatusCode != 200 {
		errMsg := fmt.Sprintf("Did not receive 200 response code for %s. code=%d", bc.Url, response.StatusCode)
		return errors.New(errMsg)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		errMsg := fmt.Sprintf("Could not read response body for %s. err=%s", bc.Url, err)
		return errors.New(errMsg)
	}

	err = ValidateConfig(body)
	if err != nil {
		return err
	}

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

	if !bytes.Equal(bc.RawConfig, body) {
		err := bc.Config.ParseConfig(body)
		if err != nil {
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

	for _, m := range bc.Config.Managers {
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
		} else {
			log.Debugf("Config::RunCMHandler(): cannot copy files. cleaning up...")
			PrimaryChan.CleanTmpFiles()
			AdditionalChan.CleanTmpFiles()
		}
	}

	if len(ReloadManager) == 0 {
		log.Debugf("Config::RunCMHandler(): CM files unchanged... continuing.")
	} else {
		for _, m := range ReloadManager {
			log.Debugf("Config::RunCMHandler(): m=%#v", m)
			err := bc.Config.Managers[m].Reload()
			log.Debugf("Config::RunCMHandler(): err=%#v", err)
			if err != nil {
				stats.SetButlerReloadVal(stats.FAILURE, m)
				if bc.Config.Managers[m].EnableCache {
					RestoreCachedConfigs(m, bc.Config.GetAllConfigLocalPaths())
				}
			} else {
				stats.SetButlerReloadVal(stats.SUCCESS, m)
				if bc.Config.Managers[m].EnableCache {
					CacheConfigs(m, bc.Config.GetAllConfigLocalPaths())
				}
			}
		}
	}

	return nil
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
