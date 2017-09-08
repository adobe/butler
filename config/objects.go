package config

import (
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jasonlvhit/gocron"
	log "github.com/sirupsen/logrus"
)

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
	Name          string
	Urls          []string `mapstructure:"urls"`
	CleanFiles    bool     `mapstructure:"clean-files"`
	MustacheSubs  []string `mapstructure:"mustache-subs"`
	DestPath      string   `mapstructure:"dest-path"`
	ManagerOpts   map[string]*ButlerManagerOpts
	Reloader      ButlerManagerReloader
	ReloadManager bool
}

type ButlerManagerOpts struct {
	Method                          string   `mapstructure:"method"`
	UriPath                         string   `mapstructure:"uri-path"`
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
