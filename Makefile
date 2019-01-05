.PHONY: dep build

VERSION := $(shell git describe --tags)
LDFLAGS := '-s -w -X main.version=$(VERSION)'

build: main.go dep
	go build -ldflags $(LDFLAGS) -o $@

dep: Gopkg.toml Gopkg.lock
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
	dep ensure
