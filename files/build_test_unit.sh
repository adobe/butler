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

if [ ! -d /tmp/coverage ]; then
    mkdir -p /tmp/coverage
fi

mkdir -p $BUTLER_GO_PATH
cd $BUTLER_GO_PATH
mv /root/butler/vendor .
mv /root/butler/.git .

## make butler directories
mkdir -p cmd/butler internal/monitor internal/stats internal/config internal/alog internal/environment internal/methods internal/reloaders

## move butler main
mv /root/butler/cmd/butler/*.go cmd/butler

## move stats files
mv /root/butler/internal/stats/*.go internal/stats

## move config files
mv /root/butler/internal/config/*.go internal/config

## move environment files
mv /root/butler/internal/environment/*.go internal/environment

## move alog files
mv /root/butler/internal/alog/*.go internal/alog

## move monitor files
mv /root/butler/internal/monitor/*.go internal/monitor

## move internal/methods files
mv /root/butler/internal/methods/*.go internal/methods

## move internal/reloaders files
mv /root/butler/internal/reloaders/*.go internal/reloaders

cd $BUTLER_GO_PATH/cmd/butler
go test -check.vv -coverprofile=/tmp/coverage-main.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/internal/config
go test -check.vv -coverprofile=/tmp/coverage-config.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/internal/methods
go test -check.vv -coverprofile=/tmp/coverage-config-methods.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/internal/reloaders
go test -check.vv -coverprofile=/tmp/coverage-config-reloaders.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/internal/monitor
go test -check.vv -coverprofile=/tmp/coverage-monitor.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

cd $BUTLER_GO_PATH/internal/stats
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

if [ -f /tmp/coverage-monitor.out ]; then
    go tool cover -func /tmp/coverage-monitor.out
    echo
fi

touch /tmp/coverage/coverage.txt
for i in /tmp/coverage-*; do
  cat $i >> /tmp/coverage/coverage.txt
done

if [ x${CODECOV_TOKEN} != "x" ]; then
    cd $BUTLER_GO_PATH
    echo "uplaoding coverage to codecov.io"
    bash <(curl -s https://codecov.io/bash) -s /tmp/coverage --retry 3
    exit $?
else
   echo "Could not find CODECOV_TOKEN environment. Not uploading to codecov.io."
   exit 0
fi
