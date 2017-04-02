#!/bin/sh

set -e

go fmt $(go list | grep -v /vendor/)

glide install
go build -a

docker build -t mopsalarm/go-pr0gramm-tags .
docker push mopsalarm/go-pr0gramm-tags
