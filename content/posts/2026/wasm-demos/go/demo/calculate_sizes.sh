#!/bin/bash

# Navigate to the demo directory
cd "$(dirname "$0")"

# Ensure dependencies are tidy
go mod tidy > /dev/null 2>&1

# Build Go WASM
export GOOS=js GOARCH=wasm
go build -o demo.go.wasm main.go
gzip -c demo.go.wasm > demo.go.wasm.gz

# Build TinyGo WASM
tinygo build -o demo.tiny.wasm -target wasm main.go
gzip -c demo.tiny.wasm > demo.tiny.wasm.gz

# Function to format bytes to human readable (MB/KB)
format_size() {
    local bytes=$1
    if [ $bytes -ge 1048576 ]; then
        echo "$(echo "scale=1; $bytes / 1048576" | bc) MB"
    else
        echo "$(echo "scale=0; $bytes / 1024" | bc) KB"
    fi
}

# Get sizes in bytes
size_go_raw=$(stat -f%z demo.go.wasm)
size_go_gz=$(stat -f%z demo.go.wasm.gz)
size_tiny_raw=$(stat -f%z demo.tiny.wasm)
size_tiny_gz=$(stat -f%z demo.tiny.wasm.gz)

# Output Markdown table
echo "| Compiler | Raw Size | Gzipped Size |"
echo "| :--- | :--- | :--- |"
echo "| **Go** | $(format_size $size_go_raw) | $(format_size $size_go_gz) |"
echo "| **TinyGo** | $(format_size $size_tiny_raw) | $(format_size $size_tiny_gz) |"

# Cleanup
rm demo.go.wasm demo.go.wasm.gz demo.tiny.wasm demo.tiny.wasm.gz
