module ints-bench

go 1.26.3

require (
	github.com/planetscale/vtprotobuf v0.6.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

tool (
	github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto
	google.golang.org/protobuf/cmd/protoc-gen-go
)
