FROM golang:1.21 as builder

WORKDIR /go/src/github.com/ukhomeoffice/vault-sidekick

COPY . .

RUN make build

FROM alpine:3.19.1

RUN apk update
RUN apk add ca-certificates bash
RUN adduser -D vault

COPY --from=builder /go/src/github.com/ukhomeoffice/vault-sidekick /vault-sidekick

RUN chmod 755 /vault-sidekick

USER vault

ENTRYPOINT [ "/vault-sidekick" ]
