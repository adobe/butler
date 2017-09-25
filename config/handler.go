package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	//"path"
	"path/filepath"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

func NewButlerConfig() *ButlerConfig {
	return &ButlerConfig{FirstRun: true}
}

func NewConfigChanEvent() *ConfigChanEvent {
	var (
		c ConfigChanEvent
		f RepoFileEvent
	)
	c = ConfigChanEvent{}
	_ = f
	c.Repo = make(map[string]*RepoFileEvent)
	//f = RepoFileEvent{}
	//f.Success = make(map[string]bool)
	//f.Error = make(map[string]error)
	return &c
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
	log.Debugf("ButlerConfig::SetUrl(): setting bc.Url=%s", u)
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
			log.Debugf("ButlerConfig::Init(): could not initialize butler config. err=%s", err.Error())
			return err
		}
		bc.Url = ConfigUrl
	}

	c, err := NewButlerConfigClient(bc.Scheme)
	if err != nil {
		log.Debugf("ButlerConfig::Init(): could not initialize butler config. err=%s", err.Error())
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
	var (
		ReloadManager []string
	)
	log.Debugf("ButlerConfig::RunCMHandler(): entering")

	c1 := make(chan ButlerChanEvent)
	c2 := make(chan ButlerChanEvent)

	bc.CheckPaths()

	for _, m := range bc.Config.Managers {
		go m.DownloadPrimaryConfigFiles(c1)
		go m.DownloadAdditionalConfigFiles(c2)
		primaryChan, additionalChan := <-c1, <-c2

		if primaryChan.(*ConfigChanEvent).CanCopyFiles() && additionalChan.(*ConfigChanEvent).CanCopyFiles() {
			log.Debugf("ButlerConfig::RunCMHandler(): successfully retrieved files. processing...")
			p := primaryChan.(*ConfigChanEvent).CopyPrimaryConfigFiles()
			a := additionalChan.(*ConfigChanEvent).CopyAdditionalConfigFiles(m.DestPath)
			if p || a {
				ReloadManager = append(ReloadManager, m.Name)
			}
			primaryChan.(*ConfigChanEvent).CleanTmpFiles()
			additionalChan.(*ConfigChanEvent).CleanTmpFiles()
		} else {
			log.Debugf("ButlerConfig::RunCMHandler(): cannot copy files. cleaning up...")
			primaryChan.(*ConfigChanEvent).CleanTmpFiles()
			additionalChan.(*ConfigChanEvent).CleanTmpFiles()
		}
	}

	if len(ReloadManager) == 0 {
		log.Debugf("ButlerConfig::RunCMHandler(): CM files unchanged... continuing.")
	} else {
		for _, m := range ReloadManager {
			log.Debugf("ButlerConfig::RunCMHandler(): CM file changes. reloading manager %v...", m)
			bc.Config.Managers[m].Reload()
		}
	}

	return nil
}

func (bc *ButlerConfig) CheckPaths() error {
	log.Debugf("ButlerConfig::CheckPaths(): entering")
	for _, m := range bc.Config.Managers {
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
func (bm *ButlerManager) DownloadPrimaryConfigFiles(c chan ButlerChanEvent) error {
	var (
		Chan              *ConfigChanEvent
		PrimaryConfigName string
	)

	Chan = NewConfigChanEvent()
	PrimaryConfigName = fmt.Sprintf("%s/%s", bm.DestPath, bm.PrimaryConfigName)
	Chan.ConfigFile = &PrimaryConfigName

	// Create a temporary file for the merged prometheus configurations.
	tmpFile, err := ioutil.TempFile("/tmp", "pcmsfile")
	if err != nil {
		msg := fmt.Sprintf("ButlerManager::DownloadPrimaryConfigFiles(): Could not create temporary file . err=v", err.Error())
		log.Fatal(msg)
	}
	Chan.TmpFile = tmpFile

	// Process the prometheus.yml configuration files
	// We are going to iterate through each of the potential managers configured
	for _, opts := range bm.ManagerOpts {
		for i, u := range opts.GetPrimaryConfigUrls() {
			log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): i=%v, u=%v", i, u)
			log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): f=%s", opts.GetPrimaryRemoteConfigFiles()[i])
			f := opts.DownloadConfigFile(u)
			if f == nil {
				stats.SetButlerContactVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): download for %s is nil.", u)
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not download file"))
				continue
			} else {
				stats.SetButlerContactVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}
			Chan.SetTmpFile(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], f.Name())

			// Let's ensure that the files starts with #butlerstart and
			// ends with #butlerend. If they do not, then we will assume
			// we did not get a correct configuration, or that there is an
			// issue with the upstream
			if err := ValidateButlerConfig(f); err != nil {
				log.Infof("%s for %s.", err.Error(), u)
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not validate file"))
				continue
			} else {
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}

			// For the prometheus.yml we have to do some mustache replacement on downloaded file
			if err := RenderConfigMustache(f, bm.MustacheSubs); err != nil {
				log.Infof("%s for %s.\n", err.Error(), u)
				stats.SetButlerRenderVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): render for %s is nil.", opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not render file"))
				continue
			} else {
				// stats
				stats.SetButlerRenderVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}
			// Going to want to keep tabs on TmpFiles, and remove all of them at the end.
			// Remember that we want to merge all of the downloaded files, so why remove them
			// right now.
		}
	}
	// Need to verify whether or not we got all of the prometheus configuration
	// files. If not, then we should not try to process anything
	/*
		for _, v := range LegitFileMap {
			if !v.Success {
				RenderFile = false
			}
		}
	*/

	// Let's process and merge the prometheus files
	/*
		* we aren't going to do this here anymore
		log.Infof("ButlerManager::DownloadPrimaryConfigFiles(): Chan.CanCopyFiles()=%v", Chan.CanCopyFiles())
		if Chan.CanCopyFiles() {
			out, err := os.OpenFile(Chan.TmpFile.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
			_ = out
			if err != nil {
				log.Infof("Could not process and merge new %s err=%s.", PrimaryConfigName, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(PrimaryConfigName))
				// Just giving up at this point. Cleaning the temporary files.
				Chan.CleanTmpFiles()

				// Clean up the Primary temp file
				// This is handled elsewhere now
				//os.Remove(TmpMergedFile.Name())
				Chan.SetFailure("local", stats.GetStatsLabel(PrimaryConfigName), errors.New("could not process file"))
				c <- Chan
				return err
			} else {
				//for _, f := range ProcessedFiles {
				for _, f := range Chan.GetTmpFileMap() {
					in, err := os.Open(f)
					if err != nil {
						log.Infof("Could not process and merge new %s err=%s.", PrimaryConfigName, err.Error())
						stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f))
						// just giving up at this point, as well...
						// Clean up the temporary files
						Chan.CleanTmpFiles()

						// Clean up the primary config temp file
						//os.Remove(TmpMergedFile.Name())
						Chan.SetFailure("local", stats.GetStatsLabel(PrimaryConfigName), errors.New("could not process file"))
						c <- Chan
						return err
					}
					_, err = io.Copy(out, in)
					if err != nil {
						log.Infof("Could not process and merge new %s err=%s.", PrimaryConfigName, err.Error())
						stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f))
						// just giving up at this point, again...
						// Clean up the temporary files
						Chan.CleanTmpFiles()

						// Clean up the primary config temp file
						//os.Remove(TmpMergedFile.Name())
						Chan.SetFailure("local", stats.GetStatsLabel(PrimaryConfigName), errors.New("could not process file"))
						c <- Chan
						return err
					}
					in.Close()
				}
				log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): rendering final primary config file %s", PrimaryConfigName)
			}
			out.Close()
			IsModified = CompareAndCopy(Chan.TmpFile.Name(), PrimaryConfigName)
			if !IsModified {
				log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): primary config file %s is unchanged", PrimaryConfigName)
			}
		} else {
			log.Debugf("ButlerManager::DownloadPrimaryConfigFiles(): not rendering final primary config file %s", PrimaryConfigName)
			IsModified = false
		}
	*/

	// Clean up Primary temp file
	//os.Remove(TmpMergedFile.Name())

	// Update the channel
	c <- Chan

	return nil
}

func (bm *ButlerManager) DownloadAdditionalConfigFiles(c chan ButlerChanEvent) error {
	var (
		Chan       *ConfigChanEvent
		IsModified bool
	)

	Chan = NewConfigChanEvent()
	IsModified = false
	_ = IsModified

	// Process the additional configuration files
	for _, opts := range bm.ManagerOpts {
		for i, u := range opts.GetAdditionalConfigUrls() {
			log.Debugf("ButlerManager::DownloadAdditionalConfigFiles(): i=%v, u=%v", i, u)
			f := opts.DownloadConfigFile(u)
			if f == nil {
				log.Debugf("ButlerManager::DownloadAdditionalConfigFiles(): download for %s is nil.", u)
				stats.SetButlerContactVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not download file"))
				continue
			} else {
				stats.SetButlerContactVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
				Chan.SetTmpFile(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], f.Name())
			}

			// Let's ensure that the files starts with #butlerstart and
			// ends with #butlerend. IF they do not, then we will assume
			// we did not get a correct configuration, or that there is an
			// issue with the upstream
			if err := ValidateButlerConfig(f); err != nil {
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not validate file"))
				continue
			} else {
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
			}

			// Let's process some mustache ...
			// NOTE: We USED to do this only for the primary configuration. Unsure how this will
			// affect the additional configurations. we can remove this if there are adverse
			// effects.
			if err := RenderConfigMustache(f, bm.MustacheSubs); err != nil {
				stats.SetButlerRenderVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not render file"))
				continue
			} else {
				stats.SetButlerRenderVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
			}
		}
	}

	/*
		if Chan.CanCopyFiles() {
		}

		Chan.GetTmpFileMap()

		// Clean up the temporary files
		Chan.CleanTmpFiles()
	*/

	// Update the channel
	c <- Chan
	return nil
}

// PathCleanup
func (bm *ButlerManager) PathCleanup(path string, f os.FileInfo, err error) error {
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

func (bmo *ButlerManagerOpts) GetPrimaryLocalConfigFiles() []string {
	return bmo.PrimaryConfigsFullLocalPaths
}

func (bmo *ButlerManagerOpts) GetPrimaryRemoteConfigFiles() []string {
	return bmo.PrimaryConfig
}

func (bmo *ButlerManagerOpts) GetAdditionalConfigUrls() []string {
	return bmo.AdditionalConfigsFullUrls
}

func (bmo *ButlerManagerOpts) GetAdditionalLocalConfigFiles() []string {
	return bmo.AdditionalConfigsFullLocalPaths
}

func (bmo *ButlerManagerOpts) GetAdditionalRemoteConfigFiles() []string {
	return bmo.AdditionalConfig
}

// Really need to come up with a better method for this.
func (bmo *ButlerManagerOpts) DownloadConfigFile(file string) *os.File {
	switch bmo.Method {
	case "http", "https":
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
