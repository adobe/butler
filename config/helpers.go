package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"git.corp.adobe.com/TechOps-IAO/butler/config/methods"
	"git.corp.adobe.com/TechOps-IAO/butler/config/reloaders"
	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hoisie/mustache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/udhos/equalfile"
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
//func ValidateConfig(f *os.File) error {
func ValidateConfig(f interface{}) error {
	var (
		configLine    string
		isFirstLine   bool
		isValidHeader bool
		isValidFooter bool
		scanner       *bufio.Scanner
	)
	isFirstLine = true
	isValidHeader = true
	isValidFooter = true

	switch t := f.(type) {
	case *os.File:
		newf := f.(*os.File)
		file, err := os.Open(newf.Name())
		if err != nil {
			return err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	case []byte:
		newf := f.([]byte)
		file := bytes.NewReader(newf)
		scanner = bufio.NewScanner(file)
	default:
		return errors.New(fmt.Sprintf("ValidateConfig(): unknown file type %s for %s", t, f))
	}

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
		return errors.New("Invalid butler header and footer")
	} else if !isValidHeader {
		return errors.New("Invalid butler header")
	} else if !isValidFooter {
		return errors.New("Invalid butler footer")
	} else {
		return nil
	}
}

func ParseMustacheSubs(pairs []string) (map[string]string, error) {
	var (
		subs map[string]string
	)
	subs = make(map[string]string)

	for _, p := range pairs {
		p = strings.TrimSpace(p)
		keyvalpairs := strings.Split(p, "=")
		if len(keyvalpairs) != 2 {
			log.Infof("ParseMustacheSubs(): invalid key value pair \"%s\"... ignoring.", keyvalpairs)
			continue
		}
		key := strings.TrimSpace(keyvalpairs[0])
		val := strings.TrimSpace(keyvalpairs[1])
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

func CompareAndCopy(source string, dest string) bool {
	// Let's compare the source and destination files
	cmp := equalfile.New(nil, equalfile.Options{})
	equal, err := cmp.CompareFile(source, dest)
	if !equal {
		log.Infof("config.CompareAndCopy(): Found difference in \"%s.\"  Updating.", dest)
		err = CopyFile(source, dest)
		if err != nil {
			stats.SetButlerWriteVal(stats.FAILURE, stats.GetStatsLabel(dest))
			log.Infof(err.Error())
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
		err error
		in  *os.File
		out *os.File
	)

	// open source
	in, err = os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// open destination
	if _, err = os.Stat(dst); err != nil {
		out, err = os.Create(dst)
	} else {
		out, err = os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0644)
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
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
	log.Infof("CacheConfig(): Storing known good configurations to cache.")
	if ConfigCache == nil {
		ConfigCache = make(map[string]map[string][]byte)
	}
	ConfigCache[manager] = make(map[string][]byte)
	for _, file := range files {
		out, err := ioutil.ReadFile(file)
		if err != nil {
			msg := fmt.Sprintf("CacheConfig(): Could not store %s to cache. err=%s", file, err.Error())
			log.Infof(msg)
			return errors.New(msg)
		} else {
			ConfigCache[manager][file] = out
		}
	}
	log.Infof("CacheConfig(): Done storing known good configurations to cache")
	stats.SetButlerKnownGoodCachedVal(stats.SUCCESS, manager)
	stats.SetButlerKnownGoodRestoredVal(stats.FAILURE, manager)
	return nil
}

// RestoreCachedConfigs takes in a strint of the base directory for
// the config directory and a slice of config file names
// and restores those files from the cache back to the
// filesystem. It returns an error on the event of an error
func RestoreCachedConfigs(manager string, files []string) error {
	// If we do not have a good configuration cache, then there's nothing for us to do.
	if ConfigCache == nil {
		log.Infof("RestoreCachedConfigFs(): No current known good configurations in cache. Cleaning configuration...")
		for _, file := range files {
			log.Infof("RestoreCachedConfigs(): Removing bad Prometheus configuration file %s.", file)
			os.Remove(file)
		}
		log.Infof("RestoreCachedConfigs(): Done cleaning broken configuration. Returning...")
		stats.SetButlerKnownGoodCachedVal(stats.FAILURE, manager)
		stats.SetButlerKnownGoodRestoredVal(stats.FAILURE, manager)
		return nil
	}

	log.Infof("RestoreCachedConfigs(): Restoring known good Prometheus configurations from cache.")
	for _, file := range files {
		fileData := ConfigCache[manager][file]

		f, err := os.OpenFile(file, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Infof("RestoreCachedConfigs(): Could not open %s for writing! err=%s.", file, err.Error())
			continue
		} else {
			count, err := f.Write(fileData)
			if err != nil {
				log.Infof("RestoreCachedConfigs(): Could write to %s! err=%s.", file, err.Error())
				continue
			} else {
				f.Close()
				log.Infof("RestoreCachedConfigs(): Wrote %d bytes for %s.", count, file)
			}
		}
	}
	log.Infof("RestoreCachedConfigs(): Done restoring known good Prometheus configurations from cache.")
	stats.SetButlerKnownGoodCachedVal(stats.FAILURE, manager)
	stats.SetButlerKnownGoodRestoredVal(stats.SUCCESS, manager)
	return nil
}

func ParseConfigManager(config []byte) (Manager, error) {
	return Manager{}, nil
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

	switch MgrOpts.Method {
	case "http", "https":
		break
	default:
		msg := fmt.Sprintf("unknown manager.method=%v", MgrOpts.Method)
		return &ManagerOpts{}, errors.New(msg)
	}

	if MgrOpts.RepoPath == "" {
		return &ManagerOpts{}, errors.New("no manager.repo-path defined")
	}

	repoSplit := strings.Split(entry, ".")
	MgrOpts.Repo = strings.Join(repoSplit[1:len(repoSplit)], ".")

	if len(MgrOpts.PrimaryConfig) < 1 {
		return &ManagerOpts{}, errors.New("no manager.primary-config defined")
	}

	managerNameSlice := strings.Split(entry, ".")
	var managerName string
	if len(managerName) >= 1 {
		managerName = managerNameSlice[0]
	
	} else {
		// shouldn't get this, but hey.
		managerName = "unconfigured"
	}
	methodOpts := fmt.Sprintf("%s.%s", entry, MgrOpts.Method)
	mopts, err := methods.New(managerName, MgrOpts.Method, methodOpts)
	MgrOpts.Opts = mopts

	return &MgrOpts, nil
}

func GetConfigManager(entry string, bc *ConfigSettings) error {
	var (
		err     error
		Manager Manager
	)

	Manager.Name = entry
	Manager.ReloadManager = false

	err = viper.UnmarshalKey(entry, &Manager)
	if err != nil {
		return err
	}

	if len(Manager.Repos) < 1 {
		msg := fmt.Sprintf("No repos configured for manager %s", entry)
		return errors.New(msg)
	}

	if Manager.DestPath == "" {
		msg := fmt.Sprintf("No dest-path configured for manager %s", entry)
		errors.New(msg)
	}

	Manager.ManagerOpts = make(map[string]*ManagerOpts)
	for _, m := range Manager.Repos {
		bc.Managers[entry] = &Manager
		mopts := fmt.Sprintf("%s.%s", entry, m)
		opts, err := GetManagerOpts(mopts, bc)
		if err != nil {
			return err
		}
		bc.Managers[entry].ManagerOpts[mopts] = opts
	}

	//reloader, err := GetConfigReloader(entry, bc)
	reloader, err := reloaders.New(entry)
	if err != nil {
		return err
	}

	Manager.MustacheSubs, err = ParseMustacheSubs(Manager.MustacheSubsArray)
	if err != nil {
		log.Debugf("GetConfigManager(): could not get mustache subs. err=%s", err.Error())
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
			//Config.Managers[entry] = Manager{}
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
	default:
		errMsg := fmt.Sprintf("Unsupported butler config scheme: %s", scheme)
		return &ConfigClient{}, errors.New(errMsg)
	}
	return &c, nil
}

func NewConfigSettings() *ConfigSettings {
	return &ConfigSettings{}
}
