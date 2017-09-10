#!/bin/sh

set -e

go fmt $(go list | grep -v /vendor/)

glide install --strip-vendor
go build -v -a -ldflags="-s -w"

docker build -t mopsalarm/go-pr0gramm-tags .
docker push mopsalarm/go-pr0gramm-tags
