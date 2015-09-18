
NAME=vault-sidekick
AUTHOR=gambol99
HARDWARE=$(shell uname -m)
VERSION=$(shell awk '/const Version/ { print $$4 }' version.go | sed 's/"//g')

.PHONY: test examples authors changelog build docker

default: build

build:
	mkdir -p build
	go build -o build/${NAME}

docker: build
	sudo docker build -t ${AUTHOR}/${NAME} .

clean:
	rm -rf ./build 2>/dev/null

authors:
	git log --format='%aN <%aE>' | sort -u > AUTHORS

test:
	go get
	go get github.com/stretchr/testify/assert
	go test -v

changelog: release
	git log $(shell git tag | tail -n1)..HEAD --no-merges --format=%B > changelog
