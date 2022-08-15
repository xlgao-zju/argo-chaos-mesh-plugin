GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

VERSION ?= "v0.0.1"
BIN_NAME ?= argo-chaos-mesh-plugin

.PHONY: bin
bin:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(BIN_NAME) -v ./server

.PHONY: image
image:
	docker build -f Dockerfile -t ccr.ccs.tencentyun.com/xlgao/argo-chaos-mesh-plugin:$(VERSION) .

clean:
	rm -rf bin
	go clean -i .

.PHONY: vendor
vendor:
	go mod tidy -compat=1.17
	go mod vendor
