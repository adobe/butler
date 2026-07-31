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

package reloaders

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/adobe/butler/internal/environment"

	log "github.com/sirupsen/logrus"
)

func NewCmdReloader(manager string, method string, entry []byte) (Reloader, error) {
	var (
		err    error
		result CmdReloader
		opts   CmdReloaderOpts
	)

	err = json.Unmarshal(entry, &opts)
	if err != nil {
		return result, err
	}

	newTimeout, _ := strconv.Atoi(environment.GetVar(opts.Timeout))
	if newTimeout == 0 {
		log.Warnf("NewCmdReloader(): could not convert %v to integer for timeout, defaulting to 0. This is probably undesired.", opts.Timeout)
	}
	opts.Timeout = fmt.Sprintf("%d", newTimeout)

	// Let's populate some environment variables
	opts.Command = environment.GetVar(opts.Command)
	for i, arg := range opts.Args {
		opts.Args[i] = environment.GetVar(arg)
	}

	result.Method = method
	result.Opts = opts
	result.Manager = manager

	return result, err
}

type CmdReloader struct {
	Manager string          `json:"-"`
	Counter int             `json:"-"`
	Method  string          `mapstructure:"method" json:"method"`
	Opts    CmdReloaderOpts `json:"opts"`
}

type CmdReloaderOpts struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Timeout string   `json:"timeout"`
}

func (cr CmdReloader) Reload() error {
	var (
		err error
	)

	log.Debugf("CmdReloader::Reload()[count=%v][manager=%v]: reloading manager using cmd", cr.Counter, cr.Manager)
	o := cr.GetOpts().(CmdReloaderOpts)

	if o.Command == "" {
		msg := "no command has been defined for cmd reloader"
		log.Errorf("CmdReloader::Reload()[count=%v][manager=%v]: err=%v", cr.Counter, cr.Manager, msg)
		return NewReloaderError().WithMessage(msg).WithCode(2)
	}

	timeout, _ := strconv.Atoi(o.Timeout)
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, o.Command, o.Args...)

	log.Debugf("CmdReloader::Reload()[count=%v][manager=%v]: running %v %v", cr.Counter, cr.Manager, o.Command, o.Args)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		msg := fmt.Sprintf("command timed out after %vs", timeout)
		log.Errorf("CmdReloader::Reload()[count=%v][manager=%v]: err=%v", cr.Counter, cr.Manager, msg)
		return NewReloaderError().WithMessage(msg).WithCode(1)
	}

	if err != nil {
		msg := fmt.Sprintf("%v. output=%v", err.Error(), string(output))
		log.Errorf("CmdReloader::Reload()[count=%v][manager=%v]: err=%v", cr.Counter, cr.Manager, msg)
		return NewReloaderError().WithMessage(msg).WithCode(1)
	}

	log.Infof("CmdReloader::Reload()[count=%v][manager=%v]: successfully reloaded config. output=%v", cr.Counter, cr.Manager, string(output))
	return nil
}

func (cr CmdReloader) GetMethod() string {
	return cr.Method
}

func (cr CmdReloader) GetOpts() ReloaderOpts {
	return cr.Opts
}

func (cr CmdReloader) SetOpts(opts ReloaderOpts) bool {
	cr.Opts = opts.(CmdReloaderOpts)
	return true
}

func (cr CmdReloader) SetCounter(c int) Reloader {
	cr.Counter = c
	return cr
}
