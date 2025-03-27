VERSION := $(shell git describe --tags --always --dirty="-dev" --match "v*.*.*" || echo "development" )
VERSION := $(VERSION:v%=%)

.PHONY: build
build:
	@CGO_ENABLED=0 go build \
			-ldflags "-X main.version=${VERSION}" \
			-o ./bin/bmonitor \
		github.com/flashbots/bmonitor/cmd

.PHONY: snapshot
snapshot:
	@goreleaser release --snapshot --clean

.PHONY: help
help:
	@go run github.com/flashbots/bmonitor/cmd serve --help

.PHONY: serve
serve:
	@go run github.com/flashbots/bmonitor/cmd -log-level debug serve \
		--monitor-builders builder-0=http://127.0.0.1:8645,builder-1=http://127.0.0.1:8646,builder-2=http://127.0.0.1:8647
