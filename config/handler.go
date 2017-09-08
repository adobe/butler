package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

func NewButlerConfig() *ButlerConfig {
	return &ButlerConfig{FirstRun: true}
}

func (bc *ButlerConfig) SetScheme(s string) error {
	scheme := strings.ToLower(s)
	if !IsValidScheme(scheme) {
		errMsg := fmt.Sprintf("%s is an invalid scheme", scheme)
		log.Debugf("ButlerConfig::SetScheme(): %s is an invalid scheme", scheme)
		return errors.New(errMsg)
	} else {
		log.Debugf("ButlerConfig::SetScheme(): setting bc.Scheme=%s", scheme)
		bc.Scheme = scheme
	}
	return nil
}

func (bc *ButlerConfig) SetPath(p string) error {
	log.Debugf("ButlerConfig::SetPath(): setting bc.Path=%s", p)
	bc.Path = p
	return nil
}

func (bc *ButlerConfig) GetPath() string {
	log.Debugf("ButlerConfig::GetPath(): getting bc.Path=%s", bc.Path)
	return bc.Path
}

func (bc *ButlerConfig) SetInterval(t int) error {
	log.Debugf("ButlerConfig::SetInterval(): setting bc.Interval=%v", t)
	bc.Interval = t
	return nil
}

func (bc *ButlerConfig) GetCMInterval() int {
	//log.Debugf("ButlerConfig::GetCMInterval(): getting bc.Config.Globals.SchedulerInterval=%v", bc.Config.Globals.SchedulerInterval)
	return bc.Config.Globals.SchedulerInterval
}

func (bc *ButlerConfig) SetCMInterval(i int) error {
	//log.Debugf("ButlerConfig::SetCMInterval(): setting bc.Config.Globals.SchedulerInterval=%v", i)
	bc.Config.Globals.SchedulerInterval = i
	return nil
}

func (bc *ButlerConfig) GetCMPrevInterval() int {
	//log.Debugf("ButlerConfig::GetCMPrevInterval(): getting bc.Config.Globals.PrevSchedulerInterval=%v", bc.Config.Globals.PrevSchedulerInterval)
	return bc.PrevCMSchedulerInterval
}

func (bc *ButlerConfig) SetCMPrevInterval(i int) error {
	//log.Debugf("ButlerConfig::SetCMPrevInterval(): setting bc.Config.Globals.PrevSchedulerInterval=%v", i)
	bc.PrevCMSchedulerInterval = i
	return nil
}

func (bc *ButlerConfig) GetInterval() int {
	//log.Debugf("ButlerConfig::GetInterval(): getting bc.Interval=%v", bc.Interval)
	return bc.Interval
}

func (bc *ButlerConfig) SetTimeout(t int) error {
	log.Debugf("ButlerConfig::SetTimeout(): setting bc.Timeout=%v", t)
	bc.Timeout = t
	return nil
}

func (bc *ButlerConfig) SetRetries(t int) error {
	log.Debugf("ButlerConfig::SetRetries(): setting bc.Retries=%v", t)
	bc.Retries = t
	return nil
}

func (bc *ButlerConfig) SetRetryWaitMin(t int) error {
	log.Debugf("ButlerConfig::SetRetryWaitMin(): setting bc.RetryWaitMin=%v", t)
	bc.RetryWaitMin = t
	return nil
}

func (bc *ButlerConfig) SetRetryWaitMax(t int) error {
	log.Debugf("ButlerConfig::SetRetryWaitMax(): setting bc.RetryWaitMax=%v", t)
	bc.RetryWaitMax = t
	return nil
}

func (bc *ButlerConfig) SetUrl(u string) error {
	log.Debugf("ButlerConfig::SetwUrl(): setting bc.Url=%s", u)
	bc.Url = u
	return nil
}

func (bc *ButlerConfig) SetLogLevel(level log.Level) {
	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	bc.LogLevel = level
	log.Debugf("ButlerConfig::SetLogLevel(): setting log level to %s", level)
}

func (bc *ButlerConfig) GetLogLevel() log.Level {
	return bc.LogLevel
}

func (bc *ButlerConfig) Init() error {
	log.Debugf("ButlerConfig::Init(): initializing butler config.")
	var err error

	if bc.Url == "" {
		ConfigUrl := fmt.Sprintf("%s://%s", bc.Scheme, bc.Path)
		if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
			log.Debugf("ButlerConfig::Init(): could not initialize butler config. err=%s", err)
			return err
		}
		bc.Url = ConfigUrl
	}

	c, err := NewButlerConfigClient(bc.Scheme)
	if err != nil {
		log.Debugf("ButlerConfig::Init(): could not initialize butler config. err=%s", err)
		return err
	}

	bc.Client = c
	bc.Client.SetTimeout(bc.Timeout)
	bc.Client.SetRetryMax(bc.Retries)
	bc.Client.SetRetryWaitMin(bc.RetryWaitMin)
	bc.Client.SetRetryWaitMax(bc.RetryWaitMax)

	bc.Config = NewButlerConfigSettings()

	log.Debugf("ButlerConfig::Init(): butler config initialized.")
	return nil
}

func (bc *ButlerConfig) Handler() error {
	log.Debugf("ButlerConfig::Handler(): entering.")
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

	err = ValidateButlerConfig(body)
	if err != nil {
		return err
	}

	if bc.RawConfig == nil {
		err := bc.Config.ParseButlerConfig(body)
		if err != nil {
			if bc.Config.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				return err
			}
		} else {
			log.Debugf("ButlerConfig::Handler(): bc.RawConfig is nil. Filling it up.")
			bc.RawConfig = body
		}
	}

	if !bytes.Equal(bc.RawConfig, body) {
		err := bc.Config.ParseButlerConfig(body)
		if err != nil {
			if bc.Config.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				return err
			}
		} else {
			log.Debugf("ButlerConfig::Handler(): butler config has changed. updating.")
			bc.RawConfig = body
		}
	} else {
		log.Debugf("ButlerConfig::Handler(): butler config unchanged.")
	}

	// We don't want to handle the scheduler stuff on the first run. The scheduler doesn't yet exist
	log.Debugf("ButlerConfig::Handler(): CM PrevSchedulerInterval=%v SchedulerInterval=%v", bc.GetCMPrevInterval(), bc.GetCMInterval())

	// This is going to manage the CM scheduler. If it changes in the butler configuration, we should be aware of it.
	if bc.FirstRun {
		bc.FirstRun = false
	} else {
		// If we need to start the scheduler, then let's do that
		// If PrevInterval == 0, then no scheduler has been started
		if bc.GetCMPrevInterval() == 0 {
			log.Debugf("ButlerConfig::Handler(): starting scheduler for RunCMHandler each %v seconds", bc.GetCMInterval())
			bc.Scheduler.Every(uint64(bc.GetCMInterval())).Seconds().Do(bc.RunCMHandler)
			bc.SetCMPrevInterval(bc.GetCMInterval())
		}
		// If PrevInterval is > 0 and the Intervals differ, then the configuration has changed.
		// We should restart the scheduler
		if (bc.GetCMPrevInterval() != 0) && (bc.GetCMPrevInterval() != bc.GetCMInterval()) {
			log.Debugf("ButlerConfig::Handler(): butler CM interval has changed from %v to %v", bc.GetCMPrevInterval(), bc.GetCMInterval())
			log.Debugf("ButlerConfig::Handler(): stopping current butler scheduler for RunCMHandler")
			bc.Scheduler.Remove(bc.RunCMHandler)
			log.Debugf("ButlerConfig::Handler(): re-starting scheduler for RunCMHandler each %v seconds", bc.GetCMInterval())
			bc.Scheduler.Every(uint64(bc.GetCMInterval())).Seconds().Do(bc.RunCMHandler)
			bc.SetCMPrevInterval(bc.GetCMInterval())
		}
	}
	return nil
}

func (bc *ButlerConfig) SetScheduler(s *gocron.Scheduler) error {
	log.Debugf("ButlerConfig::SetScheduler(): entering")
	bc.Scheduler = s
	return nil
}

func (bc *ButlerConfig) RunCMHandler() error {
	log.Debugf("ButlerConfig::RunCMHandler(): entering")
	c := make(chan bool)
	_ = c

	bc.CheckPaths()

	for _, m := range bc.Config.Managers {
		go m.ProcessPrimaryConfigFiles(c)
		go m.ProcessAdditionalConfigFiles(c)
	}
	return nil
}

func (bc *ButlerConfig) CheckPaths() error {
	log.Debugf("ButlerConfig::CheckPaths(): entering")
	for _, m := range bc.Config.Managers {
		log.Debugf("ButlerConfig::CheckPaths(): m=%#v", m)
		for _, f := range m.GetAllLocalPaths() {
			dir := filepath.Dir(f)
			if _, err := os.Stat(dir); err != nil {
				err = os.MkdirAll(dir, 0755)
				if err != nil {
					msg := fmt.Sprintf("ButlerConfig::CheckPaths(): err=%s", err.Error())
					log.Fatal(msg)
				}
				log.Infof("ButlerConfig::CheckPaths(): Created directory \"%s\"", dir)
				log.Debugf("ButlerConfig::CheckPaths(): setting m.ReloadManager=true")
				m.ReloadManager = true
			}
		}

		if m.CleanFiles {
			err := filepath.Walk(m.DestPath, m.PathCleanup)
			if err != nil {
				log.Debugf("ButlerConfig::CheckPaths(): got err for filepath. setting m.ReloadManager=true")
				m.ReloadManager = true
			}
		}
	}
	return nil
}

// Rewrite of main.ProcessPrometheusConfigFiles
func (bm *ButlerManager) ProcessPrimaryConfigFiles(c chan bool) error {
	var (
		TmpFiles     []string
		LegitFileMap map[string]ConfigFileMap
		IsModified   bool
		RenderFile   bool
	)

	IsModified = false
	RenderFile = true
	LegitFileMap = make(map[string]ConfigFileMap)
	_ = TmpFiles
	_ = IsModified
	_ = RenderFile
	_ = LegitFileMap

	TmpMergedFile, err := ioutil.TempFile("/tmp", "pcmsfile")
	if err != nil {
		msg := fmt.Sprintf("ButlerManager::ProcessPrimaryConfigFiles(): Could not create temporary file . err=v", err.Error())
		log.Fatal(msg)
	}

	//for _, opts := range bm.Opts.GetPrimaryConfigUrls
	for _, opts := range bm.ManagerOpts {
		log.Debugf("ButlerManager::ProcessPrimaryConfigFiles(): opts=%#v", opts)
		for i, u := range opts.GetPrimaryConfigUrls() {
			log.Debugf("ButlerManager::ProcessPrimaryConfigFiles(): i=%v, u=%v", i, u)
			FileMap := ConfigFileMap{}
			_ = FileMap
			f := opts.DownloadConfigFile(u)
			if f == nil {
				// unsure what to do with this right now
				//stats.SetButlerContactVal(stats.FAILURE, GetPrometheusLabels(Files)[i])
				continue
			} else {
				// unsure what to do with this right now
				//stats.SetButlerContactVal(stats.SUCCESS, GetPrometheusLabels(Files)[i])
			}
		}
	}

	// Clean up the temporary configuration file
	os.Remove(TmpMergedFile.Name())
	return nil
}

func (bm *ButlerManager) ProcessAdditionalConfigFiles(c chan bool) error {
	var ()
	return nil
}

// PathCleanup
func (bm *ButlerManager) PathCleanup(path string, f os.FileInfo, err error) error {
	log.Debugf("ButlerManager::PathCleanup(): entering")
	var (
		Found bool
	)
	Found = false

	// We don't have to do anything with a directory
	if f.Mode().IsDir() {
		log.Debugf("ButlerManager::PathCleanup(): %s is a directory... returning nil", f.Name())
		return nil
	}

	for _, file := range bm.GetAllLocalPaths() {
		if path == file {
			Found = true
		}
	}

	if !Found {
		message := fmt.Sprintf("Found unknown file \"%s\". deleting...", path)
		log.Debugf("ButlerManager::PathCleanup(): Found unknown file \"%s\". deleting...", path)
		os.Remove(path)
		return errors.New(message)
	}
	return nil
}

func (bm *ButlerManager) GetAllLocalPaths() []string {
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

func (bmo *ButlerManagerOpts) AppendPrimaryConfigUrl(c string) error {
	log.Debugf("ButlerManagerOpts::AppendPrimaryConfigUrl(): adding %s to PrimaryConfigsUrls...", c)
	bmo.PrimaryConfigsFullUrls = append(bmo.PrimaryConfigsFullUrls, c)
	return nil
}

func (bmo *ButlerManagerOpts) AppendPrimaryConfigFile(c string) error {
	log.Debugf("ButlerManagerOpts::AppendPrimaryConfigFile(): adding %s to PrimaryConfigsFullLocalPaths...", c)
	bmo.PrimaryConfigsFullLocalPaths = append(bmo.PrimaryConfigsFullLocalPaths, c)
	return nil
}

func (bmo *ButlerManagerOpts) AppendAdditionalConfigUrl(c string) error {
	log.Debugf("ButlerManagerOpts::AppendAdditionalConfigUrl(): adding %s to AdditionalConfigsUrls...", c)
	bmo.AdditionalConfigsFullUrls = append(bmo.AdditionalConfigsFullUrls, c)
	return nil
}

func (bmo *ButlerManagerOpts) AppendAdditionalConfigFile(c string) error {
	log.Debugf("ButlerManagerOpts::AppendAdditionalConfigFile(): adding %s to AdditionalConfigsFullLocalPaths...", c)
	bmo.AdditionalConfigsFullLocalPaths = append(bmo.AdditionalConfigsFullLocalPaths, c)
	return nil
}

func (bmo *ButlerManagerOpts) GetPrimaryConfigUrls() []string {
	return bmo.PrimaryConfigsFullUrls
}

func (bmo *ButlerManagerOpts) GetPrimaryConfigFiles() []string {
	return bmo.PrimaryConfigsFullLocalPaths
}

func (bmo *ButlerManagerOpts) GetAdditionalConfigUrls() []string {
	return bmo.AdditionalConfigsFullUrls
}

func (bmo *ButlerManagerOpts) GetAdditionalConfigFiles() []string {
	return bmo.AdditionalConfigsFullLocalPaths
}

// Really need to come up with a better method for this.
func (bmo *ButlerManagerOpts) DownloadConfigFile(file string) *os.File {
	switch bmo.Method {
	case "http", "https":
		log.Debugf("ButlerManagereOpts::DownloadConfigFile(): here i am")
		tmpFile, err := ioutil.TempFile("/tmp", "pcmsfile")
		if err != nil {
			msg := fmt.Sprintf("ButlerManagerOpts::DownloadConfigFile(): could not create temporary file. err=%v", err)
			log.Fatal(msg)
		}

		response, err := bmo.Opts.(ButlerManagerMethodHttpOpts).Client.Get(file)

		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Infof("ButlerManagerOpts::DownloadConfigFile(): Could not download from %s, err=%s", file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		defer response.Body.Close()
		defer tmpFile.Close()

		if response.StatusCode != 200 {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Infof("ButlerManagerOpts::DownloadConfigFile(): Did not receive 200 response code for %s. code=%v", file, response.StatusCode)
			tmpFile = nil
			return tmpFile
		}

		_, err = io.Copy(tmpFile, response.Body)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Infof("ButlerManagerOpts::DownloadConfigFile(): Could not copy to %s, err=%s", file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		return tmpFile
	default:
		return nil
	}
}
