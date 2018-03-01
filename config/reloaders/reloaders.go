package reloaders

import (
	"encoding/json"
	"errors"
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
		return NewGenericReloader(entry, "error", []byte(entry))
	}

	// No reloader has been defined. We'll assume that is OK
	// but will let the upstream know and they can handle it
	if result == nil {
		return NewGenericReloaderWithCustomError(entry, "error", errors.New("no reloader has been defined"))
	}
	method := result["method"].(string)
	jsonRes, err := json.Marshal(result[method])
	if err != nil {
		return NewGenericReloader(entry, method, []byte(entry))
	}

	log.Debugf("reloaders.New() method=%v", method)
	switch method {
	case "http", "https":
		return NewHttpReloader(entry, method, jsonRes)
	default:
		return NewGenericReloader(entry, method, jsonRes)
	}
}
