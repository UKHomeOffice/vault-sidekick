FROM alpine:latest
MAINTAINER Rohith <gambol99@gmail.com>

ADD bin/vault-sidekick /vault-sidekick

ENTRYPOINT [ "/vault-sidekick" ]
