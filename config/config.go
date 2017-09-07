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

func GetButlerManagerMethodOpts(entry string, method string, bc *ButlerConfigSettings) (interface{}, error) {
	var (
		result interface{}
		err    error
	)

	switch method {
	case "http", "https":
		var httpOpts ButlerManagerMethodHttpOpts
		err = viper.UnmarshalKey(entry, &httpOpts)
		if err != nil {
			return result, err
		}
		return httpOpts, nil
	default:
		msg := fmt.Sprintf("unknown manager.method=%s opts for %s", method, entry)
		return &result, errors.New(msg)
	}
	// Shouldn't get here.
	return result, nil
}

func GetButlerManagerOpts(entry string, bc *ButlerConfigSettings) (*ButlerManagerOpts, error) {
	var (
		err         error
		ManagerOpts ButlerManagerOpts
	)
	err = viper.UnmarshalKey(entry, &ManagerOpts)
	if err != nil {
		return &ButlerManagerOpts{}, err
	}

	switch ManagerOpts.Method {
	case "http", "https":
		break
	default:
		msg := fmt.Sprintf("unknown manager.method=%v", ManagerOpts.Method)
		return &ButlerManagerOpts{}, errors.New(msg)
	}

	if ManagerOpts.UriPath == "" {
		return &ButlerManagerOpts{}, errors.New("no manager.uri-path defined")
	}

	/*
		if ManagerOpts.DestPath == "" {
			return &ButlerManagerOpts{}, errors.New("no manager.dest-path defined")
		}
	*/

	if len(ManagerOpts.PrimaryConfig) < 1 {
		return &ButlerManagerOpts{}, errors.New("no manager.primary-config defined")
	}

	methodOpts := fmt.Sprintf("%s.%s", entry, ManagerOpts.Method)
	mopts, err := GetButlerManagerMethodOpts(methodOpts, ManagerOpts.Method, bc)
	ManagerOpts.Opts = mopts

	return &ManagerOpts, nil
}

func GetButlerConfigReloaderOpts(entry string, method string, bc *ButlerConfigSettings) (interface{}, error) {
	var (
		result interface{}
		err    error
	)

	switch method {
	case "http", "https":
		var httpOpts ButlerManagerReloaderHttpOpts
		err = viper.UnmarshalKey(entry, &httpOpts)
		if err != nil {
			return result, err
		}
		return httpOpts, nil
	default:
		msg := fmt.Sprintf("unknown config reloader method=%s opts for %s", method, entry)
		return result, errors.New(msg)
	}
	return result, err
}

func GetButlerConfigReloader(entry string, bc *ButlerConfigSettings) (ButlerManagerReloader, error) {
	var (
		result ButlerManagerReloader
		err    error
	)
	key := fmt.Sprintf("%s.reloader", entry)
	err = viper.UnmarshalKey(key, &result)
	if err != nil {
		return ButlerManagerReloader{}, err
	}

	switch result.Method {
	case "http", "https":
		ent := fmt.Sprintf("%s.%s", key, result.Method)
		opts, err := GetButlerConfigReloaderOpts(ent, result.Method, bc)
		if err != nil {
			return ButlerManagerReloader{}, err
		}
		result.Opts = opts
		break
	default:
		msg := fmt.Sprintf("unknown reloader method=%s for %s", result.Method, entry)
		return ButlerManagerReloader{}, errors.New(msg)
	}
	return result, err
}

func GetButlerConfigManager(entry string, bc *ButlerConfigSettings) error {
	var (
		err     error
		Manager ButlerManager
	)

	Manager.Name = entry

	err = viper.UnmarshalKey(entry, &Manager)
	if err != nil {
		return err
	}
	if len(Manager.Urls) < 1 {
		msg := fmt.Sprintf("No urls configured for manager %s", entry)
		return errors.New(msg)
	}

	if Manager.DestPath == "" {
		msg := fmt.Sprintf("No dest-path configured for manager %s", entry)
		errors.New(msg)
	}

	Manager.ManagerOpts = make(map[string]ButlerManagerOpts)
	for _, m := range Manager.Urls {
		bc.Managers[entry] = Manager
		mopts := fmt.Sprintf("%s.%s", entry, m)
		opts, err := GetButlerManagerOpts(mopts, bc)
		if err != nil {
			return err
		}
		bc.Managers[entry].ManagerOpts[mopts] = *opts
	}

	reloader, err := GetButlerConfigReloader(entry, bc)
	if err != nil {
		return err
	}
	m := bc.Managers[entry]
	m.Reloader = reloader
	bc.Managers[entry] = m
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

func NewButlerConfigSettings() *ButlerConfigSettings {
	return &ButlerConfigSettings{}
}

func (c *ButlerConfigSettings) ParseButlerConfig(config []byte) error {
	var (
		//handlers []string
		ButlerConfig  ButlerConfigSettings
		ButlerGlobals ButlerConfigGlobals
	)
	log.Debugf("ButlerConfigSettings::ParseButlerConfig(): entering.")
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
	log.Debugf("ButlerConfigSettings::ParseButlerConfig(): ButlerGlobals=%#v", ButlerGlobals)
	ButlerConfig.Globals = ButlerGlobals

	// Let's grab some of the global settings
	if ButlerConfig.Globals.SchedulerInterval == 0 {
		ButlerConfig.Globals.SchedulerInterval = ConfigSchedulerInterval
	}

	log.Debugf("ButlerConfigSettings::ParseButlerConfig(): globals.config-managers=%#v", ButlerConfig.Globals.Managers)
	log.Debugf("ButlerConfigSettings::ParseButlerConfig(): len(globals.config-managers)=%v", len(ButlerConfig.Globals.Managers))

	// If there are no entries for config-managers, then the Unmarshal will create an empty array
	if len(ButlerConfig.Globals.Managers) < 1 {
		if ButlerConfig.Globals.ExitOnFailure {
			log.Fatalf("ButlerConfigSettings::ParseButlerConfig(): globals.config-managers has no entries! exiting...")
		} else {
			log.Debugf("ButlerConfigSettings::ParseButlerConfig(): globals.config-managers has no entries!")
			return errors.New("globals.config-managers has no entries. Nothing to do")
		}
	}

	ButlerConfig.Managers = make(map[string]ButlerManager)
	// Now let's start processing the managers. This is going
	for _, entry := range ButlerConfig.Globals.Managers {
		log.Debugf("ButlerConfigSettings::ParseButlerConfig(): checking config entry=%s", entry)
		if !viper.IsSet(entry) {
			if ButlerConfig.Globals.ExitOnFailure {
				log.Fatalf("ButlerConfigSettings::ParseButlerConfig(): %v is not in the configuration as a manager! exiting...", entry)
			} else {
				log.Debugf("ButlerConfigSettings::ParseButlerConfig(): %v is not in the configuration as a manager", entry)
				msg := fmt.Sprintf("Cannot find manager for %s", entry)
				return errors.New(msg)
			}
		} else {
			err = GetButlerConfigManager(entry, &ButlerConfig)
			if err != nil {
				if ButlerConfig.Globals.ExitOnFailure {
					log.Fatalf("ButlerConfigSettings::ParseButlerConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
				} else {
					log.Debugf("ButlerConfigSettings::ParseButlerConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
					msg := fmt.Sprintf("could not retrieve config options for %v. err=%v", entry, err.Error())
					return errors.New(msg)
				}
			}
		}
	}

	// Set the values in the config structure
	c.Managers = ButlerConfig.Managers
	c.Globals = ButlerConfig.Globals
	c.Init()

	return nil
}

func (c *ButlerConfigSettings) Init() error {
	log.Debugf("ButlerConfigSettings::Init(): entering")
	for _, m := range c.Managers {
		log.Debugf("ButlerConfigSettings::Init(): manager=%#v", m)
		for _, u := range m.Urls {
			log.Debugf("ButlerConfigSettings::Init(): url=%#v", u)
			opts := fmt.Sprintf("%s.%s", m.Name, u)
			log.Debugf("ButlerConfigSettings::Init(): opts=%s", opts)
			log.Debugf("ButlerConfigSettings::Init(): ManagerOpts=%#v", m.ManagerOpts[opts])
		}
	}
	return nil
}
