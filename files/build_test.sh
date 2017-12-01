#!/bin/bash

export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=/bin:/usr/bin:/usr/local/bin:/sbin:/usr/sbin:/usr/local/sbin:/usr/local/go/bin:$GOPATH/bin
export BUTLER_GO_PATH=/root/go/src/git.corp.adobe.com/TechOps-IAO/butler

if [ ! -d /tmp ]; then
    mkdir /tmp
fi

mkdir -p $BUTLER_GO_PATH
cd $BUTLER_GO_PATH
#rm -v /root/butler/vendor/vendor.json
#mv /root/butler/vendor/* /root/go/src
mv /root/butler/vendor .
mv /root/butler/*.go .

## stats dir
mkdir stats
mv /root/butler/stats/*.go stats

## config dir
mkdir config
mv /root/butler/config/*.go config

## config/methods
mkdir config/methods
mv /root/butler/config/methods/*.go config/methods

## config/reloaders
mkdir config/reloaders
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

go tool cover -func /tmp/coverage-main.out
echo
go tool cover -func /tmp/coverage-config.out
echo
go tool cover -func /tmp/coverage-config-methods.out
echo
go tool cover -func /tmp/coverage-config-reloaders.out
echo
go tool cover -func /tmp/coverage-stats.out
echo
