FROM gliderlabs/alpine:latest
MAINTAINER Rohith <gambol99@gmail.com>

ADD build/vault-sidekick /vault-sidekick

ENTRYPOINT [ "/vault-sidekick" ]
