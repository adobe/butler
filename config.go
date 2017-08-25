package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/viper"
)

type ButlerConfigClient struct {
	Scheme     string
	HttpClient *retryablehttp.Client
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
	viper.SetConfigType("toml")
	err := viper.ReadConfig(bytes.NewBuffer(config))
	if err != nil {
		return err
	}
	return nil
}

func ButlerConfigHandler() error {
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
		return errors.New(fmt.Sprintf("Did not receive 200 response code for %s. code=%d\n", ButlerConfigUrl, response.StatusCode))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not read response body for %s. err=%s\n", ButlerConfigUrl, err))
	}

	err = ValidateButlerConfig(body)
	if err != nil {
		return err
	}

	if ButlerRawConfig == nil {
		err = ParseButlerConfig(body)
		if err != nil {
			return err
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
