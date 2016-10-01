FROM alpine:3.4
MAINTAINER Rohith <gambol99@gmail.com>

RUN apk update && \
    apk add ca-certificates bash

ADD bin/vault-sidekick /vault-sidekick

ENTRYPOINT [ "/vault-sidekick" ]
