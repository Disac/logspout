#!/bin/sh
set -e
WORKDIR=/go/src/github.com/acs/logspout
cd $WORKDIR
export GOPATH=$WORKDIR/vendor:$GOPATH
export VERSION=$(cat VERSION)
go build -ldflags "-X main.Version=$VERSION" -o ./bin/logspout
