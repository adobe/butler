package config

import (
	"fmt"
)

var (
	ConfigCache map[string]map[string][]byte
)

type TmpFile struct {
	Name string
	File string
}

type RepoFileEvent struct {
	Success map[string]bool
	Error   map[string]error
	TmpFile map[string]string
}

func (r *RepoFileEvent) SetSuccess(file string, err error) error {
	r.Success[file] = true
	r.Error[file] = err
	return nil
}

func (r *RepoFileEvent) SetFailure(file string, err error) error {
	r.Success[file] = false
	r.Error[file] = err
	return nil
}

func (r *RepoFileEvent) SetTmpFile(file string, tmpfile string) error {
	r.TmpFile[file] = tmpfile
	return nil
}

type ConfigFileMap struct {
	TmpFile string
	Success bool
}

type ConfigSettings struct {
	Managers map[string]*Manager `json:"managers"`
	Globals  ConfigGlobals       `json:"globals"`
}

func (b *ConfigSettings) GetAllConfigLocalPaths() []string {
	var result []string
	for _, m := range b.Managers {
		result = append(result, fmt.Sprintf("%s/%s", m.DestPath, m.PrimaryConfigName))
		for _, o := range m.ManagerOpts {
			for _, f := range o.AdditionalConfigsFullLocalPaths {
				result = append(result, f)
			}
		}
	}
	return result
}

type ConfigGlobals struct {
	Managers          []string `mapstructure:"config-managers" json:"-"`
	SchedulerInterval int      `mapstructure:"scheduler-interval" json:"scheduler-interval"`
	ExitOnFailure     bool     `mapstructure:"exit-on-config-failure" json:"exit-on-failure"`
}
