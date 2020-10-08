FROM golang:1.15 as builder

ENV GOPATH /go
ENV GO11MODULES on
ENV CGOENABLED 0

COPY . /go/src/github.com/BrosSquad/currency-fetcher
WORKDIR /go/src/github.com/BrosSquad/currency-fetcher/

RUN go test -v ./...
RUN go build -v -o /build/currency-fetcher ./cli/currency-fetcher

FROM alpine:latest

COPY --from=builder /build/currency-fetcher /bin/currency-fetcher
