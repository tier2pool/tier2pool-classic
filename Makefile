VERSION=$(shell git rev-parse --short HEAD)

.PHONY: build
build:
	mkdir -p ./build

	GOOS=darwin GOARCH=amd64 \
	go build \
	-ldflags "-w -s -X github.com/tier2pool/tier2pool/internal/flag.Version=$(VERSION)" \
	-o ./build/tier2pool_darwin_amd64 ./cmd/main.go

	GOOS=linux GOARCH=amd64 \
	go build \
	-ldflags "-w -s -X github.com/tier2pool/tier2pool/internal/flag.Version=$(VERSION)" \
	-o ./build/tier2pool_linux_amd64 ./cmd/main.go
