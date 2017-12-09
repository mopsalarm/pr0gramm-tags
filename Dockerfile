FROM golang:1.9.2-stretch as builder

RUN go get github.com/Masterminds/glide/

COPY glide.* src/github.com/mopsalarm/go-pr0gramm-tags/
RUN cd src/github.com/mopsalarm/go-pr0gramm-tags/ && glide install --strip-vendor

COPY . src/github.com/mopsalarm/go-pr0gramm-tags/
RUN go build -v -ldflags="-s -w" -o /go-pr0gramm-tags github.com/mopsalarm/go-pr0gramm-tags


FROM debian:stretch-slim

ENV GIN_MODE release
COPY --from=builder /go-pr0gramm-tags /

EXPOSE 8080

ENTRYPOINT ["/go-pr0gramm-tags"]
