#!/bin/bash
# Copyright 2017 Adobe. All rights reserved.
# This file is licensed to you under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License. You may obtain a copy
# of the License at http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
# OF ANY KIND, either express or implied. See the License for the specific language
# governing permissions and limitations under the License.

export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=/bin:/usr/bin:/usr/local/bin:/sbin:/usr/sbin:/usr/local/sbin:/usr/local/go/bin:$GOPATH/bin
export BUTLER_GO_PATH=/root/go/src/github.com/adobe/butler

if [ ! -d /tmp ]; then
    mkdir /tmp
fi

mkdir -p $BUTLER_GO_PATH
cd $BUTLER_GO_PATH
mv /root/butler/vendor .
mv /root/butler/*.go .

## make butler directories
mkdir -p stats config alog environment config/methods config/reloaders

## move stats files
mv /root/butler/stats/*.go stats

## move config files
mv /root/butler/config/*.go config

## move environment files
mv /root/butler/environment/*.go environment

## move alog files
mv /root/butler/alog/*.go alog

## move config/methods files
mv /root/butler/config/methods/*.go config/methods

## move config/reloaders files
mv /root/butler/config/reloaders/*.go config/reloaders

go test -check.vv -coverprofile=/tmp/coverage-main.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/config
go test -check.vv -coverprofile=/tmp/coverage-config.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/config/methods
go test -check.vv -coverprofile=/tmp/coverage-config-methods.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/config/reloaders
go test -check.vv -coverprofile=/tmp/coverage-config-reloaders.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/stats
go test -check.vv -coverprofile=/tmp/coverage-stats.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

if [ -f /tmp/coverage-main.out ]; then
    go tool cover -func /tmp/coverage-main.out
    echo
fi

if [ -f /tmp/coverage-config.out ]; then
    go tool cover -func /tmp/coverage-config.out
    echo
fi

if [ -f /tmp/coverage-config-methods.out ]; then
    go tool cover -func /tmp/coverage-config-methods.out
    echo
fi

if [ -f /tmp/coverage-config-reloaders.out ]; then
    go tool cover -func /tmp/coverage-config-reloaders.out
    echo
fi

if [ -f /tmp/coverage-stats.out ]; then
    go tool cover -func /tmp/coverage-stats.out
    echo
fi
