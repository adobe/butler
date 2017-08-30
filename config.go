package main

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
)

type ButlerConfigClient struct {
	Scheme     string
	HttpClient *retryablehttp.Client
}

type ButlerConfigSettings struct {
	Handlers map[string]ButlerHandler
	Globals  ButlerConfigGlobals
}

type ButlerConfigGlobals struct {
	Handlers          []string
	SchedulerInterval int
	ExitOnFailure     bool
	CleanFiles        bool
}

type ButlerHandler struct {
}

func NewButlerConfigClient(scheme string) (ButlerConfigClient, error) {
	var c ButlerConfigClient
	switch scheme {
	case "http", "https":
		c.Scheme = "http"
		c.HttpClient = retryablehttp.NewClient()
		c.HttpClient.Logger.SetFlags(0)
		c.HttpClient.Logger.SetOutput(ioutil.Discard)
	default:
		errMsg := fmt.Sprintf("Unsupported butler config scheme: %s", scheme)
		return ButlerConfigClient{}, errors.New(errMsg)
	}
	return c, nil
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

func ParseButlerConfig(config []byte) error {
	var (
		//handlers []string
		ButlerConfig ButlerConfigSettings
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

	// Let's grab the exit on failure val
	if viper.IsSet("globals.exit-on-config-failure") {
		log.Debugf("ParseButlerConfig(): setting ButlerConfig.Globals.ExitOnFailure to \"%v\"", viper.GetBool("globals.exit-on-config-failure"))
		ButlerConfig.Globals.ExitOnFailure = viper.GetBool("globals.exit-on-config-failure")
	} else {
		ButlerConfig.Globals.ExitOnFailure = false
		log.Debugf("ParseButlerConfig(): setting ButlerConfig.Globals.ExitOnFailure to \"%v\"", false)
	}

	// Let's grab some of the global settings
	if viper.IsSet("globals.scheduler-interval") {
		ButlerConfig.Globals.SchedulerInterval = viper.GetInt("globals.scheduler-interval")
	} else {
		ButlerConfig.Globals.SchedulerInterval = ConfigSchedulerInterval
	}

	if viper.IsSet("globals.clean-files") {
		ButlerConfig.Globals.CleanFiles = viper.GetBool("globals.config-handlers")
	} else {
		ButlerConfig.Globals.CleanFiles = false
	}

	// We need these handlers. If there are no handlers, then we've really got nothing
	// to do.
	if !viper.IsSet("globals.config-handlers") {
		if ButlerConfig.Globals.ExitOnFailure {
			log.Fatalf("ParseButlerConfig(): globals.config-handlers unset! exiting...")
		} else {
			log.Debugf("ParseButlerConfig(): globals.config-handlers unset!")
			return errors.New("No globals.config-handlers in butler configuration. Nothing to do.")
		}
	} else {
		ButlerConfig.Globals.Handlers = viper.GetStringSlice("globals.config-handlers")
	}
	log.Debugf("ParseButlerConfig(): globals.config-handlers=%#v", ButlerConfig.Globals.Handlers)
	log.Debugf("ParseButlerConfig(): len(globals.config-handlers)=%v", len(ButlerConfig.Globals.Handlers))

	// If no handlers are set. eg: config-handlers = []
	if ButlerConfig.Globals.Handlers == nil {
		if ButlerConfig.Globals.ExitOnFailure {
			log.Fatalf("ParseButlerConfig(): globals.config-handlers = []! exiting...")
		} else {
			log.Debugf("ParseButlerConfig(): globals.config-handlers unset!")
			return errors.New("globals.config-handlers = []. Nothing to do")
		}
	}

	// If somehow the length of the config handlers array is < 1
	// Not sure how to replicate this case, since config-handlers = []
	// returns nil, but would still like to catch it
	if len(ButlerConfig.Globals.Handlers) < 1 {
		if ButlerConfig.Globals.ExitOnFailure {
			log.Fatalf("ParseButlerConfig(): globals.config-handlers has no entries! exiting...")
		} else {
			log.Debugf("ParseButlerConfig(): globals.config-handlers has no entries!")
			return errors.New("globals.config-handlers has no entries. Nothing to do")
		}
	}

	ButlerConfig.Handlers = make(map[string]ButlerHandler)
	// Now let's start processing the handlers. This is going
	for _, entry := range ButlerConfig.Globals.Handlers {
		if !viper.IsSet(entry) {
			log.Debugf("ParseButlerConfig(): %v is not in the configuration as a handler", entry)
		}
	}

	return nil
}

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
