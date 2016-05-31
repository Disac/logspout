#!/bin/bash
GIT_SHA=`git rev-parse --short HEAD || echo "GitNotFound"`

docker run --rm -v $PWD:/go/src/github.com/acs/logspout golang:1.5.3-alpine /go/src/github.com/acs/logspout/builder.sh
