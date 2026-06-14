package main

import (
	"log"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	sayRequestDesc, sentenceField := loadDescriptors()

	// Run the examples
	wireBytes := runDynamicPB(sayRequestDesc, sentenceField)
	runHyperPB(sayRequestDesc, sentenceField, wireBytes)
	runHyperPBShared(sayRequestDesc, sentenceField, wireBytes)
}

// start: load
func loadDescriptors() (protoreflect.MessageDescriptor, protoreflect.FieldDescriptor) {
	// start: register
	// Read compiled schema descriptors
	descriptorBytes, err := os.ReadFile("eliza.binpb")
	if err != nil {
		log.Fatalf("failed to read descriptor file (did you run 'buf build -o eliza.binpb'?): %v", err)
	}

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(descriptorBytes, &fds); err != nil {
		log.Fatalf("failed to unmarshal file descriptor set: %v", err)
	}

	// Register files
	registry, err := protodesc.NewFiles(&fds)
	if err != nil {
		log.Fatalf("failed to create protodesc registry: %v", err)
	}
	// end: register

	// start: lookup
	// Retrieve message descriptor for Eliza's SayRequest
	sayRequestName := protoreflect.FullName("connectrpc.eliza.v1.SayRequest")
	desc, err := registry.FindDescriptorByName(sayRequestName)
	if err != nil {
		log.Fatalf("failed to find descriptor for %s: %v", sayRequestName, err)
	}

	sayRequestDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		log.Fatalf("descriptor for %s is not a message descriptor", sayRequestName)
	}

	sentenceField := sayRequestDesc.Fields().ByName("sentence")
	if sentenceField == nil {
		log.Fatalf("failed to find 'sentence' field in %s", sayRequestName)
	}
	// end: lookup

	return sayRequestDesc, sentenceField
}
