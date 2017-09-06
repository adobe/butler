#!/bin/bash -x

export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=/bin:/usr/bin:/usr/local/bin:/sbin:/usr/sbin:/usr/local/sbin:/usr/local/go/bin:$GOPATH/bin
export BUTLER_GO_PATH=/root/go/src/git.corp.adobe.com/TechOps-IAO/butler

mkdir -p $BUTLER_GO_PATH
cd $BUTLER_GO_PATH
cp -Rp /root/butler/* .

go build

cp butler /root/butler
