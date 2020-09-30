#!/usr/bin/env bash

docker network create --driver bridge ccloud

# server
make server_docker
docker run --net ccloud -it -d --restart=always \
  --name server \
  -p 12345:12345 \
  server:latest
#  -v /etc/localtime:/etc/localtime:ro \
#  -v "${HOME}"/data/k8s/config/server/conf:/conf \

# client
make client_docker
docker run --net ccloud -it -d --restart=always \
  --name client \
  -p 12345:12345 \
  client:latest
#  -v /etc/localtime:/etc/localtime:ro \
#  -v "${HOME}"/data/k8s/config/client/conf:/conf \

