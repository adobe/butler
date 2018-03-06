package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"git.corp.adobe.com/TechOps-IAO/butler/config/methods"
	"git.corp.adobe.com/TechOps-IAO/butler/config/reloaders"
	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	log "github.com/sirupsen/logrus"
	"strings"
)

type Manager struct {
	Name                string                  `json:"name"`
	Repos               []string                `mapstructure:"repos" json:"repos"`
	CfgCleanFiles       string                  `mapstructure:"clean-files" json:"-"`
	CleanFiles          bool                    `json:"clean-files"`
	GoodCache           bool                    `json:"good-cache"`
	LastRun             time.Time               `json:"last-run"`
	MustacheSubsArray   []string                `mapstructure:"mustache-subs" json:"-"`
	MustacheSubs        map[string]string       `json:"mustache-subs"`
	CfgEnableCache      string                  `mapstructure:"enable-cache" json:"-"`
	EnableCache         bool                    `json:"enable-cache"`
	CachePath           string                  `mapstructure:"cache-path" json:"cache-path"`
	DestPath            string                  `mapstructure:"dest-path" json:"dest-path"`
	PrimaryConfigName   string                  `mapstructure:"primary-config-name" json:"primary-config-name"`
	CfgManagerTimeoutOk string                  `mapstructure:"manager-timeout-ok" json:"-"`
	ManagerTimeoutOk    bool                    `json:"manager-timeout-ok"`
	ManagerOpts         map[string]*ManagerOpts `json:"opts"`
	Reloader            reloaders.Reloader      `mapstructure:"-" json:"reloader,omitempty"`
	ReloadManager       bool                    `json:"-"`
}

type ManagerOpts struct {
	Method                          string         `mapstructure:"method" json:"method"`
	RepoPath                        string         `mapstructure:"repo-path" json:"repo-path"`
	Repo                            string         `json:"repo"`
	PrimaryConfig                   []string       `mapstructure:"primary-config" json:"primary-config"`
	AdditionalConfig                []string       `mapstructure:"additional-config" json:"additional-config"`
	PrimaryConfigsFullUrls          []string       `json:"-"`
	AdditionalConfigsFullUrls       []string       `json:"-"`
	PrimaryConfigsFullLocalPaths    []string       `json:"-"`
	AdditionalConfigsFullLocalPaths []string       `json:"-"`
	ContentType                     string         `mapstructure:"content-type" json:"content-type"`
	Opts                            methods.Method `json:"opts"`
}

func (bm *Manager) Reload() error {
	log.Debugf("Manager::Reload(): reloading %s manager...", bm.Name)
	if bm.Reloader == nil {
		log.Warnf("Manager::Reload(): No reloader defined for %s manager. Moving on...", bm.Name)
		return nil
	} else {
		return bm.Reloader.Reload()
	}
}

func (bm *Manager) DownloadPrimaryConfigFiles(c chan ChanEvent) error {
	var (
		Chan              *ConfigChanEvent
		PrimaryConfigName string
	)

	Chan = NewConfigChanEvent()
	PrimaryConfigName = fmt.Sprintf("%s/%s", bm.DestPath, bm.PrimaryConfigName)
	Chan.ConfigFile = &PrimaryConfigName

	// Create a temporary file for the merged prometheus configurations.
	tmpFile, err := ioutil.TempFile("/tmp", "bcmsfile")
	if err != nil {
		msg := fmt.Sprintf("Manager::DownloadPrimaryConfigFiles(): Could not create temporary file . err=v", err.Error())
		log.Fatal(msg)
	}
	Chan.TmpFile = tmpFile

	// Process the prometheus.yml configuration files
	// We are going to iterate through each of the potential managers configured
	for _, opts := range bm.ManagerOpts {
		for i, u := range opts.GetPrimaryConfigUrls() {
			log.Debugf("Manager::DownloadPrimaryConfigFiles(): i=%v, u=%v", i, u)
			log.Debugf("Manager::DownloadPrimaryConfigFiles(): f=%s", opts.GetPrimaryRemoteConfigFiles()[i])
			f := opts.DownloadConfigFile(u)
			if f == nil {
				stats.SetButlerContactVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])

				// Set this stats global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				stats.SetButlerRemoteRepoUp(stats.FAILURE, bm.Name)

				log.Debugf("Manager::DownloadPrimaryConfigFiles(): download for %s is nil.", u)
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not download file"))
				continue
			} else {
				stats.SetButlerContactVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}
			Chan.SetTmpFile(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], f.Name())

			// For the prometheus.yml we have to do some mustache replacement on downloaded file
			// We are doing this before the header/footer check because YAML parsing doesn't like
			// the mustache entries... so we shuffled this around.
			if err := RenderConfigMustache(f, bm.MustacheSubs); err != nil {
				log.Errorf("%s for %s.", err.Error(), u)
				stats.SetButlerRenderVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				log.Debugf("Manager::DownloadPrimaryConfigFiles(): render for %s is nil.", opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not render file"))
				continue
			} else {
				// stats
				stats.SetButlerRenderVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}

			// Let's ensure that the files starts with #butlerstart and
			// ends with #butlerend. If they do not, then we will assume
			// we did not get a correct configuration, or that there is an
			// issue with the upstream
			filename := opts.GetPrimaryRemoteConfigFiles()[i]
			if err := ValidateConfig(NewValidateOpts().WithContentType(opts.ContentType).WithFileName(filename).WithData(f)); err != nil {
				log.Errorf("%s for %s.", err.Error(), u)
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])

				// Set this stats global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				stats.SetButlerRemoteRepoSanity(stats.FAILURE, bm.Name)

				Chan.SetFailure(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], errors.New("could not validate file"))
				continue
			} else {
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetPrimaryRemoteConfigFiles()[i], nil)
			}
		}
	}

	// Update the channel
	c <- Chan

	return nil
}

func (bm *Manager) DownloadAdditionalConfigFiles(c chan ChanEvent) error {
	var (
		Chan       *ConfigChanEvent
		IsModified bool
	)

	Chan = NewConfigChanEvent()
	IsModified = false
	_ = IsModified

	// Process the additional configuration files
	for _, opts := range bm.ManagerOpts {
		for i, u := range opts.GetAdditionalConfigUrls() {
			log.Debugf("Manager::DownloadAdditionalConfigFiles(): i=%v, u=%v", i, u)
			f := opts.DownloadConfigFile(u)
			if f == nil {
				log.Debugf("Manager::DownloadAdditionalConfigFiles(): download for %s is nil.", u)
				stats.SetButlerContactVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])

				// Set this stats global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				stats.SetButlerRemoteRepoUp(stats.FAILURE, bm.Name)

				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not download file"))
				continue
			} else {
				stats.SetButlerContactVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
				Chan.SetTmpFile(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], f.Name())
			}

			// Let's process some mustache ...
			// NOTE: We USED to do this only for the primary configuration. Unsure how this will
			// affect the additional configurations. we can remove this if there are adverse
			// effects.
			// We are doing this before the header/footer check because YAML parsing doesn't like
			// the mustache entries... so we shuffled this around.
			if err := RenderConfigMustache(f, bm.MustacheSubs); err != nil {
				stats.SetButlerRenderVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not render file"))
				continue
			} else {
				stats.SetButlerRenderVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
			}

			// Let's ensure that the files starts with #butlerstart and
			// ends with #butlerend. IF they do not, then we will assume
			// we did not get a correct configuration, or that there is an
			// issue with the upstream
			filename := opts.GetAdditionalRemoteConfigFiles()[i]
			if err := ValidateConfig(NewValidateOpts().WithContentType(opts.ContentType).WithFileName(filename).WithData(f)); err != nil {
				stats.SetButlerConfigVal(stats.FAILURE, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])

				// Set this stats global as failure here, since we aren't sure whether or not it was a parse error or
				// download error in RunCMHandler()
				stats.SetButlerRemoteRepoSanity(stats.FAILURE, bm.Name)

				Chan.SetFailure(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], errors.New("could not validate file"))
				continue
			} else {
				stats.SetButlerConfigVal(stats.SUCCESS, opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i])
				Chan.SetSuccess(opts.Repo, opts.GetAdditionalRemoteConfigFiles()[i], nil)
			}
		}
	}

	// Update the channel
	c <- Chan
	return nil
}

// PathCleanup
func (bm *Manager) PathCleanup(path string, f os.FileInfo, err error) error {
	var (
		Found bool
	)
	Found = false

	// We don't have to do anything with a directory
	if f.Mode().IsDir() {
		log.Debugf("Manager::PathCleanup(): %s is a directory... returning nil", f.Name())
		return nil
	}

	for _, file := range bm.GetAllLocalPaths() {
		if path == file {
			Found = true
		}
	}

	if !Found {
		message := fmt.Sprintf("Found unknown file \"%s\". deleting...", path)
		log.Debugf("Manager::PathCleanup(): Found unknown file \"%s\". deleting...", path)
		os.Remove(path)
		return errors.New(message)
	}
	return nil
}

func (bm *Manager) GetAllLocalPaths() []string {
	var result []string

	for _, opt := range bm.ManagerOpts {
		for _, f := range opt.PrimaryConfigsFullLocalPaths {
			result = append(result, f)
		}
		for _, f := range opt.AdditionalConfigsFullLocalPaths {
			result = append(result, f)
		}
	}
	return result
}

func (bmo *ManagerOpts) AppendPrimaryConfigUrl(c string) error {
	log.Debugf("ManagerOpts::AppendPrimaryConfigUrl(): adding %s to PrimaryConfigsUrls...", c)
	bmo.PrimaryConfigsFullUrls = append(bmo.PrimaryConfigsFullUrls, c)
	return nil
}

func (bmo *ManagerOpts) AppendPrimaryConfigFile(c string) error {
	log.Debugf("ManagerOpts::AppendPrimaryConfigFile(): adding %s to PrimaryConfigsFullLocalPaths...", c)
	bmo.PrimaryConfigsFullLocalPaths = append(bmo.PrimaryConfigsFullLocalPaths, c)
	return nil
}

func (bmo *ManagerOpts) AppendAdditionalConfigUrl(c string) error {
	log.Debugf("ManagerOpts::AppendAdditionalConfigUrl(): adding %s to AdditionalConfigsUrls...", c)
	bmo.AdditionalConfigsFullUrls = append(bmo.AdditionalConfigsFullUrls, c)
	return nil
}

func (bmo *ManagerOpts) AppendAdditionalConfigFile(c string) error {
	log.Debugf("ManagerOpts::AppendAdditionalConfigFile(): adding %s to AdditionalConfigsFullLocalPaths...", c)
	bmo.AdditionalConfigsFullLocalPaths = append(bmo.AdditionalConfigsFullLocalPaths, c)
	return nil
}

func (bmo *ManagerOpts) GetPrimaryConfigUrls() []string {
	return bmo.PrimaryConfigsFullUrls
}

func (bmo *ManagerOpts) GetPrimaryLocalConfigFiles() []string {
	return bmo.PrimaryConfigsFullLocalPaths
}

func (bmo *ManagerOpts) GetPrimaryRemoteConfigFiles() []string {
	return bmo.PrimaryConfig
}

func (bmo *ManagerOpts) GetAdditionalConfigUrls() []string {
	return bmo.AdditionalConfigsFullUrls
}

func (bmo *ManagerOpts) GetAdditionalLocalConfigFiles() []string {
	return bmo.AdditionalConfigsFullLocalPaths
}

func (bmo *ManagerOpts) GetAdditionalRemoteConfigFiles() []string {
	return bmo.AdditionalConfig
}

// Really need to come up with a better method for this.
func (bmo *ManagerOpts) DownloadConfigFile(file string) *os.File {
	switch bmo.Method {
	case "blob", "file", "http", "https", "s3", "S3":
		tmpFile, err := ioutil.TempFile("/tmp", "bcmsfile")
		if err != nil {
			msg := fmt.Sprintf("ManagerOpts::DownloadConfigFile(): could not create temporary file. err=%v", err)
			log.Fatal(msg)
		}

		if (bmo.Method == "s3") || (bmo.Method == "S3") {
			prefix := bmo.Method + "://"
			file = strings.TrimPrefix(file, prefix)
		}

		if bmo.Method == "file" {
			// the file argument for the Get()'ing configs are passed in like:
			// file://repo/full/path/to/file. We need to strip out file:// and
			// repo to get the actual path on the filesystem. So that is what
			// we are doing here.
			file = fmt.Sprintf("/%s", strings.Join(strings.Split(strings.Split(file, "://")[1], "/")[1:], "/"))
		}

		if bmo.Method == "blob" {
			// the file argument for the Get()'ing configs are passed in like:
			// blob://storageaccount/container/file. We need to strip out blob://
			file = strings.Split(file, "://")[1]
		}

		response, err := bmo.Opts.Get(file)

		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile(): Could not download from %s, err=%s", file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		defer response.GetResponseBody().Close()
		defer tmpFile.Close()

		if response.GetResponseStatusCode() != 200 {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile(): Did not receive 200 response code for %s. code=%v", file, response.GetResponseStatusCode())
			tmpFile = nil
			return tmpFile
		}

		_, err = io.Copy(tmpFile, response.GetResponseBody())
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			log.Errorf("ManagerOpts::DownloadConfigFile(): Could not copy to %s, err=%s", file, err.Error())
			tmpFile = nil
			return tmpFile
		}
		return tmpFile
	default:
		return nil
	}
}
