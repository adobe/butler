/*
Copyright 2017-2026 Adobe. All rights reserved.
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

	"github.com/adobe/butler/internal/environment"
	"github.com/adobe/butler/internal/metrics"
	"github.com/adobe/butler/internal/methods"

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

type ButlerConfigOpts struct {
	InsecureSkipVerify bool
	LogLevel           log.Level
	URL                *url.URL
}

type ConfigClient struct {
	Scheme     string
	Method     methods.Method
	HTTPClient *retryablehttp.Client
}

func (c *ConfigClient) SetTimeout(val int) {
	switch c.Scheme {
	case "http", "https":
		log.Debugf("ConfigClient::SetTimeout(): setting timeout to %v", val)
		c.HTTPClient.HTTPClient.Timeout = time.Duration(val) * time.Second
	}
}

func (c *ConfigClient) SetRetryMax(val int) {
	switch c.Scheme {
	case "http", "https":
		log.Debugf("ConfigClient::SetRetryMax(): setting retry max to %v", val)
		c.HTTPClient.RetryMax = val
	}
}

func (c *ConfigClient) SetRetryWaitMin(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HTTPClient.RetryWaitMin = time.Duration(val) * time.Second
	}
}

func (c *ConfigClient) SetRetryWaitMax(val int) {
	switch c.Scheme {
	case "http", "https":
		c.HTTPClient.RetryWaitMax = time.Duration(val) * time.Second
	}
}

func (c *ConfigClient) Get(val *url.URL) (*methods.Response, error) {
	var (
		response *methods.Response
		err      error
	)
	if IsValidScheme(val.Scheme) {
		response, err = c.Method.Get(val)
	} else {
		response = &methods.Response{}
		err = errors.New("unsupported scheme")
	}
	return response, err
}

func (c *ConfigSettings) ParseConfig(config []byte) error {
	var (
		Config  ConfigSettings
		Globals ConfigGlobals
		path    string
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

	envEnableHTTPLog := strings.ToLower(environment.GetVar(Config.Globals.CfgEnableHTTPLog))
	if envEnableHTTPLog == "true" {
		Config.Globals.EnableHTTPLog = true
		// enable http logging
	} else if envEnableHTTPLog == "false" {
		Config.Globals.EnableHTTPLog = false
		// disable http logging
	} else {
		Config.Globals.EnableHTTPLog = true
		// enable http logging
	}

	// Let's determine the http proto and the port
	envHTTPPort, _ := strconv.Atoi(environment.GetVar(Config.Globals.CfgHTTPPort))
	if envHTTPPort == 0 {
		Config.Globals.HTTPPort = 8080
	} else {
		Config.Globals.HTTPPort = envHTTPPort
	}

	Config.Globals.HTTPProto = strings.ToLower(environment.GetVar(Config.Globals.CfgHTTPProto))
	if (Config.Globals.HTTPProto != "http") && (Config.Globals.HTTPProto != "https") {
		Config.Globals.HTTPProto = "http"
	}

	if Config.Globals.HTTPProto == "https" {
		Config.Globals.HTTPTLSCert = environment.GetVar(Config.Globals.CfgHTTPTLSCert)
		Config.Globals.HTTPTLSKey = environment.GetVar(Config.Globals.CfgHTTPTLSKey)
		if (Config.Globals.HTTPTLSCert == "") || (Config.Globals.HTTPTLSKey == "") {
			if Config.Globals.ExitOnFailure {
				log.Fatalf("ConfigSetings::ParseConfig(): globlals.http-proto set to \"https\" but no cert and/or key defined! exiting...")
			} else {
				log.Debugf("ConfigSetings::ParseConfig(): globlals.http-proto set to \"https\" but no cert and/or key defined")
				return errors.New("globals.http-proto set to https but no tls options defined.")
			}
		}
	}

	// If there are no entries for config-managers, then the Unmarshal will create an empty array
	if len(Config.Globals.Managers) < 1 {
		if Config.Globals.ExitOnFailure {
			log.Fatalf("ConfigSettings::ParseConfig(): globals.config-managers has no entries! exiting...")
		} else {
			log.Debugf("ConfigSettings::ParseConfig(): globals.config-managers has no entries!")
			return errors.New("globals.config-managers has no entries. Nothing to do")
		}
	}

	// Snapshot the metric identities known under the previous configuration so
	// that, once the new configuration has been parsed, we can delete any
	// Prometheus series for managers/repos/files that no longer exist. Without
	// this, removed managers or repos leave their last-reported gauge values
	// stuck in /metrics forever, since prometheus.GaugeVec entries persist
	// until explicitly deleted.
	oldMetricIDs := collectMetricIdentities(c.Managers)

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
			repo := strings.Replace(u, "/", "", -1)
			// stripping a leading slash
			if strings.HasPrefix(m.ManagerOpts[opts].RepoPath, "/") {
				path = strings.Replace(m.ManagerOpts[opts].RepoPath, "/", "", 1)
			} else {
				path = m.ManagerOpts[opts].RepoPath
			}
			baseRemotePath := fmt.Sprintf("%s://%s/%s", m.ManagerOpts[opts].Method, repo, path)
			for _, f := range m.ManagerOpts[opts].PrimaryConfig {
				fullRemotePath := fmt.Sprintf("%s/%s", baseRemotePath, f)
				m.ManagerOpts[opts].AppendPrimaryConfigRemotePath(fullRemotePath)
				log.Debugf("ConfigSettings::ParseConfig(): full remote path to primary config: %s", fullRemotePath)
			}
			// we've only got one primary config, so we only need the array to have that element
			// we still need to populate the remote paths, since we are merging multiple files
			// into one. This used to be in the above loop
			fullLocalPath := fmt.Sprintf("%s/%s", m.DestPath, m.PrimaryConfigName)
			m.ManagerOpts[opts].AppendPrimaryLocalConfigFile(fullLocalPath)
			log.Debugf("ConfigSettings::ParseConfig(): full local path to primary config: %s", fullLocalPath)
			for _, f := range m.ManagerOpts[opts].AdditionalConfig {
				fullRemotePath := fmt.Sprintf("%s/%s", baseRemotePath, f)
				fullLocalPath := fmt.Sprintf("%s/%s", m.DestPath, f)
				log.Debugf("ConfigSettings::ParseConfig(): full remote path to additional config: %s", fullRemotePath)
				log.Debugf("ConfigSettings::ParseConfig(): full local path to primary config: %s", fullLocalPath)
				m.ManagerOpts[opts].AppendAdditionalConfigRemotePath(fullRemotePath)
				m.ManagerOpts[opts].AppendAdditionalLocalConfigFile(fullLocalPath)
			}
		}
	}

	// Now that the new configuration is fully built, diff it against the
	// snapshot taken before the reparse and delete any stale metric series
	// for managers/repos/files that disappeared from the configuration.
	newMetricIDs := collectMetricIdentities(c.Managers)
	deleteStaleMetrics(oldMetricIDs, newMetricIDs)

	return nil
}

// metricIdentities holds the set of label combinations that the various
// butler Prometheus gauges are currently reporting under, keyed so that they
// can be diffed between successive configuration parses.
type metricIdentities struct {
	managers  map[string]bool
	repoFiles map[[2]string]bool
	files     map[string]bool
}

// collectMetricIdentities walks a manager map and enumerates every label
// combination ("manager"; "repo"+"config_file"; "config_file") that butler's
// metrics package could currently have a gauge series for.
func collectMetricIdentities(managers map[string]*Manager) metricIdentities {
	ids := metricIdentities{
		managers:  make(map[string]bool),
		repoFiles: make(map[[2]string]bool),
		files:     make(map[string]bool),
	}

	for name, m := range managers {
		if m == nil {
			continue
		}
		ids.managers[name] = true

		for _, opts := range m.ManagerOpts {
			if opts == nil {
				continue
			}
			for _, f := range opts.GetPrimaryRemoteConfigFiles() {
				ids.repoFiles[[2]string{opts.Repo, f}] = true
			}
			for _, f := range opts.GetAdditionalRemoteConfigFiles() {
				ids.repoFiles[[2]string{opts.Repo, f}] = true
			}
		}

		if m.PrimaryConfigName != "" {
			ids.files[metrics.GetStatsLabel(fmt.Sprintf("%s/%s", m.DestPath, m.PrimaryConfigName))] = true
		}
		for _, opts := range m.ManagerOpts {
			if opts == nil {
				continue
			}
			for _, f := range opts.GetAdditionalRemoteConfigFiles() {
				ids.files[metrics.GetStatsLabel(fmt.Sprintf("%s/%s", m.DestPath, f))] = true
			}
		}
	}

	return ids
}

// deleteStaleMetrics removes every Prometheus series present in oldIDs but
// absent from newIDs, i.e. everything that belonged to a manager, repo, or
// file that has been removed from the butler configuration.
func deleteStaleMetrics(oldIDs metricIdentities, newIDs metricIdentities) {
	for manager := range oldIDs.managers {
		if !newIDs.managers[manager] {
			log.Infof("ConfigSettings::ParseConfig(): manager=%s no longer in configuration. deleting its metrics.", manager)
			metrics.DeleteButlerManagerVals(manager)
		}
	}

	for key := range oldIDs.repoFiles {
		if !newIDs.repoFiles[key] {
			log.Infof("ConfigSettings::ParseConfig(): repo=%s config_file=%s no longer in configuration. deleting its metrics.", key[0], key[1])
			metrics.DeleteButlerRepoFileVals(key[0], key[1])
		}
	}

	for file := range oldIDs.files {
		if !newIDs.files[file] {
			log.Infof("ConfigSettings::ParseConfig(): config_file=%s no longer in configuration. deleting its metrics.", file)
			metrics.DeleteButlerWriteVal(file)
		}
	}
}
