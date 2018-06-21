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

package config

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/config/methods"
	"git.corp.adobe.com/TechOps-IAO/butler/environment"

	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	ConfigSchedulerInterval = 300
	ValidSchemes            = []string{"blob", "file", "http", "https", "s3", "S3", "etcd"}
)

// butlerHeader and butlerFooter represent the strings that need to be matched
// against in the configuration files. If these entries do not exist in the
// downloaded file, then we cannot be assured that these files are legitimate
// configurations.
const (
	butlerHeader = "#butlerstart"
	butlerFooter = "#butlerend"
)

type ConfigClient struct {
	Scheme     string
	Method     methods.Method
	HttpClient *retryablehttp.Client
}

func (c *ConfigClient) SetTimeout(val int) {
	switch c.Scheme {
	case "http", "https":
		log.Debugf("ConfigClient::SetTimeout(): setting timeout to %v", val)
		c.HttpClient.HTTPClient.Timeout = time.Duration(val) * time.Second
	}
}

func (c *ConfigClient) SetRetryMax(val int) {
	switch c.Scheme {
	case "http", "https":
		log.Debugf("ConfigClient::SetRetryMax(): setting retry max to %v", val)
		c.HttpClient.RetryMax = val
	}
}

func (c *ConfigClient) SetRetryWaitMin(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HttpClient.RetryWaitMin = time.Duration(val) * time.Second
	}
}

func (c *ConfigClient) SetRetryWaitMax(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HttpClient.RetryWaitMax = time.Duration(val) * time.Second
	}
}

func (c *ConfigClient) Get(val *url.URL) (*methods.Response, error) {
	var (
		response *methods.Response
		err      error
	)
	switch val.Scheme {
	case "blob", "file", "http", "https", "s3", "S3", "etcd":
		response, err = c.Method.Get(val)
	default:
		response = &methods.Response{}
		err = errors.New("unsupported scheme")
	}
	return response, err
}

func (c *ConfigSettings) ParseConfig(config []byte) error {
	var (
		Config  ConfigSettings
		Globals ConfigGlobals
	)
	log.Debugf("ConfigSettings::ParseConfig(): entering.")
	// The  configuration is in TOML format
	viper.SetConfigType("toml")

	// We grab the config from a remote repo so it's in []byte format. let's see
	// if we can process it.
	err := viper.ReadConfig(bytes.NewBuffer(config))
	if err != nil {
		log.Debugf("ConfigSettings::ParseConfig(): could not parse config. err=%v", err)
		return err
	}

	Config = ConfigSettings{}

	// Let's start piecing together the globals
	err = viper.UnmarshalKey("globals", &Globals)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}
	Config.Globals = Globals

	// Let's grab some of the global settings
	envExitOnFailure := strings.ToLower(environment.GetVar(Config.Globals.CfgExitOnFailure))
	if envExitOnFailure == "true" {
		Config.Globals.ExitOnFailure = true
	} else if envExitOnFailure == "false" {
		Config.Globals.ExitOnFailure = false
	} else {
		Config.Globals.ExitOnFailure = false
	}

	envSchedulerInterval, _ := strconv.Atoi(environment.GetVar(Config.Globals.CfgSchedulerInterval))
	if envSchedulerInterval == 0 {
		log.Warnf("ConfigSettings::ParseConfig() could not convert %v to integer for scheduler-interval, defaulting to 0. This is probably undesired.", Config.Globals.CfgSchedulerInterval)
		Config.Globals.SchedulerInterval = ConfigSchedulerInterval
	} else {
		Config.Globals.SchedulerInterval = envSchedulerInterval
	}

	Config.Globals.StatusFile = environment.GetVar(Config.Globals.CfgStatusFile)
	if Config.Globals.StatusFile == "" {
		Config.Globals.StatusFile = "/var/tmp/butler.status"
	}

	envEnableHttpLog := strings.ToLower(environment.GetVar(Config.Globals.CfgEnableHttpLog))
	if envEnableHttpLog == "true" {
		Config.Globals.EnableHttpLog = true
		// enable http logging
	} else if envEnableHttpLog == "false" {
		Config.Globals.EnableHttpLog = false
		// disable http logging
	} else {
		Config.Globals.EnableHttpLog = true
		// enable http logging
	}

	log.Debugf("ConfigSettings::ParseConfig(): globals.config-managers=%#v", Config.Globals.Managers)
	log.Debugf("ConfigSettings::ParseConfig(): len(globals.config-managers)=%v", len(Config.Globals.Managers))

	// If there are no entries for config-managers, then the Unmarshal will create an empty array
	if len(Config.Globals.Managers) < 1 {
		if Config.Globals.ExitOnFailure {
			log.Fatalf("ConfigSettings::ParseConfig(): globals.config-managers has no entries! exiting...")
		} else {
			log.Debugf("ConfigSettings::ParseConfig(): globals.config-managers has no entries!")
			return errors.New("globals.config-managers has no entries. Nothing to do")
		}
	}

	Config.Managers = make(map[string]*Manager)
	// Now let's start processing the managers. This is going
	for _, entry := range Config.Globals.Managers {
		log.Debugf("ConfigSettings::ParseConfig(): checking config entry=%s", entry)
		if !viper.IsSet(entry) {
			if Config.Globals.ExitOnFailure {
				log.Fatalf("ConfigSettings::ParseConfig(): %v is not in the configuration as a manager! exiting...", entry)
			} else {
				log.Debugf("ConfigSettings::ParseConfig(): %v is not in the configuration as a manager", entry)
				msg := fmt.Sprintf("Cannot find manager for %s", entry)
				return errors.New(msg)
			}
		} else {
			err = GetConfigManager(entry, &Config)
			if err != nil {
				if Config.Globals.ExitOnFailure {
					log.Fatalf("ConfigSettings::ParseConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
				} else {
					log.Debugf("ConfigSettings::ParseConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
					msg := fmt.Sprintf("could not retrieve config options for %v. err=%v", entry, err.Error())
					return errors.New(msg)
				}
			}
		}
	}

	// Set the values in the config structure
	c.Managers = Config.Managers
	c.Globals = Config.Globals

	// Let's get the path arrays dialed in
	for _, m := range c.Managers {
		for _, u := range m.Repos {
			opts := fmt.Sprintf("%s.%s", m.Name, u)
			m.ManagerOpts[opts].SetParentManager(m.Name)
			baseRemotePath := fmt.Sprintf("%s://%s%s", m.ManagerOpts[opts].Method, u, m.ManagerOpts[opts].RepoPath)
			for _, f := range m.ManagerOpts[opts].PrimaryConfig {
				fullRemotePath := fmt.Sprintf("%s/%s", baseRemotePath, f)
				m.ManagerOpts[opts].AppendPrimaryConfigUrl(fullRemotePath)
				log.Debugf("ConfigSettings::ParseConfig(): full remote path to primary config: %s", fullRemotePath)
			}
			// we've only got one primary config, so we only need the array to have that element
			// we still need to populate the remote paths, since we are merging multiple files
			// into one. This used to be in the above loop
			fullLocalPath := fmt.Sprintf("%s/%s", m.DestPath, m.PrimaryConfigName)
			m.ManagerOpts[opts].AppendPrimaryConfigFile(fullLocalPath)
			log.Debugf("ConfigSettings::ParseConfig(): full local path to primary config: %s", fullLocalPath)
			for _, f := range m.ManagerOpts[opts].AdditionalConfig {
				fullRemotePath := fmt.Sprintf("%s/%s", baseRemotePath, f)
				fullLocalPath := fmt.Sprintf("%s/%s", m.DestPath, f)
				log.Debugf("ConfigSettings::ParseConfig(): full remote path to additional config: %s", fullRemotePath)
				log.Debugf("ConfigSettings::ParseConfig(): full local path to primary config: %s", fullLocalPath)
				m.ManagerOpts[opts].AppendAdditionalConfigUrl(fullRemotePath)
				m.ManagerOpts[opts].AppendAdditionalConfigFile(fullLocalPath)
			}
		}
	}

	return nil
}
