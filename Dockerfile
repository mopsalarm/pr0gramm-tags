FROM golang:1.10beta1-stretch as builder

RUN go get github.com/Masterminds/glide/

COPY glide.* src/github.com/mopsalarm/go-pr0gramm-tags/
RUN cd src/github.com/mopsalarm/go-pr0gramm-tags/ && glide install

COPY . src/github.com/mopsalarm/go-pr0gramm-tags/
RUN go build -v -o /go-pr0gramm-tags github.com/mopsalarm/go-pr0gramm-tags


FROM debian:stretch-slim

ENV GIN_MODE release
COPY --from=builder /go-pr0gramm-tags /

EXPOSE 8080

ENTRYPOINT ["/go-pr0gramm-tags"]
