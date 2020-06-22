GO111MODULE=on
BINARY_NAME=signalling
BINARY_NAME_CROSS_LINUX=signalling
RIMRAF=rm -rf
PACKAGE_NAME=go.sirus.dev/p2p-comm/signalling

# setup OS variables
ifeq ($(OS), Windows_NT)
	BINARY_NAME=signalling.exe
endif

.PHONY: all test docs

all:
	make init
	make build

init:
	make clean
	make install-dependency

build:
	go build -o $(BINARY_NAME) -v

run:
	go run $(PACKAGE_NAME)

help:~
	go run $(PACKAGE_NAME) help

lint:
	revive -config revive.toml -formatter stylish $(PACKAGE_NAME) pkg/...

test:
	ginkgo -cover ./...

proto:
	protoc --proto_path protos/ --go_out=plugins=grpc:protos protos/signalling.proto

build-cross-linux:
	make lint
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME_CROSS_LINUX) -v

install-dependency:
	GO111MODULE=off go get -u github.com/mgechev/revive
	GO111MODULE=off go get -u github.com/onsi/ginkgo/ginkgo
	go mod tidy

clean:
	$(RIMRAF) $(BINARY_NAME)
