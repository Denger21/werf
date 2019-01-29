FROM golang:1.11

ADD cmd /go/src/github.com/flant/werf/cmd
ADD pkg /go/src/github.com/flant/werf/pkg
ADD go-get.sh /go/src/github.com/flant/werf/go-get.sh
ADD release/go-get.sh /go/src/github.com/flant/werf/release/go-get.sh

WORKDIR /go/src/github.com/flant/werf

RUN /go/src/github.com/flant/werf/release/go-get.sh
