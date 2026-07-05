#!/usr/bin/env bash
# generate.sh — regenerate all isolated proto packages.
#
# Uses `buf generate` for each strategy directory. Plugins are invoked via
# `go tool <plugin>` directly in the buf config files.
#
# Outputs:
#   proto/vanilla/    — protoc-gen-go (package vanillapb)
#   proto/vtproto/    — protoc-gen-go + vtprotobuf  (package vtprotopb)
#   proto/jsonplugin/ — protoc-gen-go + go-json     (package jsonpluginpb)
#   proto/protojsonx/ — protoc-gen-go + protojsonx  (package protojsonxpb)

set -euo pipefail
cd "$(dirname "$0")"

# ── generation ────────────────────────────────────────────────────────────────
echo "==> proto/vanilla/   (protoc-gen-go only)"
mise exec -- buf generate --template buf.gen.vanilla.yaml --path proto/vanilla/event.proto

echo "==> proto/vtproto/   (protoc-gen-go + vtprotobuf)"
mise exec -- buf generate --template buf.gen.vtproto.yaml --path proto/vtproto/event.proto

echo "==> proto/jsonplugin/ (protoc-gen-go + protoc-gen-go-json)"
mise exec -- buf generate --template buf.gen.jsonplugin.yaml --path proto/jsonplugin/event.proto

echo "==> proto/protojsonx/ (protoc-gen-go + protoc-gen-go-protojsonx)"
mise exec -- buf generate --template buf.gen.protojsonx.yaml --path proto/protojsonx/event.proto

echo ""
echo "==> Done. Generated files:"
find proto/vanilla proto/vtproto proto/jsonplugin proto/protojsonx \
  \( -name "*.pb.go" -o -name "*.pb.json.go" -o -name "*.protojsonx.pb.go" \) | sort
