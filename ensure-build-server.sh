#!/usr/bin/env bash
set -e

go install ./cmd/kmcd-render
killall kmcd-render | true
$(go env GOPATH)/bin/kmcd-render &
until curl --output /dev/null --silent --head --fail http://127.0.0.1:7001; do
    printf '.'
    sleep 1
done
echo "build server started"