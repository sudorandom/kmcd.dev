#!/usr/bin/env bash
# generate.sh — regenerate all isolated proto packages.
#
# Uses `buf generate` for each strategy directory. Plugins are invoked via
# `go tool <plugin>` so versions are pinned by the `tool` directive in go.mod
# and no system-wide binary installation is required.
#
# Outputs:
#   proto/vanilla/    — protoc-gen-go + vtprotobuf  (package vanillapb)
#   proto/vtproto/    — protoc-gen-go + vtprotobuf  (package vtprotopb)
#   proto/jsonplugin/ — protoc-gen-go + go-json     (package jsonpluginpb)

set -euo pipefail
cd "$(dirname "$0")"

# ── go tool wrappers ──────────────────────────────────────────────────────────
# buf discovers local plugins by name in PATH. We create thin wrapper scripts
# that delegate each invocation to `go tool <plugin>`, so buf never needs the
# binaries installed globally.
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

for plugin in protoc-gen-go protoc-gen-go-vtproto protoc-gen-go-json; do
  printf '#!/bin/sh\nexec go tool %s "$@"\n' "$plugin" > "$tmpdir/$plugin"
  chmod +x "$tmpdir/$plugin"
done

export PATH="$tmpdir:$PATH"

# ── generation ────────────────────────────────────────────────────────────────
echo "==> proto/vanilla/   (protoc-gen-go only)"
mise exec -- buf generate --template buf.gen.vanilla.yaml --path proto/vanilla/event.proto

echo "==> proto/vtproto/   (protoc-gen-go + vtprotobuf)"
mise exec -- buf generate --template buf.gen.vtproto.yaml --path proto/vtproto/event.proto

echo "==> proto/jsonplugin/ (protoc-gen-go + protoc-gen-go-json)"
mise exec -- buf generate --template buf.gen.jsonplugin.yaml --path proto/jsonplugin/event.proto

echo ""
echo "==> Done. Generated files:"
find proto/vanilla proto/vtproto proto/jsonplugin \
  \( -name "*.pb.go" -o -name "*.pb.json.go" \) | sort
