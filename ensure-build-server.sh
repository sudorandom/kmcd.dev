#!/usr/bin/env bash
set -e

go install ./cmd/kmcd-render
killall kmcd-render | true
$(go env GOPATH)/bin/kmcd-render &
