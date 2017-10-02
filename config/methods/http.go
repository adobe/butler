package methods

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/viper"
)

func NewHttpMethod(entry string) (Method, error) {
	var (
		err    error
		result HttpMethod
	)

	err = viper.UnmarshalKey(entry, &result)
	if err != nil {
		return result, err
	}
	result.Client = retryablehttp.NewClient()
	result.Client.Logger.SetFlags(0)
	result.Client.Logger.SetOutput(ioutil.Discard)
	result.Client.HTTPClient.Timeout = time.Duration(result.Timeout) * time.Second
	result.Client.RetryMax = result.Retries
	result.Client.RetryWaitMax = time.Duration(result.RetryWaitMax) * time.Second
	result.Client.RetryWaitMin = time.Duration(result.RetryWaitMin) * time.Second
	return result, err
}

type HttpMethod struct {
	Client       *retryablehttp.Client `json:"-"`
	Retries      int                   `mapstructure:"retries" json:"retries"`
	RetryWaitMax int                   `mapstructure:"retry-wait-max" json:"retry-wait-max"`
	RetryWaitMin int                   `mapstructure:"retry-wait-min" json:"retry-wait-min"`
	Timeout      int                   `mapstructure:"timeout" json:"timeout"`
}

func (m HttpMethod) Get(file string) (*http.Response, error) {
	res, err := m.Client.Get(file)
	return res, err
}
