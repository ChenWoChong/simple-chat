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

TARGETS = server
#DOCKER_TARGETS=$(foreach n,$(TARGETS),$(n)_docker)
#DOCKER_PUSH=$(foreach n,$(TARGETS),$(n)_push)

.PHONY: ${TARGETS}

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
	go get ./...

# ------------------------------------------------------------------------------------------------------------------------------

$(PROTO_TARGETS):     %.pb.go: %.proto
	protoc -I $(dir $@) $< --go_out=plugins=grpc:$(dir $@)

proto: $(PROTO_TARGETS)

# ------------------------------------------------------------------------------------------------------------------------------

server:
	@echo "创建 server-${VERSION}目录"
	@mkdir -p ${DESTDIR}/server-${VERSION}/conf
	@mkdir -p ${DESTDIR}/server-${VERSION}/bin

	@echo "拷贝配置文件"
	@cp -rf ${PROJECT_PATH}/config/conf.dev.yml ${DESTDIR}/server-${VERSION}/conf/conf.yml

	@echo "编译 server"
	@env GOOS=linux GOARCH=amd64 go build -ldflags ${LDFLAGS} -o ${DESTDIR}/server-${VERSION}/bin/server ./cmd/server

	@echo "打包文件 server-${VERSION}.tar.gz"
	@cd ${DESTDIR}; tar -czf server-${VERSION}.tar.gz server-${VERSION}

server_docker: server
	@cp -f ${PROJECT_PATH}/cmd/server/.dockerignore ${DESTDIR}/server-${VERSION}/
	@cp -f ${PROJECT_PATH}/cmd/server/docker-entrypoint.sh ${DESTDIR}/server-${VERSION}/
	@cp -f ${PROJECT_PATH}/cmd/server/Dockerfile ${DESTDIR}/server-${VERSION}/

	@chmod +x ${DESTDIR}/server-${VERSION}/docker-entrypoint.sh
	cd ${DESTDIR}/server-${VERSION}/;docker build -t server:${VERSION} ./

clean:
	rm -rf ${DESTDIR}
