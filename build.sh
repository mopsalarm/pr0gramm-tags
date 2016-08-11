#!/bin/sh

set -e

glide install
go build -a

docker build -t mopsalarm/go-pr0gramm-tags .
docker push mopsalarm/go-pr0gramm-tags
