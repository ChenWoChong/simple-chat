#!/usr/bin/env bash

docker stop server && docker rm server
docker rmi -f server:latest

make server_docker
docker run --net ccloud -it -d --restart=always \
  --name server \
  -p 12345:12345 \
  server:latest
#  -v /etc/localtime:/etc/localtime:ro \
#  -v "${HOME}"/data/k8s/config/server/conf:/conf \

docker stop client && docker rm client
docker rmi -f client:latest

make client_docker
docker run --net ccloud -it -d --restart=always \
  --name client \
  client:latest
#  -p 12345:12345 \

