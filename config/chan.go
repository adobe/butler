package config

import (
	"fmt"
	"io"
	"os"
	"sort"

	"git.corp.adobe.com/TechOps-IAO/butler/stats"

	log "github.com/sirupsen/logrus"
)

type ChanEvent interface {
	CanCopyFiles() bool
	CleanTmpFiles() error
	GetTmpFileMap() []TmpFile
	SetSuccess(string, string, error) error
	SetTmpFile(string, string, string) error
	CopyPrimaryConfigFiles() bool
	CopyAdditionalConfigFiles(string) bool
}

type ConfigChanEvent struct {
	HasChanged bool
	TmpFile    *os.File
	ConfigFile *string
	Repo       map[string]*RepoFileEvent
}

func (c *ConfigChanEvent) CanCopyFiles() bool {
	var (
		res bool
	)
	res = true

	log.Debugf("ConfigChanEvent::CanCopyFiles(): seeing if we can copy files")
	for _, r := range c.Repo {
		for _, v := range r.Success {
			if v == false {
				res = false
			}
		}
	}
	log.Debugf("ConfigChanEvent::CanCopyFiles(): returning %v", res)
	return res
}

func (c *ConfigChanEvent) CleanTmpFiles() error {
	log.Debugf("ConfigChanEvent::CleanTmpFiles(): cleaning up temporary files")
	for _, r := range c.Repo {
		for _, f := range r.TmpFile {
			log.Debugf("ConfigChanEvent::CleanTmpFiles(): removing file %#v", f)
			os.Remove(f)
		}
	}

	if c.TmpFile != nil {
		os.Remove(c.TmpFile.Name())
	}
	return nil
}

func (c *ConfigChanEvent) GetTmpFileMap() []TmpFile {
	var (
		keys   []string
		res    []TmpFile
		tmpRes map[string]string
	)
	tmpRes = make(map[string]string)

	for _, r := range c.Repo {
		for k, v := range r.TmpFile {
			keys = append(keys, k)
			tmpRes[k] = v
		}
	}

	// Due to the way that golang handles the ordering of maps (random), we have to
	// enforce a sorted ordering, otherwise we may write config files differently,
	// but with the same data (eg: the merged primary config file), causing an undesired
	// configuration reload
	sort.Strings(keys)
	for _, v := range keys {
		res = append(res, TmpFile{Name: v, File: tmpRes[v]})
	}
	log.Debugf("ConfigChanEvent::GetTmpFileMap(): res=%#v", res)
	return res
}

func (c *ConfigChanEvent) SetSuccess(repo string, file string, err error) error {
	// If c.Repo has not been initialized, do so.
	if c.Repo == nil {
		c.Repo = make(map[string]*RepoFileEvent)
	}
	if _, ok := c.Repo[repo]; !ok {
		rfe := &RepoFileEvent{}
		rfe.Success = make(map[string]bool)
		rfe.Error = make(map[string]error)
		rfe.TmpFile = make(map[string]string)
		c.Repo[repo] = rfe
	}
	c.Repo[repo].SetSuccess(file, err)
	return nil
}

func (c *ConfigChanEvent) SetFailure(repo string, file string, err error) error {
	// If c.Repo has not been initialized, do so.
	if c.Repo == nil {
		c.Repo = make(map[string]*RepoFileEvent)
	}
	if _, ok := c.Repo[repo]; !ok {
		rfe := &RepoFileEvent{}
		rfe.Success = make(map[string]bool)
		rfe.Error = make(map[string]error)
		rfe.TmpFile = make(map[string]string)
		c.Repo[repo] = rfe
	}
	c.Repo[repo].SetFailure(file, err)
	return nil
}

func (c *ConfigChanEvent) SetTmpFile(repo string, file string, tmpfile string) error {
	if _, ok := c.Repo[repo]; ok {
		c.Repo[repo].SetTmpFile(file, tmpfile)
	}
	return nil
}

func (c *ConfigChanEvent) CopyPrimaryConfigFiles() bool {
	log.Debugf("Manager::CopyPrimaryConfigFiles(): entering")
	out, err := os.OpenFile(c.TmpFile.Name(), os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
		stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(*c.ConfigFile))
		c.CleanTmpFiles()
		return false
	} else {
		for _, f := range c.GetTmpFileMap() {
			in, err := os.Open(f.File)
			if err != nil {
				log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f.Name))
				c.CleanTmpFiles()
				out.Close()
				return false
			}
			_, err = io.Copy(out, in)
			if err != nil {
				log.Infof("ConfigChanEvent::CopyPrimaryConfigFiles(): Could not process and merge new %s err=%s.", c.ConfigFile, err.Error())
				stats.SetButlerConfigVal(stats.FAILURE, "local", stats.GetStatsLabel(f.Name))
				c.CleanTmpFiles()
				out.Close()
				return false
			}
			in.Close()
		}
	}
	out.Close()
	return CompareAndCopy(c.TmpFile.Name(), *c.ConfigFile)
}

func (c *ConfigChanEvent) CopyAdditionalConfigFiles(destDir string) bool {
	var (
		IsModified bool
	)
	log.Debugf("Manager::CopyAdditionalConfigFiles(): entering")
	IsModified = false

	for _, f := range c.GetTmpFileMap() {
		destFile := fmt.Sprintf("%s/%s", destDir, f.Name)
		if CompareAndCopy(f.File, destFile) {
			IsModified = true
		}
	}
	return IsModified
}
