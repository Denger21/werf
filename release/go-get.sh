#!/bin/bash

set -e

for os in linux darwin windows ; do
  for arch in amd64 ; do
    export GOOS=$os
    export GOARCH=$arch
    source $GOPATH/src/github.com/flant/werf/go-get.sh
  done
done
