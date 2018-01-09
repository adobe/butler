package environment

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func GetVar(entry interface{}) string {
	// if the length of the entry is less than what we're trying
	// to check against, then this is probably not an environment
	// entry
	switch val := entry.(type) {
	case int:
		return fmt.Sprintf("%d", val)
	case string:
		if len(val) < len("env:") {
			return val
		}
		if strings.ToLower(val[:4]) == "env:" {
			envKey := val[4:]
			envVal := os.Getenv(envKey)
			if envVal == "" {
				log.Warnf("Environment variable %s does not exist.", envKey)
			}
			return envVal
		} else {
			return val
		}
	default:
		return ""
	}
}
