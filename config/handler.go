package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ButlerConfig struct {
	Url          string
	Client       *ButlerConfigClient
	Config       *ButlerConfigSettings
	Interval     int
	Path         string
	Scheme       string
	Timeout      int
	Retries      int
	RetryWaitMin int
	RetryWaitMax int
}

func NewButlerConfig() *ButlerConfig {
	return &ButlerConfig{}
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

func (bc *ButlerConfig) SetInterval(t int) error {
	log.Debugf("ButlerConfig::SetInterval(): setting bc.Interval=%v", t)
	bc.Interval = t
	return nil
}

func (bc *ButlerConfig) GetInterval() int {
	log.Debugf("ButlerConfig::GetInterval(): getting bc.Interval=%v", bc.Interval)
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
	log.Debugf("ButlerConfig::SetwUrl(): setting bc.Url=%s", u)
	bc.Url = u
	return nil
}

func (bc *ButlerConfig) Init() error {
	log.Debugf("ButlerConfig::Init(): initializing butler config.")
	var err error

	if bc.Url == "" {
		ConfigUrl := fmt.Sprintf("%s://%s", bc.Scheme, bc.Path)
		if _, err = url.ParseRequestURI(ConfigUrl); err != nil {
			log.Debugf("ButlerConfig::Init(): could not initialize butler config. err=%s", err)
			return err
		}
		bc.Url = ConfigUrl
	}

	c, err := NewButlerConfigClient(bc.Scheme)
	if err != nil {
		log.Debugf("ButlerConfig::Init(): could not initialize butler config. err=%s", err)
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

	if ButlerRawConfig == nil {
		out, err := bc.Config.ParseButlerConfig(body)
		_ = out
		if err != nil {
		}
	}
	return nil
}
