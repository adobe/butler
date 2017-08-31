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
mv /root/butler/*.go .

go test -check.vv -coverprofile=/tmp/coverage.out
ret=$?

if [ $ret -ne 0 ]; then
    exit $ret
fi

go tool cover -func /tmp/coverage.out
