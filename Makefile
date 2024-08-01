.PHONY: dep build

VERSION := $(shell git describe --tags)
LDFLAGS := '-s -w -X main.version=$(VERSION)'

build: main.go
	go build -ldflags $(LDFLAGS) -o $@