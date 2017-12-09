#!/bin/sh
set -e

docker build -t mopsalarm/go-pr0gramm-tags .
docker push mopsalarm/go-pr0gramm-tags
