default: build

build: go_build docker_build

DOCKER_IMAGE ?= stephenking/flannel-operator
DOCKER_TAG ?= latest
BINARY ?= operator

go_build:
	GOOS=linux go build -o $(BINARY) cmd/operator/main.go

docker_build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

clean:
	rm $(BINARY)

test:
	go test $(shell go list ./...)
