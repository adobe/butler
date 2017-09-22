package config

import (
	//"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

var (
	RequiredSubKeys = []string{"ethos-cluster-id"}
)

type ButlerChanEvent interface{}

type ConfigChanEvent struct {
	HasChanged bool
	TmpFile    *os.File
	ConfigFile *string
	Repo       map[string]*RepoFileEvent
}

func (c *ConfigChanEvent) CanCopyFiles() bool {
	var (
		res bool
	)
	res = true

	log.Debugf("ConfigChanEvent::CanCopyFiles(): seeing if we can copy files")
	for _, r := range c.Repo {
		//log.Debugf("ConfigChanEvent::CanCopyFiles(): r=%#v", r)
		for _, v := range r.Success {
			//log.Debugf("ConfigChanEvent::CanCopyFiles(): k=%#v, v=%#v", k, v)
			if v == false {
				res = false
			}
		}
	}
	log.Debugf("ConfigChanEvent::CanCopyFiles(): returning %v", res)
	return res
}

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

func (c *ConfigChanEvent) GetTmpFileMap() map[string]string {
	var (
		keys    []string
		tmpRes     map[string]string
		res map[string]string
	)
	tmpRes = make(map[string]string)
	res = make(map[string]string)

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
		res[v] = tmpRes[v]
	}
	log.Debugf("ConfigChanEvent::GetTmpFileMap(): res=%v", res)
	return res
}

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

func (c *ConfigChanEvent) CopyPrimaryConfigFiles() bool {
	log.Debugf("ButlerManager::CopyPrimaryConfigFiles(): entering")
	out, err := os.OpenFile(c.TmpFile.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
		stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(*c.ConfigFile))
		c.CleanTmpFiles()
		return false
	} else {
		for _, f := range c.GetTmpFileMap() {
			in, err := os.Open(f)
			if err != nil {
				log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f))
				c.CleanTmpFiles()
				out.Close()
				return false
			}
			_, err = io.Copy(out, in)
			if err != nil {
				log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f))
				c.CleanTmpFiles()
				out.Close()
				return false
			}
			in.Close()
		}
	}
	out.Close()
	return CompareAndCopy(c.TmpFile.Name(), *c.ConfigFile)
}

func (c *ConfigChanEvent) CopyAdditionalConfigFiles(destDir string) bool {
	var (
		IsModified bool
	)
	log.Debugf("ButlerManager::CopyAdditionalConfigFiles(): entering")
	log.Debugf("ButlerManager::CopyAdditionalConfigFiles(): destDir=%v", destDir)
	IsModified = false

	for i, f := range c.GetTmpFileMap() {
		destFile := fmt.Sprintf("%s/%s", destDir, i)
		if CompareAndCopy(f, destFile) {
			IsModified = true
		}
	}
	return IsModified
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

type ButlerConfig struct {
	Url                     string
	Client                  *ButlerConfigClient
	Config                  *ButlerConfigSettings
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

type ButlerConfigSettings struct {
	Managers map[string]*ButlerManager `json:"managers"`
	Globals  ButlerConfigGlobals       `json:"globals"`
}

type ButlerConfigGlobals struct {
	Managers          []string `mapstructure:"config-managers",json:"-"`
	SchedulerInterval int      `mapstructure:"scheduler-interval",json:"scheduler_interval"`
	ExitOnFailure     bool     `mapstructure:"exit-on-config-failure",json:"exit_on_failure"`
}

type ButlerManager struct {
	Name              string
	Urls              []string          `mapstructure:"urls"`
	CleanFiles        bool              `mapstructure:"clean-files"`
	MustacheSubsArray []string          `mapstructure:"mustache-subs",json:"-"`
	MustacheSubs      map[string]string `json:"mustache_subs"`
	DestPath          string            `mapstructure:"dest-path"`
	PrimaryConfigName string            `mapstructure:"primary-config-name",json:"primary_config-name"`
	ManagerOpts       map[string]*ButlerManagerOpts
	Reloader          ButlerManagerReloader
	ReloadManager     bool
}

func (bm *ButlerManager) Reload() error {
	log.Debugf("ButlerManager::Reload(): reloading...")
	var Reloader interface{}
	log.Debugf("ButlerManager::Reload(): bm.Reloader=%#v", bm.Reloader)
	switch bm.Reloader.Method {
	case "http", "https":
		Reloader = bm.Reloader.Opts.(*ButlerManagerReloaderHttpOpts)
		log.Debugf("ButlerManager::Reload(): Reloader=%#v", Reloader)
	default:
		return nil
	}
	//return Reloader.Reload()
	return nil
}

type ButlerManagerOpts struct {
	Method                          string `mapstructure:"method"`
	UriPath                         string `mapstructure:"uri-path"`
	Repo                            string
	PrimaryConfig                   []string `mapstructure:"primary-config"`
	AdditionalConfig                []string `mapstructure:"additional-config"`
	PrimaryConfigsFullUrls          []string
	AdditionalConfigsFullUrls       []string
	PrimaryConfigsFullLocalPaths    []string
	AdditionalConfigsFullLocalPaths []string
	Opts                            interface{}
}

type ButlerManagerMethodHttpOpts struct {
	Client       *retryablehttp.Client `json:"-"`
	Retries      int                   `mapstructure:"retries"`
	RetryWaitMax int                   `mapstructure:"retry-wait-max"`
	RetryWaitMin int                   `mapstructure:"retry-wait-min"`
	Timeout      int                   `mapstructure:"timeout"`
}

func (o *ButlerManagerMethodHttpOpts) Reload() error {
	log.Debugf("ButlerManagerMethodHttpOpts::Reload(): running http reloader")
	return nil
}

type ButlerManagerReloader struct {
	Method string `mapstructure:"method"`
	Opts   interface{}
}

type ButlerManagerReloaderHttpOpts struct {
	Client       *retryablehttp.Client `json:"-"`
	Host         string                `mapstructure:"host"`
	Port         int                   `mapstructure:"port"`
	Uri          string                `mapstructure:"uri"`
	Method       string                `mapstructure:"method"`
	Payload      string                `mapstructure:"payload"`
	Retries      int                   `mapstructure:"retries"`
	RetryWaitMax int                   `mapstructure:"retry-wait-max`
	RetryWaitMin int                   `mapstructure:"retry-wait-min`
	Timeout      int                   `mapstructure:"timeout`
}
