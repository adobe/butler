package config

import (
	"errors"
	"fmt"
	//"io"
	"net/http"
	//"os"
	//"sort"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

var (
	RequiredSubKeys = []string{"ethos-cluster-id"}
	ConfigCache     map[string][]byte
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

func (b *ButlerConfigSettings) GetAllConfigLocalPaths() []string {
	var result []string
	for _, m := range b.Managers {
		result = append(result, fmt.Sprintf("%s/%s", m.DestPath, m.PrimaryConfigName))
		for _, o := range m.ManagerOpts {
			for _, f := range o.AdditionalConfigsFullLocalPaths {
				result = append(result, f)
			}
		}
	}
	return result
}

type ButlerConfigGlobals struct {
	Managers          []string `mapstructure:"config-managers",json:"-"`
	SchedulerInterval int      `mapstructure:"scheduler-interval",json:"scheduler-interval"`
	ExitOnFailure     bool     `mapstructure:"exit-on-config-failure",json:"exit-on-failure"`
}

type ButlerManager struct {
	Name              string
	Urls              []string          `mapstructure:"urls"`
	CleanFiles        bool              `mapstructure:"clean-files"`
	MustacheSubsArray []string          `mapstructure:"mustache-subs",json:"-"`
	MustacheSubs      map[string]string `json:"mustache-subs"`
	EnableCache       bool              `mapstructure:"enable-cache"`
	CachePath         string            `mapstructure:"cache-path"`
	DestPath          string            `mapstructure:"dest-path"`
	PrimaryConfigName string            `mapstructure:"primary-config-name",json:"primary-config-name"`
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
	Opts                            ButlerManagerMethodOpts
}

type ButlerManagerMethodOpts interface {
	Get(string) (*http.Response, error)
}

type ButlerManagerMethodHttpOpts struct {
	Client       *retryablehttp.Client `json:"-"`
	Retries      int                   `mapstructure:"retries",json:"retries"`
	RetryWaitMax int                   `mapstructure:"retry-wait-max",json:"retry-wait-max"`
	RetryWaitMin int                   `mapstructure:"retry-wait-min",json:"retry-wait-min"`
	Timeout      int                   `mapstructure:"timeout",json:"timeout"`
}

func (b ButlerManagerMethodHttpOpts) Get(file string) (*http.Response, error) {
	res, err := b.Client.Get(file)
	return res, err
}

type ButlerManagerMethodGenericOpts struct {
}

func (b ButlerManagerMethodGenericOpts) Get(file string) (*http.Response, error) {
	var (
		err error
		res *http.Response
	)
	return res, err
}

type ButlerManagerReloader interface {
	Reload() error
	GetMethod() string
	GetOpts() ReloaderOpts
	SetOpts(ReloaderOpts) bool
}

type ReloaderOpts interface {
}

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
	// Set the reloader retry policy
	c.CheckRetry = r.ManagerReloadRetryPolicy
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
			// at this point error should be nil, so things are OK
		} else {
			msg := fmt.Sprintf("ButlerManagerReloaderHttp::Reload(): received bad response from server. reverting to last known good config. http_code=%d", int(resp.StatusCode))
			log.Infof(msg)
			stats.SetButlerKnownGoodCachedVal(stats.FAILURE)
			stats.SetButlerKnownGoodRestoredVal(stats.FAILURE)
			stats.SetButlerReloadVal(stats.FAILURE)
			// at this point we should raise an error
			return errors.New(msg)
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
