package main

import (
	"fmt"
	"log"

	"buf.build/go/hyperpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func runHyperPB(sayRequestDesc protoreflect.MessageDescriptor, sentenceField protoreflect.FieldDescriptor, wireBytes []byte) {
	fmt.Println("\n--- Step 2: hyperpb (Table-Driven Bytecode VM) ---")

	// start: hyperpb
	// Compile the descriptor into hyperpb optimized message type
	// Note: You should compile descriptors once at startup or pool them, not per request.
	hyperMsgType := hyperpb.CompileMessageDescriptor(sayRequestDesc)

	// Instantiate hyperpb message
	hyperMsg := hyperpb.NewMessage(hyperMsgType)

	// Unmarshal wire bytes into it (hyperpb parses without Go reflection overhead)
	if err := proto.Unmarshal(wireBytes, hyperMsg); err != nil {
		log.Fatalf("hyperpb: failed to unmarshal message: %v", err)
	}

	// Get field dynamically from hyperpb message using standard protoreflect API
	hyperVal := hyperMsg.ProtoReflect().Get(sentenceField)
	fmt.Printf("Decoded message: %s\n", hyperVal.String())
	// end: hyperpb
}
