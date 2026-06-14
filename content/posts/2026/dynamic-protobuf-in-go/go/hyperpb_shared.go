package main

import (
	"fmt"
	"log"

	"buf.build/go/hyperpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// start: hyperpb_shared
func runHyperPBShared(sayRequestDesc protoreflect.MessageDescriptor, sentenceField protoreflect.FieldDescriptor, wireBytes []byte) {
	fmt.Println("\n--- Step 3: hyperpb + Shared (Memory Reuse Arena) ---")

	// Instantiate shared arena once (usually per worker goroutine/worker thread)
	shared := new(hyperpb.Shared)

	// Compile the descriptor into hyperpb optimized message type
	hyperMsgType := hyperpb.CompileMessageDescriptor(sayRequestDesc)

	// Allocate message within the shared memory arena
	sharedMsg := shared.NewMessage(hyperMsgType)

	// Unmarshal wire bytes into the arena-allocated message
	if err := proto.Unmarshal(wireBytes, sharedMsg); err != nil {
		log.Fatalf("hyperpb shared: failed to unmarshal message: %v", err)
	}

	// Read field dynamically using standard protoreflect API
	sharedVal := sharedMsg.ProtoReflect().Get(sentenceField)
	fmt.Printf("Decoded message: %s\n", sharedVal.String())

	// Recycle the memory arena back to the pool (must be done synchronously before next request)
	shared.Free()
	fmt.Println("Memory arena recycled.")
}
