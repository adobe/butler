package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	ButlerRawConfig         []byte
	ConfigSchedulerInterval = 300
	ValidSchemes            = []string{"http", "https"}
)

// butlerHeader and butlerFooter represent the strings that need to be matched
// against in the configuration files. If these entries do not exist in the
// downloaded file, then we cannot be assured that these files are legitimate
// configurations.
const (
	butlerHeader = "#butlerstart"
	butlerFooter = "#butlerend"
)

type ButlerConfigClient struct {
	Scheme     string
	HttpClient *retryablehttp.Client
}

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
	Urls         []string `mapstructure:"urls"`
	MustacheSubs []string `mapstructure:"mustache-subs"`
	ManagerOpts  map[string]ButlerManagerOpts
	Reloader     ButlerManagerReloader
}

type ButlerManagerOpts struct {
	Method           string   `mapstructure:"method"`
	UriPath          string   `mapstructure:"uri-path"`
	DestPath         string   `mapstructure:"dest-path"`
	PrimaryConfig    []string `mapstructure:"primary-config"`
	AdditionalConfig []string `mapstructure:"additional-config"`
	Opts             map[string]interface{}
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

func NewButlerConfigClient(scheme string) (*ButlerConfigClient, error) {
	var c ButlerConfigClient
	switch scheme {
	case "http", "https":
		c.Scheme = "http"
		c.HttpClient = retryablehttp.NewClient()
		c.HttpClient.Logger.SetFlags(0)
		c.HttpClient.Logger.SetOutput(ioutil.Discard)
	default:
		errMsg := fmt.Sprintf("Unsupported butler config scheme: %s", scheme)
		return &ButlerConfigClient{}, errors.New(errMsg)
	}
	return &c, nil
}

func (c *ButlerConfigClient) SetTimeout(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HttpClient.HTTPClient.Timeout = time.Duration(val) * time.Second
	}
}

func (c *ButlerConfigClient) SetRetryMax(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HttpClient.RetryMax = val
	}
}

func (c *ButlerConfigClient) SetRetryWaitMin(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HttpClient.RetryWaitMin = time.Duration(val) * time.Second
	}
}

func (c *ButlerConfigClient) SetRetryWaitMax(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HttpClient.RetryWaitMax = time.Duration(val) * time.Second
	}
}

func (c *ButlerConfigClient) Get(val string) (*http.Response, error) {
	var (
		response *http.Response
		err      error
	)
	switch c.Scheme {
	case "http", "https":
		response, err = c.HttpClient.Get(val)
	default:
		response = &http.Response{}
		err = errors.New("unsupported scheme")
	}
	return response, err
}

func ParseButlerConfigManager(config []byte) (ButlerManager, error) {
	return ButlerManager{}, nil
}

func GetButlerManagerOpts(entry string, bc *ButlerConfigSettings) error {
	var (
		err         error
		ManagerOpts ButlerManagerOpts
	)
	err = viper.UnmarshalKey(entry, &ManagerOpts)
	if err != nil {
		return err
	}

	switch ManagerOpts.Method {
	case "http", "https":
		break
	default:
		msg := fmt.Sprintf("unknown manager.method=%v", ManagerOpts.Method)
		return errors.New(msg)
	}

	if ManagerOpts.UriPath == "" {
		return errors.New("no manager.uri-path defined")
	}

	if ManagerOpts.DestPath == "" {
		return errors.New("no manager.dest-path defined")
	}

	if len(ManagerOpts.PrimaryConfig) < 1 {
		return errors.New("no manager.primary-config defined")
	}

	// AdditionalConfig is an optional paramater, and it shouldn't matter
	// if it exists or not ...
	//if ManagerOpts.AdditionalConfig == ""
	log.Debugf("GetButlerManagerOpts(): ManagerOpts=%v", ManagerOpts)
	return nil
}

func GetButlerConfigManager(entry string, bc *ButlerConfigSettings) error {
	var (
		err         error
		Manager     ButlerManager
		ManagerOpts ButlerManagerOpts
	)

	log.Debugf("GetButlerConfigManager(): entered with -> %s and %v", entry, *bc)
	err = viper.UnmarshalKey(entry, &Manager)
	if err != nil {
		return err
	}
	if len(Manager.Urls) < 1 {
		msg := fmt.Sprintf("No urls configured for manager %s", entry)
		return errors.New(msg)
	}
	log.Debugf("GetButlerConfigManager(): Manager.Urls=%v Manager.MustacheSubs=%v", Manager.Urls, Manager.MustacheSubs)

	for _, m := range Manager.Urls {
		mopts := fmt.Sprintf("%s.%s", entry, m)
		err = GetButlerManagerOpts(mopts, bc)
		//err = viper.UnmarshalKey(mopts, &ManagerOpts)
		if err != nil {
			return err
		}
		log.Debugf("GetButlerConfigManager(): ManagerOpts=%v", ManagerOpts)
	}

	return nil
}

func ParseButlerConfig(config []byte) error {
	var (
		//handlers []string
		ButlerConfig  ButlerConfigSettings
		ButlerGlobals ButlerConfigGlobals
	)
	// The Butler configuration is in TOML format
	viper.SetConfigType("toml")

	// We grab the config from a remote repo so it's in []byte format. let's see
	// if we can process it.
	err := viper.ReadConfig(bytes.NewBuffer(config))
	if err != nil {
		return err
	}

	ButlerConfig = ButlerConfigSettings{}

	// Let's start piecing together the globals
	err = viper.UnmarshalKey("globals", &ButlerGlobals)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	log.Debugf("ButlerConfig2=%#v", ButlerGlobals)
	ButlerConfig.Globals = ButlerGlobals

	// Let's grab some of the global settings
	if ButlerConfig.Globals.SchedulerInterval == 0 {
		ButlerConfig.Globals.SchedulerInterval = ConfigSchedulerInterval
	}

	log.Debugf("ParseButlerConfig(): globals.config-managers=%#v", ButlerConfig.Globals.Managers)
	log.Debugf("ParseButlerConfig(): len(globals.config-managers)=%v", len(ButlerConfig.Globals.Managers))

	// If there are no entries for config-managers, then the Unmarshal will create an empty array
	if len(ButlerConfig.Globals.Managers) < 1 {
		if ButlerConfig.Globals.ExitOnFailure {
			log.Fatalf("ParseButlerConfig(): globals.config-managers has no entries! exiting...")
		} else {
			log.Debugf("ParseButlerConfig(): globals.config-managers has no entries!")
			return errors.New("globals.config-managers has no entries. Nothing to do")
		}
	}

	ButlerConfig.Managers = make(map[string]ButlerManager)
	// Now let's start processing the managers. This is going
	for _, entry := range ButlerConfig.Globals.Managers {
		if !viper.IsSet(entry) {
			if ButlerConfig.Globals.ExitOnFailure {
				log.Fatalf("ParseButlerConfig(): %v is not in the configuration as a manager! exiting...", entry)
			} else {
				log.Debugf("ParseButlerConfig(): %v is not in the configuration as a manager", entry)
				msg := fmt.Sprintf("Cannot find manager for %s", entry)
				return errors.New(msg)
			}
		} else {
			log.Debugf("sub=%s", viper.Sub(entry))
			err = GetButlerConfigManager(entry, &ButlerConfig)
			if err != nil {
				if ButlerConfig.Globals.ExitOnFailure {
					log.Fatalf("ParseButlerConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
				} else {
					log.Debugf("ParseButlerConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
					msg := fmt.Sprintf("could not retrieve config options for %v. err=%v", entry, err.Error())
					return errors.New(msg)
				}
			}
			//ButlerConfig.Managers[entry] = ButlerManager{}
		}
	}

	log.Debugf("ButlerConfig.Managers=%#v", ButlerConfig.Managers)
	return nil
}

/*
func ButlerConfigHandler() error {
	log.Debugf("ButlerConfigHandler(): running")
	c, err := NewButlerConfigClient(ButlerConfigScheme)
	if err != nil {
		return err
	}

	c.SetTimeout(HttpTimeout)
	c.SetRetryMax(HttpRetries)
	c.SetRetryWaitMin(HttpRetryWaitMin)
	c.SetRetryWaitMax(HttpRetryWaitMax)

	response, err := c.Get(ButlerConfigUrl)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Did not receive 200 response code for %s. code=%d", ButlerConfigUrl, response.StatusCode))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not read response body for %s. err=%s", ButlerConfigUrl, err))
	}

	err = ValidateButlerConfig(body)
	if err != nil {
		return err
	}

	if ButlerRawConfig == nil {
		err = ParseButlerConfig(body)
		if err != nil {
			if ButlerConfig.Globals.ExitOnFailure {
				log.Fatal(err)
			} else {
				return err
			}
		} else {
			ButlerRawConfig = body
		}
	}

	if !bytes.Equal(ButlerRawConfig, body) {
		err = ParseButlerConfig(body)
		if err != nil {
			return err
		} else {
			ButlerRawConfig = body
		}
	}
	return nil
}
*/

func NewButlerConfigSettings() *ButlerConfigSettings {
	return &ButlerConfigSettings{}
}

func (c *ButlerConfigSettings) ParseButlerConfig(config []byte) ([]byte, error) {
	var (
		//handlers []string
		ButlerConfig  ButlerConfigSettings
		ButlerGlobals ButlerConfigGlobals
	)
	// The Butler configuration is in TOML format
	viper.SetConfigType("toml")

	// We grab the config from a remote repo so it's in []byte format. let's see
	// if we can process it.
	err := viper.ReadConfig(bytes.NewBuffer(config))
	if err != nil {
		return []byte("error"), err
	}

	ButlerConfig = ButlerConfigSettings{}

	// Let's start piecing together the globals
	err = viper.UnmarshalKey("globals", &ButlerGlobals)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	log.Debugf("ButlerConfig2=%#v", ButlerGlobals)
	ButlerConfig.Globals = ButlerGlobals

	// Let's grab some of the global settings
	if ButlerConfig.Globals.SchedulerInterval == 0 {
		ButlerConfig.Globals.SchedulerInterval = ConfigSchedulerInterval
	}

	log.Debugf("ParseButlerConfig(): globals.config-managers=%#v", ButlerConfig.Globals.Managers)
	log.Debugf("ParseButlerConfig(): len(globals.config-managers)=%v", len(ButlerConfig.Globals.Managers))

	// If there are no entries for config-managers, then the Unmarshal will create an empty array
	if len(ButlerConfig.Globals.Managers) < 1 {
		if ButlerConfig.Globals.ExitOnFailure {
			log.Fatalf("ParseButlerConfig(): globals.config-managers has no entries! exiting...")
		} else {
			log.Debugf("ParseButlerConfig(): globals.config-managers has no entries!")
			return []byte("error"), errors.New("globals.config-managers has no entries. Nothing to do")
		}
	}

	ButlerConfig.Managers = make(map[string]ButlerManager)
	// Now let's start processing the managers. This is going
	for _, entry := range ButlerConfig.Globals.Managers {
		if !viper.IsSet(entry) {
			if ButlerConfig.Globals.ExitOnFailure {
				log.Fatalf("ParseButlerConfig(): %v is not in the configuration as a manager! exiting...", entry)
			} else {
				log.Debugf("ParseButlerConfig(): %v is not in the configuration as a manager", entry)
				msg := fmt.Sprintf("Cannot find manager for %s", entry)
				return []byte("error"), errors.New(msg)
			}
		} else {
			log.Debugf("sub=%s", viper.Sub(entry))
			err = GetButlerConfigManager(entry, &ButlerConfig)
			if err != nil {
				if ButlerConfig.Globals.ExitOnFailure {
					log.Fatalf("ParseButlerConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
				} else {
					log.Debugf("ParseButlerConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
					msg := fmt.Sprintf("could not retrieve config options for %v. err=%v", entry, err.Error())
					return []byte("error"), errors.New(msg)
				}
			}
			//ButlerConfig.Managers[entry] = ButlerManager{}
		}
	}

	log.Debugf("ButlerConfig.Managers=%#v", ButlerConfig.Managers)
	return []byte("abc"), nil
}
