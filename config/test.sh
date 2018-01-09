#!/bin/bash

go test -check.vv -v -coverprofile=./coverage.out

if [ -f ./coverage.out ]; then
    go tool cover -func ./coverage.out
    rm -f ./coverage.out
fi
