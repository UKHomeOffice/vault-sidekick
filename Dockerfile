FROM docker.io/library/golang:1.23.2-alpine3.20 as builder

# Install 'make' and other necessary build tools
RUN apk add --no-cache make gcc musl-dev git

WORKDIR /go/src/github.com/ukhomeoffice/vault-sidekick

# Copy project files (excluding .git due to .dockerignore)
COPY . .

# Run the make build command
RUN make build

FROM alpine:3.20

RUN apk update && apk upgrade
RUN apk add --no-cache ca-certificates bash

RUN adduser -D vault

COPY --from=builder /go/src/github.com/ukhomeoffice/vault-sidekick /vault-sidekick

# Add vault-sidekick to the PATH
ENV PATH="/vault-sidekick/bin:${PATH}"

RUN chmod 755 /vault-sidekick

USER vault

ENTRYPOINT ["vault-sidekick"]
