#!/bin/bash

go test -check.vv -v -coverprofile=./coverage.out
go tool cover -func ./coverage.out

if [ -f ./coverage.out ]; then
    rm -f ./coverage.out
fi
