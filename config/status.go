package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

type Status struct {
	Manager map[string]bool `json:"manager"`
}

func ReadManagerStatusFile(statusFile string) (*Status, error) {
	var (
		status Status
	)

	out, err := ioutil.ReadFile(statusFile)
	if err != nil {
		return &status, err
	}

	err = json.Unmarshal(out, &status)
	if err != nil {
		return &status, err
	}
	return &status, nil
}

func WriteManagerStatusFile(statusFile string, status Status) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(statusFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func GetManagerStatus(statusFile string, manager string) bool {
	status, err := ReadManagerStatusFile(statusFile)
	if err != nil {
		log.Debugf("GetManagerStatus(): could not read manager %v, returning false", statusFile)
		return false
	}

	if val, ok := status.Manager[manager]; ok {
		log.Debugf("GetManagerStatus(): found key for %v, returning %v", manager, val)
		return val
	} else {
		log.Debugf("GetManagerStatus(): could not find key entry for %v, returning false", manager)
		return false
	}
}

func SetManagerStatus(statusFile string, manager string, state bool) error {
	var (
		status *Status
	)
	status, err := ReadManagerStatusFile(statusFile)
	if (err != nil) || (status.Manager == nil) {
		status.Manager = make(map[string]bool)
	}

	status.Manager[manager] = state

	return WriteManagerStatusFile(statusFile, *status)
}
