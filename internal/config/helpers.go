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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/adobe/butler/internal/methods"
	"github.com/adobe/butler/internal/reloaders"
	"github.com/adobe/butler/internal/environment"
	"github.com/adobe/butler/internal/stats"

	"github.com/Jeffail/gabs"
	"github.com/hashicorp/go-retryablehttp"
	// until i get my pr merged
	//"github.com/hoisie/mustache"
	"github.com/mslocrian/mustache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/udhos/equalfile"
	"gopkg.in/yaml.v2"
)

func IsValidScheme(s string) bool {
	var (
		Found = false
	)
	for _, i := range ValidSchemes {
		if strings.ToLower(s) == i {
			Found = true
		}

	}
	return Found
}

// ValidateConfig takes a pointer to an os.File object. It scans over the
// file and ensures that it begins with the proper header, and ends with the
// proper footer. If it does not begin or end with the proper header/footer,
// then an error is returned. If the file passes the checks, a nil is returned.
func ValidateConfig(opts *ValidateOpts) error {
	var (
		err               error
		file              *bytes.Reader
		contentTypeSwitch string
	)

	log.Debugf("ValidateConfig()[count=%v][manager=%v]: checking content-type=%v FileName=%v", cmHandlerCounter, opts.Manager, opts.ContentType, opts.FileName)
	f := opts.Data
	switch t := f.(type) {
	case *os.File:
		newf := f.(*os.File)

		fd, err := os.Open(newf.Name())
		if err != nil {
			log.Errorf("ValidateConfig()[count=%v][manager=%v]: caught error on open err=%#v", cmHandlerCounter, opts.Manager, err.Error())
			return err
		}
		defer fd.Close()

		fi, err := fd.Stat()
		if err != nil {
			log.Errorf("ValidateConfig()[count=%v][manager=%v]: caught error on stat err=%#v", cmHandlerCounter, opts.Manager, err.Error())
			return err
		}

		data := make([]byte, fi.Size())
		_, err = fd.Read(data)
		if err != nil {
			log.Errorf("ValidateConfig()[count=%v][manager=%v]: caught error on fd.Read() err=%#v", cmHandlerCounter, opts.Manager, err.Error())
			return err
		}

		file = bytes.NewReader(data)
	case []byte:
		newf := f.([]byte)
		file = bytes.NewReader(newf)
	default:
		return errors.New(fmt.Sprintf("ValidateConfig()[count=%v][manager=%v]: unknown file type %s for %s", cmHandlerCounter, opts.Manager, t, f))
	}

	if opts.ContentType == "auto" {
		contentTypeSwitch = getFileExtension(opts.FileName)
	} else {
		contentTypeSwitch = opts.ContentType
	}

	switch contentTypeSwitch {
	case "text":
		err = runTextValidate(file, opts.Manager)
	case "json":
		err = runJsonValidate(file, opts.Manager)
	case "yaml":
		err = runYamlValidate(file, opts.Manager)
	default:
		err = errors.New(fmt.Sprintf("unknown content type %s", opts.ContentType))
	}

	if err != nil {
		log.Errorf("ValidateConfig()[count=%v][manager=%v]: returning err=%v for content-type=%v and FileName=%v", cmHandlerCounter, opts.Manager, err.Error(), opts.ContentType, opts.FileName)
		return err
	}

	// let's rewrite a sanitized temporary config file
	err = removeButlerHeaderFooter(opts.Data)
	if err != nil {
		log.Errorf("ValidateConfig()[count=%v][manager=%v]: returning err=%v for content-type=%v and FileName=%v", cmHandlerCounter, opts.Manager, err.Error(), opts.ContentType, opts.FileName)
	}
	return err
}

func checkButlerHeaderFooter(in []byte) bool {
	switch string(in) {
	case butlerHeader:
		return true
	case butlerFooter:
		return true
	default:
		return false
	}
}

func removeButlerHeaderFooter(file interface{}) error {
	var (
		err       error
		in        *os.File
		out       *os.File
		newSource []byte
	)

	switch f := file.(type) {
	case *os.File:
		in, err = os.Open(f.Name())
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			var line []byte
			line = scanner.Bytes()
			if !checkButlerHeaderFooter(line) {
				newSource = append(newSource, line...)
				newSource = append(newSource, []byte("\n")...)
			}
		}
		if err = scanner.Err(); err != nil {
			in.Close()
			return err
		}
		in.Close()

		out, err = os.OpenFile(f.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, bytes.NewReader(newSource))
		out.Sync()
		cerr := out.Close()
		if err != nil {
			return err
		}
		return cerr
	default:
		return nil
	}
}

func runTextValidate(f *bytes.Reader, m string) error {
	var (
		//err error
		configLine    string
		isFirstLine   bool
		isValidHeader bool
		isValidFooter bool
		scanner       *bufio.Scanner
	)
	isFirstLine = true
	isValidHeader = true
	isValidFooter = true
	scanner = bufio.NewScanner(f)

	for scanner.Scan() {
		configLine = scanner.Text()
		// Check that the header is valid
		if isFirstLine {
			if configLine != butlerHeader {
				isValidHeader = false
			}
			isFirstLine = false
		}
	}
	// Check that the footer is valid
	if configLine != butlerFooter {
		if configLine != butlerFooter {
			isValidFooter = false
		}
	}

	if !isValidHeader && !isValidFooter {
		return errors.New(fmt.Sprintf("runTextValidate()[count=%v][manager=%v]: Invalid butler header and footer", cmHandlerCounter, m))
	} else if !isValidHeader {
		return errors.New(fmt.Sprintf("runTextValidate()[count=%v][manager=%v]: Invalid butler header", cmHandlerCounter, m))
	} else if !isValidFooter {
		return errors.New(fmt.Sprintf("runTextValidate()[count=%v][manager=%v]: Invalid butler footer", cmHandlerCounter, m))
	} else {
		return nil
	}
}

func runJsonValidate(f *bytes.Reader, m string) error {
	var (
		err  error
		data []byte
	)

	data, err = ioutil.ReadAll(f)
	if err != nil {
		msg := fmt.Sprintf("runJsonValidate()[count=%v][manager=%v], could not read data from bytes.Reader. err=%v", cmHandlerCounter, m, err.Error())
		return errors.New(msg)
	}

	_, err = gabs.ParseJSON(data)
	if err != nil {
		msg := fmt.Sprintf("runJsonValidate()[count=%v][manager=%v], could not Unmarshal json data into interface. err=%v", cmHandlerCounter, m, err.Error())
		return errors.New(msg)
	}
	return nil
}

func runYamlValidate(f *bytes.Reader, m string) error {
	var (
		err  error
		data []byte
		v    interface{}
	)

	data, err = ioutil.ReadAll(f)
	if err != nil {
		msg := fmt.Sprintf("runYamlValidate()[count=%v][manager=%v]: could not read data from bytes.Reader. err=%v", cmHandlerCounter, m, err.Error())
		return errors.New(msg)
	}

	err = yaml.Unmarshal(data, &v)
	if err != nil {
		msg := fmt.Sprintf("runYamlValidate()[count=%v][manager=%v]: could not Unmarshal yaml data into interface. err=%v", cmHandlerCounter, m, err.Error())
		return errors.New(msg)
	}

	err = runTextValidate(bytes.NewReader(data), m)
	if err != nil {
		msg := fmt.Sprintf("runYamlValidate()[count=%v][manager=%v]: could not verify butler header/footer for yaml data. err=%v", cmHandlerCounter, m, err.Error())
		return errors.New(msg)
	}
	return nil
}

func getFileExtension(file string) string {
	var result string
	file = strings.ToLower(file)
	if strings.HasSuffix(file, "json") {
		result = "json"
	} else if strings.HasSuffix(file, "yaml") {
		result = "yaml"
	} else if strings.HasSuffix(file, "yml") {
		result = "yaml"
	} else {
		result = "text"
	}
	return result
}

func ParseMustacheSubs(pairs []string) (map[string]string, error) {
	var (
		subs map[string]string
	)
	subs = make(map[string]string)

	for _, p := range pairs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		keyvalpairs := strings.Split(p, "=")
		if len(keyvalpairs) != 2 {
			log.Warnf("helpers.ParseMustacheSubs(): invalid key value pair \"%s\"... ignoring.", keyvalpairs)
			continue
		}
		key := strings.TrimSpace(keyvalpairs[0])
		val := environment.GetVar(strings.TrimSpace(keyvalpairs[1]))
		subs[key] = val
	}
	return subs, nil
}

func ValidateMustacheSubs(Subs map[string]string) bool {
	var (
		subEntries map[string]bool
	)
	subEntries = make(map[string]bool)

	// range over the subs and see if the keys match the required list of substitution keys
	for k, _ := range Subs {
		if _, ok := subEntries[k]; ok {
			subEntries[k] = true
		}
	}

	// If any of the sub keys are false, then something is missing
	for _, v := range subEntries {
		if v == false {
			return false
		}
	}
	return true
}

// RenderConfigMustache takes a pointer to an os.File object. It reads the file
// attempts to parse the mustache
func RenderConfigMustache(f *os.File, subs map[string]string) error {
	tmpl, err := mustache.ParseFile(f.Name())
	if err != nil {
		return err
	}

	out := tmpl.Render(subs)

	f, err = os.OpenFile(f.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(out))
	if err != nil {
		return err
	}
	return nil
}

func CompareAndCopy(source string, dest string, m string) bool {
	// Let's compare the source and destination files
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(source, dest)
	if !equal {
		if err != nil {
			log.Errorf("helpers.CompareAndCopy()[count=%v][manager=%v]: caught error from compare. source=%v dest=%v err=%#v", cmHandlerCounter, m, source, dest, err)
		}
		log.Infof("helpers.CompareAndCopy()[count=%v][manager=%v]: Found difference in \"%s.\"  Updating.", cmHandlerCounter, m, dest)
		err = CopyFile(source, dest)
		if err != nil {
			stats.SetButlerWriteVal(stats.FAILURE, stats.GetStatsLabel(dest))
			log.Errorf("helpers.CompareAndCopy()[count=%v][manager=%v]: could not copy source=%v to dest=%v. err=%#v", cmHandlerCounter, m, source, dest, err)
			return false
		}
		stats.SetButlerWriteVal(stats.SUCCESS, stats.GetStatsLabel(dest))
		return true
	} else {
		return false
	}
}

// CopyFile copies the src path string to the dst path string. If there is an
// error, an error is returned, otherwise nil is returned.
func CopyFile(src string, dst string) error {
	var (
		err       error
		in        *os.File
		out       *os.File
		newSource []byte
	)

	// open source
	in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// at this point what we're trying to do is to strip the butler
	// header and footer out of the file
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		var line []byte
		line = scanner.Bytes()
		if !checkButlerHeaderFooter(line) {
			newSource = append(newSource, line...)
			newSource = append(newSource, []byte("\n")...)
		}
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	// open destination
	if _, err = os.Stat(dst); err != nil {
		out, err = os.Create(dst)
	} else {
		out, err = os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0644)
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(out, bytes.NewReader(newSource))
	out.Sync()
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

// CacheConfigs takes in a string of the base directory for
// the config directory and a slice of config file names and
// caches those files into memory. It returns an error
// on the event of error
func CacheConfigs(manager string, files []string) error {
	log.Infof("helpers.CacheConfig()[count=%v][manager=%v]: Storing known good configurations to cache.", cmHandlerCounter, manager)
	if ConfigCache == nil {
		ConfigCache = make(map[string]map[string][]byte)
	}
	ConfigCache[manager] = make(map[string][]byte)
	for _, file := range files {
		out, err := ioutil.ReadFile(file)
		if err != nil {
			msg := fmt.Sprintf("helpers.CacheConfig()[count=%v][manager=%v]: Could not store %s to cache. err=%s", cmHandlerCounter, manager, file, err.Error())
			log.Errorf(msg)
			return errors.New(msg)
		} else {
			ConfigCache[manager][file] = out
		}
	}
	log.Infof("helpers.CacheConfig()[count=%v][manager=%v]: Done storing known good configurations to cache.", cmHandlerCounter, manager)
	stats.SetButlerKnownGoodCachedVal(stats.SUCCESS, manager)
	stats.SetButlerKnownGoodRestoredVal(stats.FAILURE, manager)
	return nil
}

// RestoreCachedConfigs takes in a strint of the base directory for
// the config directory and a slice of config file names
// and restores those files from the cache back to the
// filesystem. It returns an error on the event of an error
func RestoreCachedConfigs(manager string, files []string, cleanFiles bool) error {
	// If we do not have a good configuration cache, then there's nothing for us to do.
	if ConfigCache == nil {
		if cleanFiles {
			log.Infof("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: No current known good configurations in cache. Cleaning configuration...", cmHandlerCounter, manager)
			for _, file := range files {
				log.Warnf("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Removing bad configuration file %s.", cmHandlerCounter, manager, file)
				os.Remove(file)
			}
			log.Infof("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Done cleaning broken configuration. Returning...", cmHandlerCounter, manager)
		}
		stats.SetButlerKnownGoodCachedVal(stats.FAILURE, manager)
		stats.SetButlerKnownGoodRestoredVal(stats.FAILURE, manager)
		return nil
	}

	log.Warnf("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Restoring known good configurations from cache.", cmHandlerCounter, manager)
	for _, file := range files {
		fileData := ConfigCache[manager][file]

		f, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Errorf("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Could not open %s for writing! err=%s.", cmHandlerCounter, manager, file, err.Error())
			continue
		} else {
			count, err := f.Write(fileData)
			if err != nil {
				log.Errorf("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Could not write to %s! err=%s.", cmHandlerCounter, manager, file, err.Error())
				continue
			} else {
				f.Close()
				log.Warnf("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Wrote %d bytes for %s.", cmHandlerCounter, manager, count, file)
			}
		}
	}
	log.Warnf("helpers.RestoreCachedConfigs()[count=%v][manager=%v]: Done restoring known good configurations from cache.", cmHandlerCounter, manager)
	stats.SetButlerKnownGoodCachedVal(stats.FAILURE, manager)
	stats.SetButlerKnownGoodRestoredVal(stats.SUCCESS, manager)
	return nil
}

func GetManagerOpts(entry string, bc *ConfigSettings) (*ManagerOpts, error) {
	var (
		err     error
		MgrOpts ManagerOpts
	)
	err = viper.UnmarshalKey(entry, &MgrOpts)
	if err != nil {
		return &ManagerOpts{}, err
	}

	MgrOpts.ContentType = environment.GetVar(MgrOpts.ContentType)
	if MgrOpts.ContentType == "" {
		MgrOpts.ContentType = "auto"
	}
	switch strings.ToLower(MgrOpts.ContentType) {
	case "auto", "json", "text", "yaml":
		MgrOpts.ContentType = strings.ToLower(MgrOpts.ContentType)
	default:
		msg := fmt.Sprintf("unknown manager.content-type=%v", MgrOpts.ContentType)
		return &ManagerOpts{}, errors.New(msg)
	}

	MgrOpts.RepoPath = filepath.Clean(environment.GetVar(MgrOpts.RepoPath))

	// This means that repo path was == "" and then filepath.Clean sets it to ".".
	// We don't want that!
	if MgrOpts.RepoPath == "." {
		MgrOpts.RepoPath = ""
	}

	switch MgrOpts.Method {
	case "blob", "file", "http", "https", "s3", "S3", "etcd":
		break
	default:
		msg := fmt.Sprintf("unknown manager.method=%v", MgrOpts.Method)
		return &ManagerOpts{}, errors.New(msg)
	}

	for i, _ := range MgrOpts.PrimaryConfig {
		MgrOpts.PrimaryConfig[i] = filepath.Clean(environment.GetVar(MgrOpts.PrimaryConfig[i]))
	}

	var additionalConfig []string
	for i, _ := range MgrOpts.AdditionalConfig {
		cfg := strings.TrimSpace(environment.GetVar(MgrOpts.AdditionalConfig[i]))
		if cfg == "" {
			continue
		} else {
			additionalConfig = append(additionalConfig, cfg)
		}
	}
	MgrOpts.AdditionalConfig = additionalConfig

	repoSplit := strings.Split(entry, ".")
	MgrOpts.Repo = strings.Join(repoSplit[1:len(repoSplit)], ".")

	if len(MgrOpts.PrimaryConfig) < 1 {
		return &ManagerOpts{}, errors.New("no manager.primary-config defined")
	}

	managerNameSlice := strings.Split(entry, ".")
	var managerName string
	if len(managerNameSlice) >= 1 {
		managerName = managerNameSlice[0]

	} else {
		// shouldn't get this, but hey.
		managerName = "unconfigured"
	}

	methodOpts := fmt.Sprintf("%s.%s", entry, MgrOpts.Method)
	mopts, err := methods.New(&managerName, MgrOpts.Method, &methodOpts)
	if err != nil {
		return &ManagerOpts{}, err
	}
	MgrOpts.Opts = mopts

	return &MgrOpts, nil
}

func GetConfigManager(entry string, bc *ConfigSettings) error {
	var (
		err error
		Mgr Manager
	)

	Mgr.Name = entry
	Mgr.ReloadManager = false
	Mgr.GoodCache = false

	err = viper.UnmarshalKey(entry, &Mgr)
	if err != nil {
		return err
	}

	if len(Mgr.Repos) < 1 {
		msg := fmt.Sprintf("No repos configured for manager %s", entry)
		return errors.New(msg)
	}

	envCleanFiles := strings.ToLower(environment.GetVar(Mgr.CfgCleanFiles))
	if envCleanFiles == "true" {
		Mgr.CleanFiles = true
	} else if envCleanFiles == "false" {
		Mgr.CleanFiles = false
	} else {
		Mgr.CleanFiles = false
	}

	envEnableCache := strings.ToLower(environment.GetVar(Mgr.CfgEnableCache))
	if envEnableCache == "true" {
		Mgr.EnableCache = true
	} else {
		Mgr.EnableCache = false
	}

	envManagerTimeoutOk := strings.ToLower(environment.GetVar(Mgr.CfgManagerTimeoutOk))
	if envManagerTimeoutOk == "true" {
		Mgr.ManagerTimeoutOk = true
	} else {
		Mgr.ManagerTimeoutOk = false
	}

	Mgr.CachePath = filepath.Clean(environment.GetVar(Mgr.CachePath))
	if Mgr.EnableCache && Mgr.CachePath == "" {
		msg := fmt.Sprintf("Caching Enabled but manager.cache-path is unset for manager %s", entry)
		return errors.New(msg)
	}

	Mgr.DestPath = filepath.Clean(environment.GetVar(Mgr.DestPath))
	Mgr.PrimaryConfigName = filepath.Clean(environment.GetVar(Mgr.PrimaryConfigName))
	if Mgr.DestPath == "" {
		msg := fmt.Sprintf("No dest-path configured for manager %s", entry)
		return errors.New(msg)
	}

	Mgr.ManagerOpts = make(map[string]*ManagerOpts)
	for _, m := range Mgr.Repos {
		if bc.Managers == nil {
			bc.Managers = make(map[string]*Manager)
		}
		bc.Managers[entry] = &Mgr
		mopts := fmt.Sprintf("%s.%s", entry, m)
		opts, err := GetManagerOpts(mopts, bc)
		if err != nil {
			return err
		}
		bc.Managers[entry].ManagerOpts[mopts] = opts
	}

	reloader, err := reloaders.New(entry)
	if err != nil {
		log.Warnf("helpers.GetConfigManager()[count=%v][manager=%v]: %v.", cmHandlerCounter, entry, err.Error())
		reloader = nil
		// If we've got no reloader for this manager, then there is no need to cache
		log.Debugf("helpers.GetConfigManager()[count=%v][manager=%v]: No reloader has been defined for manager. Setting EnableCache to false", cmHandlerCounter, entry)
		Mgr.EnableCache = false
	}

	Mgr.MustacheSubs, err = ParseMustacheSubs(Mgr.MustacheSubsArray)
	if err != nil {
		log.Debugf("helpers.GetConfigManager()[count=%v][manager=%v]: could not get mustache subs. err=%s", cmHandlerCounter, entry, err.Error())
		return err
	}
	m := bc.Managers[entry]
	m.Reloader = reloader
	bc.Managers[entry] = m
	return nil
}

func ParseConfig(config []byte) error {
	var (
		//handlers []string
		Config  ConfigSettings
		Globals ConfigGlobals
	)
	// The  configuration is in TOML format
	viper.SetConfigType("toml")

	// We grab the config from a remote repo so it's in []byte format. let's see
	// if we can process it.
	err := viper.ReadConfig(bytes.NewBuffer(config))
	if err != nil {
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
	if Config.Globals.SchedulerInterval == 0 {
		Config.Globals.SchedulerInterval = ConfigSchedulerInterval
	}

	log.Debugf("ParseConfig(): globals.config-managers=%#v", Config.Globals.Managers)
	log.Debugf("ParseConfig(): len(globals.config-managers)=%v", len(Config.Globals.Managers))

	// If there are no entries for config-managers, then the Unmarshal will create an empty array
	if len(Config.Globals.Managers) < 1 {
		if Config.Globals.ExitOnFailure {
			log.Fatalf("ParseConfig(): globals.config-managers has no entries! exiting...")
		} else {
			log.Debugf("ParseConfig(): globals.config-managers has no entries!")
			return errors.New("globals.config-managers has no entries. Nothing to do")
		}
	}

	Config.Managers = make(map[string]*Manager)
	// Now let's start processing the managers. This is going
	for _, entry := range Config.Globals.Managers {
		if !viper.IsSet(entry) {
			if Config.Globals.ExitOnFailure {
				log.Fatalf("ParseConfig(): %v is not in the configuration as a manager! exiting...", entry)
			} else {
				log.Debugf("ParseConfig(): %v is not in the configuration as a manager", entry)
				msg := fmt.Sprintf("Cannot find manager for %s", entry)
				return errors.New(msg)
			}
		} else {
			err = GetConfigManager(entry, &Config)
			if err != nil {
				if Config.Globals.ExitOnFailure {
					log.Fatalf("ParseConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
				} else {
					log.Debugf("ParseConfig(): could not retrieve config options for %v. err=%v", entry, err.Error())
					msg := fmt.Sprintf("could not retrieve config options for %v. err=%v", entry, err.Error())
					return errors.New(msg)
				}
			}
		}
	}

	log.Debugf("Config.Managers=%#v", Config.Managers)
	return nil
}

func NewButlerConfig() *ButlerConfig {
	return &ButlerConfig{FirstRun: true}
}

func NewConfigChanEvent() *ConfigChanEvent {
	var (
		c ConfigChanEvent
	)
	c = ConfigChanEvent{}
	c.Repo = make(map[string]*RepoFileEvent)
	return &c
}

func NewConfigClient(scheme string) (*ConfigClient, error) {
	var c ConfigClient
	switch scheme {
	case "http", "https":
		c.Scheme = "http"
		c.HttpClient = retryablehttp.NewClient()
		c.HttpClient.Logger.SetFlags(0)
		c.HttpClient.Logger.SetOutput(ioutil.Discard)
	case "s3", "S3":
		c.Scheme = "s3"
	case "file":
		c.Scheme = "file"
	case "blob":
		c.Scheme = "blob"
	case "etcd":
		c.Scheme = "etcd"
	default:
		errMsg := fmt.Sprintf("Unsupported butler config scheme: %s", scheme)
		return &ConfigClient{}, errors.New(errMsg)
	}
	return &c, nil
}

func NewConfigSettings() *ConfigSettings {
	return &ConfigSettings{}
}
