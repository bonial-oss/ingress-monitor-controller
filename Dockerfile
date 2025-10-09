FROM golang:1.25.1-alpine3.21 as builder

WORKDIR /src

RUN apk --update --no-cache add git make

ENV CGO_ENABLED=0

COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile

RUN go mod download

COPY *.go ./
COPY pkg/ pkg/

RUN make build

FROM alpine:3.21.3

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

COPY --from=builder /src/ingress-monitor-controller /ingress-monitor-controller

ENTRYPOINT ["/ingress-monitor-controller"]
