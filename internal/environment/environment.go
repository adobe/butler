/*
Copyright 2017 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

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
