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

## Let's build local go and perform some tests
cd $BUTLER_GO_PATH
go build -ldflags "-X main.version=$VERSION" -o /butler cmd/butler/main.go

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
