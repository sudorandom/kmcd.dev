#!/usr/bin/env bash
set -e

killall kmcd-render | true
go run ./cmd/kmcd-render &
until curl --output /dev/null --silent --head --fail http://127.0.0.1:7001; do
    printf '.'
    sleep 1
done
echo "build server started"
