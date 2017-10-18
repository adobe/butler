package reloaders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

func NewHttpReloader(manager string, method string, entry []byte) (Reloader, error) {
	var (
		err    error
		result HttpReloader
		opts   HttpReloaderOpts
	)

	err = json.Unmarshal(entry, &opts)
	if err != nil {
		return result, err
	}

	opts.Client = retryablehttp.NewClient()
	opts.Client.Logger.SetFlags(0)
	opts.Client.Logger.SetOutput(ioutil.Discard)
	opts.Client.HTTPClient.Timeout = time.Duration(opts.Timeout) * time.Second
	opts.Client.RetryMax = opts.Retries
	opts.Client.RetryWaitMax = time.Duration(opts.RetryWaitMax) * time.Second
	opts.Client.RetryWaitMin = time.Duration(opts.RetryWaitMin) * time.Second
	result.Method = method
	result.Opts = opts
	result.Manager = manager
	return result, err
}

type HttpReloader struct {
	Manager string           `json:"-"`
	Method  string           `mapstructure:"method" json:"method"`
	Opts    HttpReloaderOpts `json:"opts"`
}

type HttpReloaderOpts struct {
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

func (h *HttpReloaderOpts) GetClient() *retryablehttp.Client {
	return h.Client
}

func (h HttpReloader) Reload() error {
	var (
		err error
	)

	log.Debugf("HttpReloader::Reload(): reloading manager using http")
	o := h.GetOpts().(HttpReloaderOpts)
	c := o.GetClient()
	// Set the reloader retry policy
	c.CheckRetry = h.ReloaderRetryPolicy
	reloadUrl := fmt.Sprintf("%s://%s:%d%s", h.Method, o.Host, o.Port, o.Uri)

	switch o.Method {
	case "post":
		log.Debugf("HttpReloader::Reload(): posting up!")
		resp, err := c.Post(reloadUrl, o.ContentType, strings.NewReader(o.Payload))
		if err != nil {
			msg := fmt.Sprintf("HttpReloader::Reload(): err=%v", err.Error())
			log.Infof(msg)
			return errors.New(msg)
		}
		if resp.StatusCode == 200 {
			log.Infof("HttpReloader::Reload(): successfully reloaded config. http_code=%d", int(resp.StatusCode))
			// at this point error should be nil, so things are OK
		} else {
			msg := fmt.Sprintf("HttpReloader::Reload(): received bad response from server. reverting to last known good config. http_code=%d", int(resp.StatusCode))
			log.Infof(msg)
			// at this point we should raise an error
			return errors.New(msg)
		}
	default:
		msg := fmt.Sprintf("HttpReloader::Reload(): %s is not a supported reload method", h.Method)
		return errors.New(msg)
	}

	return err

}

func (h HttpReloader) ReloaderRetryPolicy(resp *http.Response, err error) (bool, error) {
	if err != nil {
		return true, err
	}

	// Let's set our reloader stats
	stats.SetButlerReloaderRetry(stats.SUCCESS, h.Manager)

	// Here is our policy override. By default it looks for
	// res.StatusCode >= 500 ...
	if resp.StatusCode == 0 || resp.StatusCode >= 600 {
		return true, nil
	}
	return false, nil
}

func (h HttpReloader) GetMethod() string {
	return h.Method
}
func (h HttpReloader) GetOpts() ReloaderOpts {
	return h.Opts
}

func (h HttpReloader) SetOpts(opts ReloaderOpts) bool {
	h.Opts = opts.(HttpReloaderOpts)
	return true
}
