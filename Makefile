
NAME=vault-sidekick
AUTHOR=gambol99
HARDWARE=$(shell uname -m)
VERSION=$(shell awk '/Version =/ { print $$3 }' main.go | sed 's/"//g')

.PHONY: test authors changelog build docker static release

default: build

build:
	mkdir -p bin
	go build -o bin/${NAME}

docker: build
	sudo docker build -t ${AUTHOR}/${NAME}:${VERSION} .

static:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o bin/${NAME}

push: docker
	sudo docker tag -f ${AUTHOR}/${NAME}:${VERSION} docker.io/${AUTHOR}/${NAME}:${VERSION}
	sudo docker push docker.io/${AUTHOR}/${NAME}:${VERSION}

release: static
	mkdir -p release
	gzip -c bin/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}

clean:
	rm -rf ./bin 2>/dev/null
	rm -rf ./release 2>/dev/null

authors:
	git log --format='%aN <%aE>' | sort -u > AUTHORS

cover:
	go list ./... | xargs -n1 go test --cover

test: cover
	go get
	go get github.com/stretchr/testify/assert
	go test -v

changelog: release
	git log $(shell git tag | tail -n1)..HEAD --no-merges --format=%B > changelog
