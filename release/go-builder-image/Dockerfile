FROM golang:1.11

RUN mkdir -p /go/src/github.com/flant/
RUN git clone https://github.com/flant/werf /go/src/github.com/flant/werf
WORKDIR /go/src/github.com/flant/werf

RUN /go/src/github.com/flant/werf/release/go-get.sh
