package main

import (
	"fmt"
	"log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// start: dynamicpb
func runDynamicPB(sayRequestDesc protoreflect.MessageDescriptor, sentenceField protoreflect.FieldDescriptor) []byte {
	fmt.Println("--- Step 1: dynamicpb (Standard Go Reflection) ---")

	// Create request message dynamically
	dynMsg := dynamicpb.NewMessage(sayRequestDesc)

	// Set field dynamically using the reflection interface
	dynMsg.ProtoReflect().Set(sentenceField, protoreflect.ValueOfString("Hello Eliza, how are you?"))

	// Marshal the message to binary wire format
	wireBytes, err := proto.Marshal(dynMsg)
	if err != nil {
		log.Fatalf("dynamicpb: failed to marshal message: %v", err)
	}
	fmt.Printf("Serialized bytes: %x\n", wireBytes)

	// Unmarshal wire format back into a new dynamicpb message
	dynMsg2 := dynamicpb.NewMessage(sayRequestDesc)
	if err := proto.Unmarshal(wireBytes, dynMsg2); err != nil {
		log.Fatalf("dynamicpb: failed to unmarshal message: %v", err)
	}

	// Get field dynamically
	val := dynMsg2.ProtoReflect().Get(sentenceField)
	fmt.Printf("Decoded message: %s\n", val.String())

	return wireBytes
}
