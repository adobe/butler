package reloaders

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Reloader interface {
	Reload() error
	GetMethod() string
	GetOpts() ReloaderOpts
	SetOpts(ReloaderOpts) bool
}

type ReloaderOpts interface {
}

func New(entry string) (Reloader, error) {
	var (
		err    error
		result map[string]interface{}
	)

	key := fmt.Sprintf("%s.reloader", entry)

	err = viper.UnmarshalKey(key, &result)
	if err != nil {
		log.Debugf("reloaders.New(): could not Unmarshal config key: %v", key)
		return NewGenericReloader("error", []byte(entry))
	}

	if result == nil {
		log.Debugf("reloaders.New(): reloader nil. check butler config for reloader section")
		return NewGenericReloader("error", []byte("reloader nil. check config for reloader"))
	}

	method := result["method"].(string)
	jsonRes, err := json.Marshal(result[method])
	if err != nil {
		return NewGenericReloader(method, []byte(entry))
	}

	switch method {
	case "http", "https":
		return NewHttpReloader(method, jsonRes)
	default:
		return NewGenericReloader(method, jsonRes)
	}
}
