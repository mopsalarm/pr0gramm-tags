FROM centos:7.2.1511

ENV GIN_MODE release
COPY go-pr0gramm-tags /

EXPOSE 8080

ENTRYPOINT ["/go-pr0gramm-tags"]
