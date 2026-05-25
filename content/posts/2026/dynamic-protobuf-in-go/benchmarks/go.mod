module json-vs-proto

go 1.26.3

require (
	buf.build/go/hyperpb v0.1.3
	github.com/go-json-experiment/json v0.0.0-20260520185125-572e7c383686
	github.com/planetscale/vtprotobuf v0.6.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/bufbuild/protoplugin v0.0.0-20250218205857-750e09ce93e1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/mfridman/protoc-gen-go-json v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/timandy/routine v1.1.5 // indirect
	golang.org/x/sync v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

tool (
	github.com/mfridman/protoc-gen-go-json
	github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto
	google.golang.org/protobuf/cmd/protoc-gen-go
)
