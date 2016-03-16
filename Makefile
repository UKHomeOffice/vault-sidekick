
NAME=vault-sidekick
AUTHOR=gambol99
HARDWARE=$(shell uname -m)
VERSION=$(shell awk '/Version =/ { print $$3 }' main.go | sed 's/"//g')
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

.PHONY: test authors changelog build docker static release

default: build

build:
	@echo "--> Compiling the project"
	mkdir -p bin
	go build -o bin/${NAME}

static:
	@echo "--> Compiling the static binary"
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o bin/${NAME}

docker: static
	@echo "--> Building the docker image"
	sudo docker build -t ${AUTHOR}/${NAME}:${VERSION} .

push: docker
	@echo "--> Pushing the image to docker.io"
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
	@echo "--> Updating the AUTHORS"
	git log --format='%aN <%aE>' | sort -u > AUTHORS

deps:
	@echo "--> Installing build dependencies"
	go get -d -v ./...
	go get github.com/stretchr/testify/assert

vet:
	@echo "--> Running go tool vet $(VETARGS) ."
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@go tool vet $(VETARGS) .

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

cover:
	@echo "--> Running go cover"
	go list ./... | xargs -n1 go test --cover

test: deps
	@echo "--> Running the tests"
	go test -v
	@$(MAKE) vet
	@$(MAKE) cover

changelog: release
	git log $(shell git tag | tail -n1)..HEAD --no-merges --format=%B > changelog
