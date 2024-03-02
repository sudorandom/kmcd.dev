package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path"

	eliza "buf.build/gen/go/connectrpc/eliza/protocolbuffers/go/connectrpc/eliza/v1"
	"github.com/protocolbuffers/protoscope"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	pb "github.com/sudorandom/sudorandom.dev/inspecting-protobuf/gen/proto"
)

const testdataPrefix = "testdata"

var exampleProto = map[string]proto.Message{
	"eliza.SayRequest":       &eliza.SayRequest{Sentence: "Hello World!"},
	"eliza.IntroduceRequest": &eliza.IntroduceRequest{Name: "Hello World!"},
	"variety": &pb.Message{
		AString:         "Hello World!",
		AEnum:           pb.Message_C,
		AUint32:         175,
		ABytes:          RandomBytesWithMessage(128),
		AInt64:          921,
		ABool:           true,
		AFloat:          1.2345,
		ARepeatedUint64: []uint64{1, 2, 3, 4},
		ARepeatedInt32:  []int32{-1, -2, -3, -4},
		Nested: &pb.Nested{
			Bunny: "Fluffy",
			Cute:  true,
		},
		AMap: map[string]*pb.Nested{
			"key": {
				Bunny: "Fred",
				Cute:  false,
			},
		},
	},
	"numbers": &pb.Message{
		AEnum:           pb.Message_C,
		AUint32:         175,
		AInt64:          921,
		ABool:           true,
		AFloat:          1.2345,
		ARepeatedUint64: []uint64{1, 2, 3, 4},
		ARepeatedInt32:  []int32{-1, -2, -3, -4},
	},
	"submessages": &pb.Message{
		Nested: &pb.Nested{
			Bunny: "Harey",
			Cute:  true,
		},
		Anything: &anypb.Any{
			TypeUrl: "buf.build/connectrpc/eliza/connectrpc.eliza.v1.SayRequest",
			Value:   HexToBinary("0a0c48656c6c6f20576f726c6421"),
		},
		ManyThings: []*anypb.Any{},
		Submessage: &pb.Message{},
		Children: []*pb.Message{
			{
				AString: "alpha",
			},
			{
				AString: "beta",
			},
		},
	},
	"maps": &pb.Message{
		AMap: map[string]*pb.Nested{
			"key": {
				Bunny: "Corgi",
				Cute:  true,
			},
		},
		StringMap: map[string]string{
			"Knock, knock": "who's there?",
			"Java":         "Coffee, not code.",
		},
	},
	// ResultCount: 299792458,
	// "proto.true":   &pb.Message{TrueScotsman: true},
	// "proto.false":  &pb.Message{TrueScotsman: false},
	// "proto.uint32": &pb.Message{HeightInCm: 175},
	// "proto.float":  &pb.Message{Score: 12.6},
	"bytes":  &pb.Message{ABytes: RandomBytesWithMessage(128)},
	"bytes2": &pb.Message{ABytes: []byte("Hello World!")},
	// "proto.enum":   &pb.Message{Hilarity: proto.Message_PUNS},
}

func HexToBinary(s string) []byte {
	src := []byte(s)

	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(dst, src)
	if err != nil {
		log.Fatal(err)
	}

	return dst
}

func RandomBytesWithMessage(size int) []byte {
	buf := RandomBytes(size)
	buf[0] = 's'
	buf[1] = 'e'
	buf[2] = 'c'
	buf[3] = 'r'
	buf[4] = 'e'
	buf[5] = 't'
	return buf
}

func RandomBytes(size int) []byte {
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	if err != nil {
		log.Fatalf("error while generating random string: %s", err)
	}

	return buf
}

func main() {
	if err := os.RemoveAll(testdataPrefix); err != nil {
		log.Fatalf("err: %s", err)
	}
	if err := os.MkdirAll(testdataPrefix, 0700); err != nil {
		log.Fatalf("err: %s", err)
	}
	for name, msg := range exampleProto {
		msgBytes, err := proto.Marshal(msg)
		if err != nil {
			log.Fatalf("err: %s", err)
		}

		if err := os.WriteFile(path.Join(testdataPrefix, name+".pb"), msgBytes, 0644); err != nil {
			log.Fatalf("err: %s", err)
		}

		if err := os.WriteFile(path.Join(testdataPrefix, name+".pb.hex"), []byte(fmt.Sprintf("%x", msgBytes)), 0644); err != nil {
			log.Fatalf("err: %s", err)
		}

		psBytes := []byte(protoscope.Write(msgBytes, protoscope.WriterOptions{
			NoQuotedStrings:        false,
			AllFieldsAreMessages:   false,
			ExplicitWireTypes:      true,
			NoGroups:               false,
			ExplicitLengthPrefixes: false,

			Schema:          nil,
			PrintFieldNames: false,
			PrintEnumNames:  false,
		}))

		if err := os.WriteFile(path.Join(testdataPrefix, name+".txt"), psBytes, 0644); err != nil {
			log.Fatalf("err: %s", err)
		}
	}
}
