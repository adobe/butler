package config

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

type ManagerReloader interface {
	Reload() error
	GetMethod() string
	GetOpts() ReloaderOpts
	SetOpts(ReloaderOpts) bool
}

type ReloaderOpts interface {
}

type ManagerReloaderHttp struct {
	Method string `mapstructer:"method"`
	Opts   ManagerReloaderHttpOpts
}

type ManagerReloaderHttpOpts struct {
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

func (b *ManagerReloaderHttpOpts) GetClient() *retryablehttp.Client {
	return b.Client
}

func (r ManagerReloaderHttp) Reload() error {
	var (
		err error
	)
	log.Debugf("ManagerReloaderHttp::Reload() reloading manager using http")
	o := r.GetOpts().(ManagerReloaderHttpOpts)
	c := o.GetClient()
	// Set the reloader retry policy
	c.CheckRetry = r.ManagerReloadRetryPolicy
	reloadUrl := fmt.Sprintf("%s://%s:%d%s", r.Method, o.Host, o.Port, o.Uri)

	switch o.Method {
	case "post":
		log.Debugf("ManagerReloaderHttp::Reload(): posting up!")
		resp, err := c.Post(reloadUrl, o.ContentType, strings.NewReader(o.Payload))
		if err != nil {
			msg := fmt.Sprintf("ManagerReloaderHttp::Reload(): err=%v", err.Error())
			log.Infof(msg)
			stats.SetButlerReloadVal(stats.FAILURE)
			return errors.New(msg)
		}
		if resp.StatusCode == 200 {
			log.Infof("ManagerReloaderHttp::Reload(): successfully reloaded config. http_code=%d", int(resp.StatusCode))
			stats.SetButlerKnownGoodCachedVal(stats.SUCCESS)
			stats.SetButlerKnownGoodRestoredVal(stats.FAILURE)
			stats.SetButlerReloadVal(stats.SUCCESS)
			// at this point error should be nil, so things are OK
		} else {
			msg := fmt.Sprintf("ManagerReloaderHttp::Reload(): received bad response from server. reverting to last known good config. http_code=%d", int(resp.StatusCode))
			log.Infof(msg)
			stats.SetButlerKnownGoodCachedVal(stats.FAILURE)
			stats.SetButlerKnownGoodRestoredVal(stats.FAILURE)
			stats.SetButlerReloadVal(stats.FAILURE)
			// at this point we should raise an error
			return errors.New(msg)
		}
	default:
		msg := fmt.Sprintf("ManagerReloaderHttp::Reload(): %s is not a supported reload method", r.Method)
		return errors.New(msg)
	}

	return err
}

func (r ManagerReloaderHttp) ManagerReloadRetryPolicy(resp *http.Response, err error) (bool, error) {
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

func (r ManagerReloaderHttp) GetMethod() string {
	return r.Method
}
func (r ManagerReloaderHttp) GetOpts() ReloaderOpts {
	return r.Opts
}

func (r ManagerReloaderHttp) SetOpts(opts ReloaderOpts) bool {
	r.Opts = opts.(ManagerReloaderHttpOpts)
	return true
}

type GenericReloader struct {
	Opts GenericReloaderOpts
}
type GenericReloaderOpts struct{}

func (r GenericReloader) Reload() error {
	var (
		res error
	)
	return res
}
func (r GenericReloader) GetMethod() string {
	return "none"
}
func (r GenericReloader) GetOpts() ReloaderOpts {
	return r.Opts
}

func (r GenericReloader) SetOpts(opts ReloaderOpts) bool {
	r.Opts = opts.(GenericReloaderOpts)
	return true
}
