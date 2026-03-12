FROM golang:1.26.1-alpine3.23 as builder

WORKDIR /go/src/github.com/ukhomeoffice/vault-sidekick

RUN apk add --no-cache make

COPY . .

RUN make build

FROM alpine:3.23

RUN apk update
RUN apk upgrade
RUN apk add ca-certificates bash
RUN adduser -D vault

COPY --from=builder /go/src/github.com/ukhomeoffice/vault-sidekick/bin/vault-sidekick /vault-sidekick

RUN chmod 755 /vault-sidekick

USER vault

ENTRYPOINT [ "/vault-sidekick" ]
