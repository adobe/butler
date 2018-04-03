#!/bin/bash

export GOROOT=/usr/local/go
export GOPATH=/root/go
export PATH=/bin:/usr/bin:/usr/local/bin:/sbin:/usr/sbin:/usr/local/sbin:/usr/local/go/bin:$GOPATH/bin
export BUTLER_GO_PATH=/root/go/src/git.corp.adobe.com/TechOps-IAO/butler

if [ ! -d /tmp ]; then
    mkdir /tmp
fi

## Setup nginx stuff
mkdir -p /run/nginx
nginx
cp /certs/rootCA.* /usr/local/share/ca-certificates
mv /usr/local/share/ca-certificates/rootCA.ky /usr/local/share/ca-certificates/rootCA.key
update-ca-certificates
###
mkdir -p /opt/butler /opt/cache

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

## Let's build local go and perform some tests
cd $BUTLER_GO_PATH
go build -ldflags "-X main.version=$VERSION" -o /butler

BASE_SCRIPTS="/www/scripts/base.sh /www/scripts/s3.sh /www/scripts/azure.sh"
for script in /www/scripts/base.sh /www/scripts/s3.sh /www/scripts/azure.sh
do
    echo "[running script: $script]"
    $script
    res=$?
    echo "[done running script: $script]"
    if [ $res -ne 0 ]; then
       echo "bad error response for $script. exiting..."
       exit $res
    fi
    echo
done

#echo "[running user defined scripts]"
#for script in /www/scripts/user_*.sh
#do
#    $script
#    res=$?
#    if [ $res -ne 0 ]; then
#       echo "bad error response for $script. exiting..."
#       exit $res
#    fi
#done
#echo "[done running user defined scripts]"
exit 0
