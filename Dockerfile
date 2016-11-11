FROM alpine:3.4
MAINTAINER Rohith <gambol99@gmail.com>

RUN apk update && \
    apk add ca-certificates bash

ADD bin/vault-sidekick /vault-sidekick

RUN adduser -D vault && \
    chown -R vault:vault /vault-sidekick && \
    mkdir /etc/secrets && \
    chown -R vault:vault /etc/secrets

ENTRYPOINT [ "/vault-sidekick" ]
USER vault
