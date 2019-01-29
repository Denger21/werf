#!/bin/bash

set -e

IMAGE_NAME=flant/werf-builder:latest

docker build -f release/go-builder-image/Dockerfile -t $IMAGE_NAME .
docker login -u $DOCKER_HUB_LOGIN -p $DOCKER_HUB_PASSWORD || true
docker push $IMAGE_NAME
