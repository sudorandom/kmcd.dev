# Dynamic Protobuf Reflection Example (Go)

This directory contains a complete, working example demonstrating how to load dynamic Protobuf descriptors at runtime and parse messages using both standard `dynamicpb` and Buf's high-performance `hyperpb`.

It leverages the classic [Eliza service](https://buf.build/connectrpc/eliza) schema for demonstration.

## Prerequisites

- [Go](https://go.dev) (v1.25+)
- [Buf CLI](https://buf.build) (or use `mise` to load them automatically)

## Running the Example

1. **Compile the Protobuf definitions to a descriptor set binary:**
   ```bash
   buf build -o eliza.binpb
   ```

2. **Run the Go program:**
   ```bash
   go run main.go
   ```

## Expected Output

```
--- Step 1: dynamicpb (Standard Go Reflection) ---
Serialized bytes: 0a1948656c6c6f20456c697a612c20686f772061726520796f753f
Decoded message: Hello Eliza, how are you?

--- Step 2: hyperpb (Table-Driven Bytecode VM) ---
Decoded message: Hello Eliza, how are you?

--- Step 3: hyperpb + Shared (Memory Reuse Arena) ---
Decoded message: Hello Eliza, how are you?
Memory arena recycled.
```
