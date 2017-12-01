package methods

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	//log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewHttpMethod(manager *string, entry *string) (Method, error) {
	var (
		err    error
		result HttpMethod
	)

	if (manager != nil) && (entry != nil) {
		err = viper.UnmarshalKey(*entry, &result)
		if err != nil {
			return result, err
		}
	}
	result.Client = retryablehttp.NewClient()
	result.Client.Logger.SetFlags(0)
	result.Client.Logger.SetOutput(ioutil.Discard)
	result.Client.HTTPClient.Timeout = time.Duration(result.Timeout) * time.Second
	result.Client.RetryMax = result.Retries
	result.Client.RetryWaitMax = time.Duration(result.RetryWaitMax) * time.Second
	result.Client.RetryWaitMin = time.Duration(result.RetryWaitMin) * time.Second
	result.Client.CheckRetry = result.MethodRetryPolicy
	result.Manager = manager
	return result, err
}

type HttpMethod struct {
	Client       *retryablehttp.Client `json:"-"`
	Manager      *string               `json:"-"`
	Retries      int                   `mapstructure:"retries" json:"retries"`
	RetryWaitMax int                   `mapstructure:"retry-wait-max" json:"retry-wait-max"`
	RetryWaitMin int                   `mapstructure:"retry-wait-min" json:"retry-wait-min"`
	Timeout      int                   `mapstructure:"timeout" json:"timeout"`
}

func (h HttpMethod) Get(file string) (*Response, error) {
	var res Response
	r, err := h.Client.Get(file)
	if err == nil {
		res.body = r.Body
		res.statusCode = r.StatusCode
	}
	return &res, err
}

func (h *HttpMethod) MethodRetryPolicy(resp *http.Response, err error) (bool, error) {
	// This is actually the default RetryPolicy from the go-retryablehttp library. The only
	// change is the stats monitor. We want to keep track of all the reload failures.
	if (err != nil) && (h.Manager != nil) {
		opErr := err.(*url.Error)
		stats.SetButlerContactRetryVal(stats.SUCCESS, *h.Manager, stats.GetStatsLabel(opErr.URL))
		return true, err
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || resp.StatusCode >= 500 {
		if h.Manager != nil {
			stats.SetButlerContactRetryVal(stats.SUCCESS, *h.Manager, stats.GetStatsLabel(resp.Request.RequestURI))
		}
		return true, nil
	}

	return false, nil
}
