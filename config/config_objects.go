package config

type ButlerConfigSettings struct {
	Managers map[string]ButlerManager
	Globals  ButlerConfigGlobals
}

type ButlerConfigGlobals struct {
	Managers          []string `mapstructure:"config-managers"`
	SchedulerInterval int      `mapstructure:"scheduler-interval"`
	ExitOnFailure     bool     `mapstructure:"exit-on-config-failure"`
	CleanFiles        bool     `mapstructure:"clean-files"`
}

type ButlerManager struct {
	Name         string
	Urls         []string `mapstructure:"urls"`
	MustacheSubs []string `mapstructure:"mustache-subs"`
	DestPath     string   `mapstructure:"dest-path"`
	ManagerOpts  map[string]ButlerManagerOpts
	Reloader     ButlerManagerReloader
}

type ButlerManagerOpts struct {
	Method                     string   `mapstructure:"method"`
	UriPath                    string   `mapstructure:"uri-path"`
	PrimaryConfig              []string `mapstructure:"primary-config"`
	AdditionalConfig           []string `mapstructure:"additional-config"`
	PrimaryConfigsFullPaths    []string
	AdditionalConfigsFullPaths []string
	Opts                       interface{}
}

type ButlerManagerMethodHttpOpts struct {
	Retries      int `mapstructure:"retries"`
	RetryWaitMin int `mapstructure:"retry-wait-min"`
	RetryWaitMax int `mapstructure:"retry-wait-max"`
	Timeout      int `mapstructure:"timeout"`
}

type ButlerManagerReloader struct {
	Method string `mapstructure:"method"`
	Opts   interface{}
}

type ButlerManagerReloaderHttpOpts struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Uri          string `mapstructure:"uri"`
	Method       string `mapstructure:"method"`
	Payload      string `mapstructure:"payload"`
	Retries      int    `mapstructure:"retries"`
	RetryWaitMin int    `mapstructure:"retry-wait-min`
	RetryWaitMax int    `mapstructure:"retry-wait-max`
	Timeout      int    `mapstructure:"timeout`
}
