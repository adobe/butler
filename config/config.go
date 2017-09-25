package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	//"reflect"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	//"github.com/mitchellh/mapstructure"
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
		httpOpts.Client = retryablehttp.NewClient()
		httpOpts.Client.Logger.SetFlags(0)
		httpOpts.Client.Logger.SetOutput(ioutil.Discard)
		httpOpts.Client.Logger.SetOutput(ioutil.Discard)
		httpOpts.Client.Logger.SetOutput(ioutil.Discard)
		httpOpts.Client.HTTPClient.Timeout = time.Duration(httpOpts.Timeout) * time.Second
		httpOpts.Client.RetryMax = httpOpts.Retries
		httpOpts.Client.RetryWaitMax = time.Duration(httpOpts.RetryWaitMax) * time.Second
		httpOpts.Client.RetryWaitMin = time.Duration(httpOpts.RetryWaitMin) * time.Second
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

	repoSplit := strings.Split(entry, ".")
	ManagerOpts.Repo = strings.Join(repoSplit[1:len(repoSplit)], ".")

	if len(ManagerOpts.PrimaryConfig) < 1 {
		return &ButlerManagerOpts{}, errors.New("no manager.primary-config defined")
	}

	methodOpts := fmt.Sprintf("%s.%s", entry, ManagerOpts.Method)
	mopts, err := GetButlerManagerMethodOpts(methodOpts, ManagerOpts.Method, bc)
	ManagerOpts.Opts = mopts

	return &ManagerOpts, nil
}

func GetButlerConfigReloader(entry string, bc *ButlerConfigSettings) (ButlerManagerReloader, error) {
	var (
		res    ButlerManagerReloader
		method string
		result map[string]interface{}
		err    error
	)
	key := fmt.Sprintf("%s.reloader", entry)

	err = viper.UnmarshalKey(key, &result)
	if err != nil {
		return ButlerGenericReloader{}, err
	}

	method = result["method"].(string)
	jsonRes, err := json.Marshal(result[method])
	if err != nil {
		return ButlerGenericReloader{}, err
	}
	log.Debugf("GetButlerConfigReloader(): jsonRes=%s", jsonRes)

	switch method {
	case "http", "https":
		var httpOpts ButlerManagerReloaderHttpOpts
		err = json.Unmarshal(jsonRes, &httpOpts)
		if err != nil {
			return ButlerGenericReloader{}, err
		}
		log.Debugf("GetButlerConfigReloader(): httpOpts=%#v", httpOpts)
		httpOpts.Client = retryablehttp.NewClient()
		httpOpts.Client.Logger.SetFlags(0)
		httpOpts.Client.Logger.SetOutput(ioutil.Discard)
		httpOpts.Client.HTTPClient.Timeout = time.Duration(httpOpts.Timeout) * time.Second
		httpOpts.Client.RetryMax = httpOpts.Retries
		httpOpts.Client.RetryWaitMax = time.Duration(httpOpts.RetryWaitMax) * time.Second
		httpOpts.Client.RetryWaitMin = time.Duration(httpOpts.RetryWaitMin) * time.Second
		res = ButlerManagerReloaderHttp{Method: method, Opts: httpOpts}
		break
	default:
		msg := fmt.Sprintf("unknown reloader method=%s for %s", method, entry)
		return ButlerGenericReloader{}, errors.New(msg)
	}
	return res, err
}

func GetButlerConfigManager(entry string, bc *ButlerConfigSettings) error {
	var (
		err     error
		Manager ButlerManager
	)

	Manager.Name = entry
	Manager.ReloadManager = false

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

	Manager.ManagerOpts = make(map[string]*ButlerManagerOpts)
	for _, m := range Manager.Urls {
		bc.Managers[entry] = &Manager
		mopts := fmt.Sprintf("%s.%s", entry, m)
		opts, err := GetButlerManagerOpts(mopts, bc)
		if err != nil {
			return err
		}
		bc.Managers[entry].ManagerOpts[mopts] = opts
	}

	reloader, err := GetButlerConfigReloader(entry, bc)
	if err != nil {
		return err
	}

	Manager.MustacheSubs, err = ParseMustacheSubs(Manager.MustacheSubsArray)
	if err != nil {
		log.Debugf("GetButlerConfigManager(): could not get mustache subs. err=%s", err.Error())
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

	ButlerConfig.Managers = make(map[string]*ButlerManager)
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
		log.Debugf("ButlerConfigSettings::ParseButlerConfig(): could not parse config. err=%v", err)
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

	ButlerConfig.Managers = make(map[string]*ButlerManager)
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

	// Let's get the path arrays dialed in
	for _, m := range c.Managers {
		for _, u := range m.Urls {
			log.Debugf("ButlerConfigSettings::ParseButlerConfig(): url=%#v", u)
			opts := fmt.Sprintf("%s.%s", m.Name, u)
			log.Debugf("ButlerConfigSettings::ParseButlerConfig(): ManagerOpts=%#v", m.ManagerOpts[opts])
			baseUrl := fmt.Sprintf("%s://%s%s", m.ManagerOpts[opts].Method, u, m.ManagerOpts[opts].UriPath)
			for _, f := range m.ManagerOpts[opts].PrimaryConfig {
				fullUrl := fmt.Sprintf("%s/%s", baseUrl, f)
				fullPath := fmt.Sprintf("%s/%s", m.DestPath, f)
				log.Debugf("ButlerConfigSettings::ParseButlerConfig(): full url to primary config: %s", fullUrl)
				log.Debugf("ButlerConfigSettings::ParseButlerConfig(): full path to primary config: %s", fullPath)
				m.ManagerOpts[opts].AppendPrimaryConfigUrl(fullUrl)
				m.ManagerOpts[opts].AppendPrimaryConfigFile(fullPath)
			}
			for _, f := range m.ManagerOpts[opts].AdditionalConfig {
				fullUrl := fmt.Sprintf("%s/%s", baseUrl, f)
				fullPath := fmt.Sprintf("%s/%s", m.DestPath, f)
				log.Debugf("ButlerConfigSettings::ParseButlerConfig(): full url to additional config: %s", fullUrl)
				log.Debugf("ButlerConfigSettings::ParseButlerConfig(): full path to primary config: %s", fullPath)
				m.ManagerOpts[opts].AppendAdditionalConfigUrl(fullUrl)
				m.ManagerOpts[opts].AppendAdditionalConfigFile(fullPath)
			}
		}
	}

	return nil
}
