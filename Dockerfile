FROM alpine:3.5
MAINTAINER Rohith <gambol99@gmail.com>

RUN apk update && \
    apk add ca-certificates bash

RUN adduser -D vault

ADD bin/vault-sidekick /vault-sidekick
RUN chmod 755 /vault-sidekick

USER vault

ENTRYPOINT [ "/vault-sidekick", "-logtostderr", "-v", "10"]
