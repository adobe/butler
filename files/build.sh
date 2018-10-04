#!/bin/bash -x
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

mkdir -p $BUTLER_GO_PATH
cd $BUTLER_GO_PATH
cp -Rp /root/butler/* .

go build -ldflags "-X main.version=$VERSION" -o butler cmd/butler/main.go

cp butler /root/butler
