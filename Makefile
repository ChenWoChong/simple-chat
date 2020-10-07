#  Makefile.
#
# Create by: ChenGong At 2020-08-03
#
#
PROJECT_PATH=$(shell cd "$(dirname "$0" )" &&pwd)
PROJECT_NAME=$(shell basename "$(PWD)")
VERSION=$(shell git describe --tags | sed 's/\(.*\)-.*/\1/')
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
BUILD_HASH=$(shell git rev-parse HEAD)
LDFLAGS="-X main.buildstamp=${BUILD_DATE} -X main.githash=${BUILD_HASH} -X main.version=${VERSION} -s -w"

DESTDIR=${PROJECT_PATH}/build

TARGETS = server client
#DOCKER_TARGETS=$(foreach n,$(TARGETS),$(n)_docker)
#DOCKER_PUSH=$(foreach n,$(TARGETS),$(n)_push)

.PHONY: ${TARGETS} build

# ------------------------------------------------------------------------------------------------------------------------------

export PATH := $(shell go env GOPATH)/bin:dep/protoc/bin:$(PATH)

PROTO_TARGETS = ${PROJECT_PATH}/message/message.pb.go

# install protoc
dep/protoc/bin/protoc:
	mkdir -p dep/protoc
	curl -L -o dep/protoc/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v3.8.0/protoc-3.8.0-linux-x86_64.zip
	cd dep/protoc; unzip -o protoc.zip

dep-install: dep/protoc/bin/protoc
	go install github.com/golang/protobuf/protoc-gen-go
	go mod tidy

# ------------------------------------------------------------------------------------------------------------------------------

$(PROTO_TARGETS):     %.pb.go: %.proto
	@protoc -I $(dir $@) $< --go_out=plugins=grpc:$(dir $@)

proto: $(PROTO_TARGETS)

# ------------------------------------------------------------------------------------------------------------------------------

server:
	@go mod tidy
	@echo "创建 server-${VERSION}目录"
	@mkdir -p ${DESTDIR}/server-${VERSION}/conf
	@mkdir -p ${DESTDIR}/server-${VERSION}/bin

	@echo "拷贝配置文件"
	@cp -rf ${PROJECT_PATH}/config/conf-server.dev.yml ${DESTDIR}/server-${VERSION}/conf/conf.yml

	@echo "编译 linux server"
	@env GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${DESTDIR}/server-${VERSION}/bin/server ./cmd/server

	@echo "导出 本地系统server binary file"
	@env GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${DESTDIR}/server/bin/server ./cmd/server
	@cp -rf ${DESTDIR}/server-${VERSION}/conf/ ${DESTDIR}/server/conf/
	@#echo "打包文件 server-${VERSION}.tar.gz"
	@#cd ${DESTDIR}; tar -czf server-${VERSION}.tar.gz server-${VERSION}

server_docker: server
	@cp -f ${PROJECT_PATH}/cmd/server/.dockerignore ${DESTDIR}/server-${VERSION}/
	@cp -f ${PROJECT_PATH}/cmd/server/docker-entrypoint.sh ${DESTDIR}/server-${VERSION}/
	@cp -f ${PROJECT_PATH}/cmd/server/Dockerfile ${DESTDIR}/server-${VERSION}/

	@chmod +x ${DESTDIR}/server-${VERSION}/docker-entrypoint.sh
	cd ${DESTDIR}/server-${VERSION}/;docker build -t server:latest ./

# ------------------------------------------------------------------------------------------------------------------------------

client:
	@echo "创建 client-${VERSION}目录"
	@mkdir -p ${DESTDIR}/client-${VERSION}/conf
	@mkdir -p ${DESTDIR}/client-${VERSION}/bin

	@echo "拷贝配置文件"
	@cp -rf ${PROJECT_PATH}/config/conf-client.dev.yml ${DESTDIR}/client-${VERSION}/conf/conf.yml

	@echo "编译 linux client"
	@env GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${DESTDIR}/client-${VERSION}/bin/client ./cmd/client

	@echo "导出 本地系统client binary file"
	@env GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${DESTDIR}/client/bin/client ./cmd/client
	@cp -rf ${DESTDIR}/client-${VERSION}/conf/ ${DESTDIR}/client/conf/
	@#echo "打包文件 client-${VERSION}.tar.gz"
	@#cd ${DESTDIR}; tar -czf client-${VERSION}.tar.gz client-${VERSION}

client_docker: client
	@cp -f ${PROJECT_PATH}/cmd/client/.dockerignore ${DESTDIR}/client-${VERSION}/
	@cp -f ${PROJECT_PATH}/cmd/client/docker-entrypoint.sh ${DESTDIR}/client-${VERSION}/
	@cp -f ${PROJECT_PATH}/cmd/client/Dockerfile ${DESTDIR}/client-${VERSION}/

	@chmod +x ${DESTDIR}/client-${VERSION}/docker-entrypoint.sh
	@cd ${DESTDIR}/client-${VERSION}/;docker build -t client:latest ./

# ------------------------------------------------------------------------------------------------------------------------------

prepare:
	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml up -d rabbitmq mariadb

build: server_docker client

run_client:
	@${PROJECT_PATH}/build/client/bin/client -conf ${PROJECT_PATH}/build/client/conf/conf.yml -v 4 -logtostderr true

run_server:
	@sleep 30s
	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml up -d server

run:
	@make run_server
	@make run_client

update_server:
	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml stop server
	@docker rmi -f server:latest
	@make server_docker
	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml up -d server
	@docker logs -f server

update:
	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml stop
	@docker rmi -f server:latest
	@make server_docker
	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml up -d

	@docker stop client && docker rm client
	@docker rmi -f client:latest

	@make client_docker
	@docker run --net ccloud -it -d --restart=always \
	  --name client \
	  client:latest

test:
	@go test -v ./...

clean:
	@rm -rf ${DESTDIR}

	@docker-compose -f ${PROJECT_PATH}/config/server-compose.yml down

	@docker rmi -f server:latest

