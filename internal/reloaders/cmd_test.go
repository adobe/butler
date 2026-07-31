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
	"encoding/json"
	"os"

	. "gopkg.in/check.v1"
)

func (s *ReloaderTestSuite) TestNewCmdReloader(c *C) {
	opts := CmdReloaderOpts{
		Command: "/bin/true",
		Args:    []string{"foo", "bar"},
		Timeout: "10",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	cmdReloader := reloader.(CmdReloader)
	c.Assert(cmdReloader.Method, Equals, "cmd")
	c.Assert(cmdReloader.Manager, Equals, "test-manager")
	c.Assert(cmdReloader.Opts.Command, Equals, "/bin/true")
	c.Assert(cmdReloader.Opts.Args, DeepEquals, []string{"foo", "bar"})
}

func (s *ReloaderTestSuite) TestNewCmdReloaderInvalidJSON(c *C) {
	_, err := NewCmdReloader("test-manager", "cmd", []byte("invalid json"))
	c.Assert(err, NotNil)
}

func (s *ReloaderTestSuite) TestCmdReloaderGetMethod(c *C) {
	reloader := CmdReloader{Method: "cmd"}
	c.Assert(reloader.GetMethod(), Equals, "cmd")
}

func (s *ReloaderTestSuite) TestCmdReloaderGetOpts(c *C) {
	opts := CmdReloaderOpts{Command: "/bin/true"}
	reloader := CmdReloader{Opts: opts}
	result := reloader.GetOpts().(CmdReloaderOpts)
	c.Assert(result.Command, Equals, "/bin/true")
}

func (s *ReloaderTestSuite) TestCmdReloaderSetOpts(c *C) {
	reloader := CmdReloader{}
	newOpts := CmdReloaderOpts{Command: "/bin/false"}
	result := reloader.SetOpts(newOpts)
	c.Assert(result, Equals, true)
}

func (s *ReloaderTestSuite) TestCmdReloaderSetCounter(c *C) {
	reloader := CmdReloader{Counter: 0}
	result := reloader.SetCounter(5)
	cmdResult := result.(CmdReloader)
	c.Assert(cmdResult.Counter, Equals, 5)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadSuccess(c *C) {
	opts := CmdReloaderOpts{
		Command: "/bin/true",
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, IsNil)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadFailure(c *C) {
	opts := CmdReloaderOpts{
		Command: "/bin/false",
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, NotNil)
	reloaderErr := err.(*ReloaderError)
	c.Assert(reloaderErr.Code, Equals, 1)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadWithArgs(c *C) {
	opts := CmdReloaderOpts{
		Command: "/bin/echo",
		Args:    []string{"hello", "world"},
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, IsNil)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadTimeout(c *C) {
	opts := CmdReloaderOpts{
		Command: "/bin/sleep",
		Args:    []string{"5"},
		Timeout: "1",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, NotNil)
	reloaderErr := err.(*ReloaderError)
	c.Assert(reloaderErr.Code, Equals, 1)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadNoCommand(c *C) {
	opts := CmdReloaderOpts{
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, NotNil)
	reloaderErr := err.(*ReloaderError)
	c.Assert(reloaderErr.Code, Equals, 2)
}

func (s *ReloaderTestSuite) TestCmdReloaderEnvironmentSubstitution(c *C) {
	os.Setenv("BUTLER_TEST_CMD", "/bin/true")
	defer os.Unsetenv("BUTLER_TEST_CMD")

	opts := CmdReloaderOpts{
		Command: "env:BUTLER_TEST_CMD",
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	cmdReloader := reloader.(CmdReloader)
	c.Assert(cmdReloader.Opts.Command, Equals, "/bin/true")
}

func (s *ReloaderTestSuite) TestCmdReloaderEnvironmentSubstitutionArgs(c *C) {
	os.Setenv("BUTLER_TEST_ARG", "world")
	defer os.Unsetenv("BUTLER_TEST_ARG")

	opts := CmdReloaderOpts{
		Command: "/bin/echo",
		Args:    []string{"hello", "env:BUTLER_TEST_ARG"},
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	cmdReloader := reloader.(CmdReloader)
	c.Assert(cmdReloader.Opts.Args, DeepEquals, []string{"hello", "world"})

	err = reloader.Reload()
	c.Assert(err, IsNil)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadNonexistentBinary(c *C) {
	opts := CmdReloaderOpts{
		Command: "/no/such/binary/xyz",
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, NotNil)
	reloaderErr := err.(*ReloaderError)
	c.Assert(reloaderErr.Code, Equals, 1)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadZeroTimeoutDoesNotHang(c *C) {
	// timeout of "0" means no context deadline is applied; the command
	// should still run to completion rather than being killed immediately.
	opts := CmdReloaderOpts{
		Command: "/bin/sleep",
		Args:    []string{"1"},
		Timeout: "0",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, IsNil)
}

func (s *ReloaderTestSuite) TestCmdReloaderReloadEmptyArgs(c *C) {
	opts := CmdReloaderOpts{
		Command: "/bin/true",
		Args:    nil,
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	err = reloader.Reload()
	c.Assert(err, IsNil)
}

func (s *ReloaderTestSuite) TestCmdReloaderMultipleReloadsWithCounter(c *C) {
	// Mirrors the real usage pattern in config/manager.go, where
	// SetCounter is called before each Reload().
	opts := CmdReloaderOpts{
		Command: "/bin/true",
		Timeout: "5",
	}

	jsonOpts, err := json.Marshal(opts)
	c.Assert(err, IsNil)

	reloader, err := NewCmdReloader("test-manager", "cmd", jsonOpts)
	c.Assert(err, IsNil)

	for i := 0; i < 3; i++ {
		reloader = reloader.SetCounter(i)
		err = reloader.Reload()
		c.Assert(err, IsNil)
	}
}
