#!/bin/bash -x

export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=/bin:/usr/bin:/usr/local/bin:/sbin:/usr/sbin:/usr/local/sbin:/usr/local/go/bin:$GOPATH/bin

mkdir -p $GOPATH/src/git.corp.adobe.com/TechOps-IAO
cd $GOPATH/src/git.corp.adobe.com/TechOps-IAO

git clone git@git.corp.adobe.com:TechOps-IAO/butler.git

cd butler
go build

