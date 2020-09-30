#!/usr/bin/env bash

docker network create --driver bridge ccloud

make server_docker
docker run --net ccloud -it -d --restart=always \
  --name server \
  -p 12345:12345 \
  server:v0.0.0
#  -v /etc/localtime:/etc/localtime:ro \
#  -v "${HOME}"/data/k8s/config/server/conf:/conf \

