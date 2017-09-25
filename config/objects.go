package config

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

var (
	RequiredSubKeys = []string{"ethos-cluster-id"}
)

type ButlerChanEvent interface{}

type TmpFile struct {
	Name string
	File string
}

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
			in, err := os.Open(f.File)
			if err != nil {
				log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f.Name))
				c.CleanTmpFiles()
				out.Close()
				return false
			}
			_, err = io.Copy(out, in)
			if err != nil {
				log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f.Name))
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
	IsModified = false

	for _, f := range c.GetTmpFileMap() {
		destFile := fmt.Sprintf("%s/%s", destDir, f.Name)
		if CompareAndCopy(f.File, destFile) {
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
	Reloader          ButlerManagerReloader `mapstructure:"-"`
	ReloadManager     bool
}

func (bm *ButlerManager) Reload() error {
	log.Debugf("ButlerManager::Reload(): reloading...")
	var reloader ButlerManagerReloader
	switch bm.Reloader.GetMethod() {
	case "http", "https":
		reloader = bm.Reloader
	default:
		return nil
	}
	return reloader.Reload()
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
	Retries      int                   `mapstructure:"retries",json:"retries"`
	RetryWaitMax int                   `mapstructure:"retry-wait-max",json:"retry-wait-max"`
	RetryWaitMin int                   `mapstructure:"retry-wait-min",json:"retry-wait-min"`
	Timeout      int                   `mapstructure:"timeout",json:"timeout"`
}

type ButlerManagerReloader interface {
	Reload() error
	GetMethod() string
	GetOpts() ReloaderOpts
	SetOpts(ReloaderOpts) bool
	//GetOpts() interface{}
	//SetOpts(interface{}) bool
}

type ReloaderOpts interface {
}

/*
type ButlerManagerReloader struct {
	Method string `mapstructure:"method"`
	Opts   interface{}
}
*/

type ButlerManagerReloaderHttp struct {
	Method string `mapstructer:"method"`
	Opts   ButlerManagerReloaderHttpOpts
}

type ButlerManagerReloaderHttpOpts struct {
	Client       *retryablehttp.Client `json:"-"`
	ContentType  string                `json:"content-type"`
	Host         string                `json:"host"`
	Port         int                   `json:"port"`
	Uri          string                `json:"uri"`
	Method       string                `json:"method"`
	Payload      string                `json:"payload"`
	Retries      int                   `json:"retries"`
	RetryWaitMax int                   `json:"retry-wait-max"`
	RetryWaitMin int                   `json:"retry-wait-min"`
	Timeout      int                   `json:"timeout"`
}

func (b *ButlerManagerReloaderHttpOpts) GetClient() *retryablehttp.Client {
	return b.Client
}

func (r ButlerManagerReloaderHttp) Reload() error {
	var (
		err error
	)
	log.Debugf("ButlerManagerReloaderHttp::Reload() reloading manager using http")
	o := r.GetOpts().(ButlerManagerReloaderHttpOpts)
	c := o.GetClient()
	reloadUrl := fmt.Sprintf("%s://%s:%d%s", r.Method, o.Host, o.Port, o.Uri)

	switch o.Method {
	case "post":
		log.Debugf("ButlerManagerReloaderHttp::Reload(): posting up!")
		resp, err := c.Post(reloadUrl, o.ContentType, strings.NewReader(o.Payload))
		if err != nil {
			msg := fmt.Sprintf("ButlerManagerReloaderHttp::Reload(): err=%v", err.Error())
			log.Infof(msg)
			stats.SetButlerReloadVal(stats.FAILURE)
			return errors.New(msg)
		}
		if resp.StatusCode == 200 {
			log.Infof("ButlerManagerReloaderHttp::Reload(): successfully reloaded config. http_code=%d", int(resp.StatusCode))
			stats.SetButlerKnownGoodCachedVal(stats.SUCCESS)
			stats.SetButlerKnownGoodRestoredVal(stats.FAILURE)
			stats.SetButlerReloadVal(stats.SUCCESS)
			// stegen - must do this!
			//CacheConfigs()
		} else {
			log.Infof("ButlerManagerReloaderHttp::Reload(): received bad response from server. reverting to last known good config. http_code=%d", int(resp.StatusCode))
			stats.SetButlerKnownGoodCachedVal(stats.FAILURE)
			stats.SetButlerKnownGoodRestoredVal(stats.FAILURE)
			stats.SetButlerReloadVal(stats.FAILURE)
			// stegen - must do this!
			//RestoreCachedConfigs()
		}
	default:
		msg := fmt.Sprintf("ButlerManagerReloaderHttp::Reload(): %s is not a supported reload method", r.Method)
		return errors.New(msg)
	}

	return err
}

func (r ButlerManagerReloaderHttp) ManagerReloadRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		return true, err
	}

	// Here is our policy override. By default it looks for
	// res.StatusCode >= 500 ...
	if resp.StatusCode == 0 || resp.StatusCode >= 600 {
		return true, nil
	}
	return false, nil
}

func (r ButlerManagerReloaderHttp) GetMethod() string {
	return r.Method
}
func (r ButlerManagerReloaderHttp) GetOpts() ReloaderOpts {
	return r.Opts
}

func (r ButlerManagerReloaderHttp) SetOpts(opts ReloaderOpts) bool {
	r.Opts = opts.(ButlerManagerReloaderHttpOpts)
	return true
}

type ButlerGenericReloader struct {
	Opts ButlerGenericReloaderOpts
}
type ButlerGenericReloaderOpts struct{}

func (r ButlerGenericReloader) Reload() error {
	var (
		res error
	)
	return res
}
func (r ButlerGenericReloader) GetMethod() string {
	return "none"
}
func (r ButlerGenericReloader) GetOpts() ReloaderOpts {
	return r.Opts
}

func (r ButlerGenericReloader) SetOpts(opts ReloaderOpts) bool {
	r.Opts = opts.(ButlerGenericReloaderOpts)
	return true
}
