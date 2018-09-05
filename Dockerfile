FROM golang:1.11-alpine3.8 as builder

RUN apk add --no-cache git g++

ENV GO111MODULE=on

ENV PACKAGE github.com/mopsalarm/go-pr0gramm-tags
WORKDIR $GOPATH/src/$PACKAGE/

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -ldflags="-s -w" -o /go-pr0gramm-tags .


FROM alpine:3.8
RUN apk add --no-cache ca-certificates libstdc++
EXPOSE 8080

COPY --from=builder /go-pr0gramm-tags /

ENTRYPOINT ["/go-pr0gramm-tags"]
