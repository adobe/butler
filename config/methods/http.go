package methods

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/environment"
	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
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

	newTimeout, _ := strconv.Atoi(environment.GetVar(result.Timeout))
	if newTimeout == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for timeout, defaulting to 0. This is probably undesired.", result.Timeout)
	}

	newRetries, _ := strconv.Atoi(environment.GetVar(result.Retries))
	if newRetries == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for retries, defaulting to 0. This is probably undesired.", result.Retries)
	}

	newRetryWaitMax, _ := strconv.Atoi(environment.GetVar(result.RetryWaitMax))
	if newRetryWaitMax == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for retry-wait-max, defaulting to 0. This is probably undesired.", result.RetryWaitMax)
	}

	newRetryWaitMin, _ := strconv.Atoi(environment.GetVar(result.RetryWaitMin))
	if newRetryWaitMin == 0 {
		log.Warnf("NewHttpMethod(): could not convert %v to integer for retry-wait-min, defaulting to 0. This is probably undesired.", result.RetryWaitMin)
	}

	result.Client = retryablehttp.NewClient()
	result.Client.Logger.SetFlags(0)
	result.Client.Logger.SetOutput(ioutil.Discard)
	result.Client.HTTPClient.Timeout = time.Duration(newTimeout) * time.Second
	result.Client.RetryMax = newRetries
	result.Client.RetryWaitMax = time.Duration(newRetryWaitMax) * time.Second
	result.Client.RetryWaitMin = time.Duration(newRetryWaitMin) * time.Second
	result.Client.CheckRetry = result.MethodRetryPolicy
	result.Manager = manager
	return result, err
}

type HttpMethod struct {
	Client       *retryablehttp.Client `json:"-"`
	Manager      *string               `json:"-"`
	Retries      string                `mapstructure:"retries" json:"retries"`
	RetryWaitMax string                `mapstructure:"retry-wait-max" json:"retry-wait-max"`
	RetryWaitMin string                `mapstructure:"retry-wait-min" json:"retry-wait-min"`
	Timeout      string                `mapstructure:"timeout" json:"timeout"`
}

func (h HttpMethod) Get(file string) (*Response, error) {
	var res Response
	r, err := h.Client.Get(file)
	if err != nil {
		return &Response{}, err
	}
	res.body = r.Body
	res.statusCode = r.StatusCode
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

	// resp can be nil if the remote host is unreachble of otherwise does not exist
	if resp != nil {
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
	}

	return false, nil
}
