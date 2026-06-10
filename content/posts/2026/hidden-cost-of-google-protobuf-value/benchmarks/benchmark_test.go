package benchmark

import (
	"encoding/json"
	"fmt"
	"testing"

	jsonpluginpb "json-vs-proto/proto/jsonplugin"
	vanillapb "json-vs-proto/proto/vanilla"
	vtprotopb "json-vs-proto/proto/vtproto"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"buf.build/go/hyperpb"

	jsonv2 "github.com/go-json-experiment/json"
)

var (
	hyperSmallType  *hyperpb.MessageType
	hyperMediumType *hyperpb.MessageType
	hyperLargeType  *hyperpb.MessageType

	dynamicSmallDesc  protoreflect.MessageDescriptor
	dynamicMediumDesc protoreflect.MessageDescriptor
	dynamicLargeDesc  protoreflect.MessageDescriptor
)

func init() {
	smallDesc := (*vanillapb.SmallObject)(nil).ProtoReflect().Descriptor()
	mediumDesc := (*vanillapb.MediumEvent)(nil).ProtoReflect().Descriptor()
	largeDesc := (*vanillapb.LargeEventPayload)(nil).ProtoReflect().Descriptor()

	hyperSmallType = hyperpb.CompileMessageDescriptor(smallDesc)
	hyperMediumType = hyperpb.CompileMessageDescriptor(mediumDesc)
	hyperLargeType = hyperpb.CompileMessageDescriptor(largeDesc)

	dynamicSmallDesc = smallDesc
	dynamicMediumDesc = mediumDesc
	dynamicLargeDesc = largeDesc
}

// Define concrete structs for JSON parsing.
type SmallObject struct {
	ID     string  `json:"id"`
	Active bool    `json:"active"`
	Age    int     `json:"age"`
	Score  float64 `json:"score"`
}

type Actor struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
}

type Metadata struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Attempts  int    `json:"attempts"`
}

type MediumEvent struct {
	ID        string   `json:"id"`
	Timestamp int64    `json:"timestamp"`
	EventType string   `json:"event_type"`
	Actor     Actor    `json:"actor"`
	Tags      []string `json:"tags"`
	Metadata  Metadata `json:"metadata"`
}

// Generate test inputs
func getSmallMap() map[string]any {
	return map[string]any{
		"id":     "usr_123456",
		"active": true,
		"age":    30,
		"score":  95.5,
	}
}

func getSmallStruct() SmallObject {
	return SmallObject{
		ID:     "usr_123456",
		Active: true,
		Age:    30,
		Score:  95.5,
	}
}

func getMediumMap() map[string]any {
	return map[string]any{
		"id":         "evt_987654",
		"timestamp":  int64(1620000000),
		"event_type": "user_signup",
		"actor": map[string]any{
			"id":       "usr_123456",
			"email":    "user@example.com",
			"verified": true,
		},
		"tags": []any{"marketing", "beta_user", "us_east"},
		"metadata": map[string]any{
			"ip":         "192.168.1.1",
			"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			"attempts":   int64(3),
		},
	}
}

func getMediumStruct() MediumEvent {
	return MediumEvent{
		ID:        "evt_987654",
		Timestamp: 1620000000,
		EventType: "user_signup",
		Actor: Actor{
			ID:       "usr_123456",
			Email:    "user@example.com",
			Verified: true,
		},
		Tags: []string{"marketing", "beta_user", "us_east"},
		Metadata: Metadata{
			IP:        "192.168.1.1",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			Attempts:  3,
		},
	}
}

func getLargeMap() []any {
	med := getMediumMap()
	list := make([]any, 100)
	for i := 0; i < 100; i++ {
		list[i] = med
	}
	return list
}

func getLargeStruct() []MediumEvent {
	med := getMediumStruct()
	list := make([]MediumEvent, 100)
	for i := 0; i < 100; i++ {
		list[i] = med
	}
	return list
}

// --- Vanilla Message Helpers ---

func getSmallVanillaMsg() *vanillapb.SmallObject {
	return &vanillapb.SmallObject{
		Id:     "usr_123456",
		Active: true,
		Age:    30,
		Score:  95.5,
	}
}

func getMediumVanillaMsg() *vanillapb.MediumEvent {
	return &vanillapb.MediumEvent{
		Id:        "evt_987654",
		Timestamp: 1620000000,
		EventType: "user_signup",
		Actor: &vanillapb.Actor{
			Id:       "usr_123456",
			Email:    "user@example.com",
			Verified: true,
		},
		Tags: []string{"marketing", "beta_user", "us_east"},
		Metadata: &vanillapb.Metadata{
			Ip:        "192.168.1.1",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			Attempts:  3,
		},
	}
}

func getLargeVanillaMsg() []*vanillapb.MediumEvent {
	med := getMediumVanillaMsg()
	list := make([]*vanillapb.MediumEvent, 100)
	for i := 0; i < 100; i++ {
		list[i] = med
	}
	return list
}

func getLargeVanillaMsgPayload() *vanillapb.LargeEventPayload {
	return &vanillapb.LargeEventPayload{
		Events: getLargeVanillaMsg(),
	}
}

// --- VTProto Message Helpers ---

func getSmallVTProtoMsg() *vtprotopb.SmallObject {
	return &vtprotopb.SmallObject{
		Id:     "usr_123456",
		Active: true,
		Age:    30,
		Score:  95.5,
	}
}

func getMediumVTProtoMsg() *vtprotopb.MediumEvent {
	return &vtprotopb.MediumEvent{
		Id:        "evt_987654",
		Timestamp: 1620000000,
		EventType: "user_signup",
		Actor: &vtprotopb.Actor{
			Id:       "usr_123456",
			Email:    "user@example.com",
			Verified: true,
		},
		Tags: []string{"marketing", "beta_user", "us_east"},
		Metadata: &vtprotopb.Metadata{
			Ip:        "192.168.1.1",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			Attempts:  3,
		},
	}
}

func getLargeVTProtoMsg() []*vtprotopb.MediumEvent {
	med := getMediumVTProtoMsg()
	list := make([]*vtprotopb.MediumEvent, 100)
	for i := 0; i < 100; i++ {
		list[i] = med
	}
	return list
}

func getLargeVTProtoMsgPayload() *vtprotopb.LargeEventPayload {
	return &vtprotopb.LargeEventPayload{
		Events: getLargeVTProtoMsg(),
	}
}

// --- JSONPlugin Message Helpers ---

func getSmallJSONPluginMsg() *jsonpluginpb.SmallObject {
	return &jsonpluginpb.SmallObject{
		Id:     "usr_123456",
		Active: true,
		Age:    30,
		Score:  95.5,
	}
}

func getMediumJSONPluginMsg() *jsonpluginpb.MediumEvent {
	return &jsonpluginpb.MediumEvent{
		Id:        "evt_987654",
		Timestamp: 1620000000,
		EventType: "user_signup",
		Actor: &jsonpluginpb.Actor{
			Id:       "usr_123456",
			Email:    "user@example.com",
			Verified: true,
		},
		Tags: []string{"marketing", "beta_user", "us_east"},
		Metadata: &jsonpluginpb.Metadata{
			Ip:        "192.168.1.1",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			Attempts:  3,
		},
	}
}

func getLargeJSONPluginMsg() []*jsonpluginpb.MediumEvent {
	med := getMediumJSONPluginMsg()
	list := make([]*jsonpluginpb.MediumEvent, 100)
	for i := 0; i < 100; i++ {
		list[i] = med
	}
	return list
}

func getLargeJSONPluginMsgPayload() *jsonpluginpb.LargeEventPayload {
	return &jsonpluginpb.LargeEventPayload{
		Events: getLargeJSONPluginMsg(),
	}
}

// Print byte sizes and check results.
func TestSizes(t *testing.T) {
	fmt.Println("=== SERIALIZED DATA SIZES ===")

	// 1. Small
	smallMap := getSmallMap()
	smallJSON, _ := json.Marshal(smallMap)
	smallPb, _ := structpb.NewValue(smallMap)
	smallPbBytes, _ := proto.Marshal(smallPb)

	smallVanilla := getSmallVanillaMsg()
	smallProtoBytes, _ := proto.Marshal(smallVanilla)
	smallAny, _ := anypb.New(smallVanilla)
	smallAnyBytes, _ := proto.Marshal(smallAny)
	smallPjStatic, _ := protojson.Marshal(smallVanilla)
	smallPjValue, _ := protojson.Marshal(smallPb)
	smallPjAny, _ := protojson.Marshal(smallAny)

	// VTProto sizes
	smallVT := getSmallVTProtoMsg()
	smallVTBytes, _ := smallVT.MarshalVT()

	// JSONPlugin sizes
	smallJP := getSmallJSONPluginMsg()
	smallJPBytes, _ := json.Marshal(smallJP)

	smallProtoJSONEnv := &vanillapb.JSONEnvelope{JsonData: string(smallJSON)}
	smallProtoJSONBytes, _ := proto.Marshal(smallProtoJSONEnv)

	fmt.Printf("Small payload:\n")
	fmt.Printf("  Concrete (JSON) size: %d bytes\n", len(smallJSON))
	fmt.Printf("  Concrete (proto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallProtoBytes), float64(len(smallProtoBytes))/float64(len(smallJSON))*100)
	fmt.Printf("  google.protobuf.Value (proto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallPbBytes), float64(len(smallPbBytes))/float64(len(smallJSON))*100)
	fmt.Printf("  google.protobuf.Any (proto) size: %d bytes (%.1f%% of Concrete (JSON)) [Includes type_url overhead]\n", len(smallAnyBytes), float64(len(smallAnyBytes))/float64(len(smallJSON))*100)
	fmt.Printf("  Protobuf + JSON size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallProtoJSONBytes), float64(len(smallProtoJSONBytes))/float64(len(smallJSON))*100)
	fmt.Printf("  Concrete (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallPjStatic), float64(len(smallPjStatic))/float64(len(smallJSON))*100)
	fmt.Printf("  google.protobuf.Value (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallPjValue), float64(len(smallPjValue))/float64(len(smallJSON))*100)
	fmt.Printf("  google.protobuf.Any (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallPjAny), float64(len(smallPjAny))/float64(len(smallJSON))*100)
	fmt.Printf("  VTProto size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallVTBytes), float64(len(smallVTBytes))/float64(len(smallJSON))*100)
	fmt.Printf("  JSONPlugin size: %d bytes (%.1f%% of Concrete (JSON))\n", len(smallJPBytes), float64(len(smallJPBytes))/float64(len(smallJSON))*100)

	// 2. Medium
	mediumMap := getMediumMap()
	mediumJSON, _ := json.Marshal(mediumMap)
	mediumPb, _ := structpb.NewValue(mediumMap)
	mediumPbBytes, _ := proto.Marshal(mediumPb)

	mediumVanilla := getMediumVanillaMsg()
	mediumProtoBytes, _ := proto.Marshal(mediumVanilla)
	mediumAny, _ := anypb.New(mediumVanilla)
	mediumAnyBytes, _ := proto.Marshal(mediumAny)
	mediumPjStatic, _ := protojson.Marshal(mediumVanilla)
	mediumPjValue, _ := protojson.Marshal(mediumPb)
	mediumPjAny, _ := protojson.Marshal(mediumAny)

	mediumVT := getMediumVTProtoMsg()
	mediumVTBytes, _ := mediumVT.MarshalVT()

	mediumJP := getMediumJSONPluginMsg()
	mediumJPBytes, _ := json.Marshal(mediumJP)

	mediumProtoJSONEnv := &vanillapb.JSONEnvelope{JsonData: string(mediumJSON)}
	mediumProtoJSONBytes, _ := proto.Marshal(mediumProtoJSONEnv)

	fmt.Printf("Medium payload:\n")
	fmt.Printf("  Concrete (JSON) size: %d bytes\n", len(mediumJSON))
	fmt.Printf("  Concrete (proto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumProtoBytes), float64(len(mediumProtoBytes))/float64(len(mediumJSON))*100)
	fmt.Printf("  google.protobuf.Value (proto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumPbBytes), float64(len(mediumPbBytes))/float64(len(mediumJSON))*100)
	fmt.Printf("  google.protobuf.Any (proto) size: %d bytes (%.1f%% of Concrete (JSON)) [Includes type_url overhead]\n", len(mediumAnyBytes), float64(len(mediumAnyBytes))/float64(len(mediumJSON))*100)
	fmt.Printf("  Protobuf + JSON size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumProtoJSONBytes), float64(len(mediumProtoJSONBytes))/float64(len(mediumJSON))*100)
	fmt.Printf("  Concrete (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumPjStatic), float64(len(mediumPjStatic))/float64(len(mediumJSON))*100)
	fmt.Printf("  google.protobuf.Value (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumPjValue), float64(len(mediumPjValue))/float64(len(mediumJSON))*100)
	fmt.Printf("  google.protobuf.Any (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumPjAny), float64(len(mediumPjAny))/float64(len(mediumJSON))*100)
	fmt.Printf("  VTProto size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumVTBytes), float64(len(mediumVTBytes))/float64(len(mediumJSON))*100)
	fmt.Printf("  JSONPlugin size: %d bytes (%.1f%% of Concrete (JSON))\n", len(mediumJPBytes), float64(len(mediumJPBytes))/float64(len(mediumJSON))*100)

	// 3. Large
	largeMap := getLargeMap()
	largeJSON, _ := json.Marshal(largeMap)
	largePb, _ := structpb.NewValue(largeMap)
	largePbBytes, _ := proto.Marshal(largePb)

	largeVanillaPayload := getLargeVanillaMsgPayload()
	largeProtoBytes, _ := proto.Marshal(largeVanillaPayload)

	largeVanillaList := getLargeVanillaMsg()
	largeAnyList := make([]*anypb.Any, 100)
	for i := 0; i < 100; i++ {
		largeAnyList[i], _ = anypb.New(largeVanillaList[i])
	}
	var totalAnySize int
	for _, anyVal := range largeAnyList {
		b, _ := proto.Marshal(anyVal)
		totalAnySize += len(b)
	}
	largePjStatic, _ := protojson.Marshal(largeVanillaPayload)
	largePjValue, _ := protojson.Marshal(largePb)
	var totalPjAnySize int
	for _, anyVal := range largeAnyList {
		b, _ := protojson.Marshal(anyVal)
		totalPjAnySize += len(b)
	}

	largeVT := getLargeVTProtoMsgPayload()
	largeVTBytes, _ := largeVT.MarshalVT()

	largeJP := getLargeJSONPluginMsgPayload()
	largeJPBytes, _ := json.Marshal(largeJP)

	largeProtoJSONEnv := &vanillapb.JSONEnvelope{JsonData: string(largeJSON)}
	largeProtoJSONBytes, _ := proto.Marshal(largeProtoJSONEnv)

	fmt.Printf("Large payload:\n")
	fmt.Printf("  Concrete (JSON) size: %d bytes\n", len(largeJSON))
	fmt.Printf("  Concrete (proto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largeProtoBytes), float64(len(largeProtoBytes))/float64(len(largeJSON))*100)
	fmt.Printf("  google.protobuf.Value (proto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largePbBytes), float64(len(largePbBytes))/float64(len(largeJSON))*100)
	fmt.Printf("  google.protobuf.Any (proto) size (approx sum): %d bytes (%.1f%% of Concrete (JSON))\n", totalAnySize, float64(totalAnySize)/float64(len(largeJSON))*100)
	fmt.Printf("  Protobuf + JSON size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largeProtoJSONBytes), float64(len(largeProtoJSONBytes))/float64(len(largeJSON))*100)
	fmt.Printf("  Concrete (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largePjStatic), float64(len(largePjStatic))/float64(len(largeJSON))*100)
	fmt.Printf("  google.protobuf.Value (JSONProto) size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largePjValue), float64(len(largePjValue))/float64(len(largeJSON))*100)
	fmt.Printf("  google.protobuf.Any (JSONProto) size (approx sum): %d bytes (%.1f%% of Concrete (JSON))\n", totalPjAnySize, float64(totalPjAnySize)/float64(len(largeJSON))*100)
	fmt.Printf("  VTProto size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largeVTBytes), float64(len(largeVTBytes))/float64(len(largeJSON))*100)
	fmt.Printf("  JSONPlugin size: %d bytes (%.1f%% of Concrete (JSON))\n", len(largeJPBytes), float64(len(largeJPBytes))/float64(len(largeJSON))*100)
	fmt.Println("=============================")
}

// --- BENCHMARKS ---

// Small Marshal
func BenchmarkMarshal_Small_JSON_Std_Map(b *testing.B) {
	data := getSmallMap()
	for b.Loop() {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkMarshal_Small_JSON_Std_Struct(b *testing.B) {
	data := getSmallStruct()
	for b.Loop() {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkMarshal_Small_JSON_V2_Map(b *testing.B) {
	data := getSmallMap()
	for b.Loop() {
		_, _ = jsonv2.Marshal(data)
	}
}

func BenchmarkMarshal_Small_JSON_V2_Struct(b *testing.B) {
	data := getSmallStruct()
	for b.Loop() {
		_, _ = jsonv2.Marshal(data)
	}
}

func BenchmarkMarshal_Small_Proto_Static(b *testing.B) {
	msg := getSmallVanillaMsg()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_Small_Proto_Value(b *testing.B) {
	data := getSmallMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(pbVal)
	}
}

func BenchmarkMarshal_Small_Proto_Any(b *testing.B) {
	msg := getSmallVanillaMsg()
	anyVal, _ := anypb.New(msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(anyVal)
	}
}

func BenchmarkMarshal_Small_VTProto(b *testing.B) {
	msg := getSmallVTProtoMsg()
	for b.Loop() {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_Small_JSONPlugin_Std(b *testing.B) {
	msg := getSmallJSONPluginMsg()
	for b.Loop() {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMarshal_Small_JSONPlugin_V2(b *testing.B) {
	msg := getSmallJSONPluginMsg()
	for b.Loop() {
		_, _ = jsonv2.Marshal(msg)
	}
}

func BenchmarkMarshal_Small_Proto_Dynamic(b *testing.B) {
	orig := getSmallVanillaMsg()
	data, _ := proto.Marshal(orig)
	msg := dynamicpb.NewMessage(dynamicSmallDesc)
	_ = proto.Unmarshal(data, msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_Small_Proto_HyperPB(b *testing.B) {
	orig := getSmallVanillaMsg()
	data, _ := proto.Marshal(orig)
	msg := hyperpb.NewMessage(hyperSmallType)
	_ = proto.Unmarshal(data, msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

// Small Unmarshal
func BenchmarkUnmarshal_Small_JSON_Std_Map(b *testing.B) {
	data, _ := json.Marshal(getSmallMap())
	for b.Loop() {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Small_JSON_Std_Struct(b *testing.B) {
	data, _ := json.Marshal(getSmallStruct())
	for b.Loop() {
		var s SmallObject
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkUnmarshal_Small_JSON_V2_Map(b *testing.B) {
	data, _ := jsonv2.Marshal(getSmallMap())
	for b.Loop() {
		var m map[string]any
		_ = jsonv2.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Small_JSON_V2_Struct(b *testing.B) {
	data, _ := jsonv2.Marshal(getSmallStruct())
	for b.Loop() {
		var s SmallObject
		_ = jsonv2.Unmarshal(data, &s)
	}
}

func BenchmarkUnmarshal_Small_Proto_Static(b *testing.B) {
	msg := getSmallVanillaMsg()
	data, _ := proto.Marshal(msg)
	for b.Loop() {
		var m vanillapb.SmallObject
		_ = proto.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Small_Proto_Value(b *testing.B) {
	pbVal, _ := structpb.NewValue(getSmallMap())
	data, _ := proto.Marshal(pbVal)
	for b.Loop() {
		var pb structpb.Value
		_ = proto.Unmarshal(data, &pb)
	}
}

func BenchmarkUnmarshal_Small_Proto_Any(b *testing.B) {
	anyVal, _ := anypb.New(getSmallVanillaMsg())
	data, _ := proto.Marshal(anyVal)
	for b.Loop() {
		var a anypb.Any
		_ = proto.Unmarshal(data, &a)
		var msg vanillapb.SmallObject
		_ = anypb.UnmarshalTo(&a, &msg, proto.UnmarshalOptions{})
	}
}

func BenchmarkUnmarshal_Small_VTProto(b *testing.B) {
	msg := getSmallVTProtoMsg()
	data, _ := msg.MarshalVT()
	for b.Loop() {
		var m vtprotopb.SmallObject
		_ = m.UnmarshalVT(data)
	}
}

func BenchmarkUnmarshal_Small_JSONPlugin_Std(b *testing.B) {
	msg := getSmallJSONPluginMsg()
	data, _ := json.Marshal(msg)
	for b.Loop() {
		var m jsonpluginpb.SmallObject
		_ = json.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Small_JSONPlugin_V2(b *testing.B) {
	msg := getSmallJSONPluginMsg()
	data, _ := jsonv2.Marshal(msg)
	for b.Loop() {
		var m jsonpluginpb.SmallObject
		_ = jsonv2.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Small_Proto_Dynamic(b *testing.B) {
	msg := getSmallVanillaMsg()
	data, _ := proto.Marshal(msg)
	desc := dynamicSmallDesc
	b.ResetTimer()
	for b.Loop() {
		m := dynamicpb.NewMessage(desc)
		_ = proto.Unmarshal(data, m)
	}
}

func BenchmarkUnmarshal_Small_Proto_HyperPB(b *testing.B) {
	msg := getSmallVanillaMsg()
	data, _ := proto.Marshal(msg)
	mType := hyperSmallType
	b.ResetTimer()
	for b.Loop() {
		m := hyperpb.NewMessage(mType)
		_ = proto.Unmarshal(data, m)
	}
}

func BenchmarkUnmarshal_Small_Proto_HyperPB_Shared(b *testing.B) {
	msg := getSmallVanillaMsg()
	data, _ := proto.Marshal(msg)
	mType := hyperSmallType
	shared := new(hyperpb.Shared)
	b.ResetTimer()
	for b.Loop() {
		m := shared.NewMessage(mType)
		_ = proto.Unmarshal(data, m)
		shared.Free()
	}
}

// Medium Marshal
func BenchmarkMarshal_Medium_JSON_Std_Map(b *testing.B) {
	data := getMediumMap()
	for b.Loop() {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkMarshal_Medium_JSON_Std_Struct(b *testing.B) {
	data := getMediumStruct()
	for b.Loop() {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkMarshal_Medium_JSON_V2_Map(b *testing.B) {
	data := getMediumMap()
	for b.Loop() {
		_, _ = jsonv2.Marshal(data)
	}
}

func BenchmarkMarshal_Medium_JSON_V2_Struct(b *testing.B) {
	data := getMediumStruct()
	for b.Loop() {
		_, _ = jsonv2.Marshal(data)
	}
}

func BenchmarkMarshal_Medium_Proto_Static(b *testing.B) {
	msg := getMediumVanillaMsg()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_Medium_Proto_Value(b *testing.B) {
	data := getMediumMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(pbVal)
	}
}

func BenchmarkMarshal_Medium_Proto_Any(b *testing.B) {
	msg := getMediumVanillaMsg()
	anyVal, _ := anypb.New(msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(anyVal)
	}
}

func BenchmarkMarshal_Medium_VTProto(b *testing.B) {
	msg := getMediumVTProtoMsg()
	for b.Loop() {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_Medium_JSONPlugin_Std(b *testing.B) {
	msg := getMediumJSONPluginMsg()
	for b.Loop() {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMarshal_Medium_JSONPlugin_V2(b *testing.B) {
	msg := getMediumJSONPluginMsg()
	for b.Loop() {
		_, _ = jsonv2.Marshal(msg)
	}
}

func BenchmarkMarshal_Medium_Proto_Dynamic(b *testing.B) {
	orig := getMediumVanillaMsg()
	data, _ := proto.Marshal(orig)
	msg := dynamicpb.NewMessage(dynamicMediumDesc)
	_ = proto.Unmarshal(data, msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_Medium_Proto_HyperPB(b *testing.B) {
	orig := getMediumVanillaMsg()
	data, _ := proto.Marshal(orig)
	msg := hyperpb.NewMessage(hyperMediumType)
	_ = proto.Unmarshal(data, msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

// Medium Unmarshal
func BenchmarkUnmarshal_Medium_JSON_Std_Map(b *testing.B) {
	data, _ := json.Marshal(getMediumMap())
	for b.Loop() {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Medium_JSON_Std_Struct(b *testing.B) {
	data, _ := json.Marshal(getMediumStruct())
	for b.Loop() {
		var s MediumEvent
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkUnmarshal_Medium_JSON_V2_Map(b *testing.B) {
	data, _ := jsonv2.Marshal(getMediumMap())
	for b.Loop() {
		var m map[string]any
		_ = jsonv2.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Medium_JSON_V2_Struct(b *testing.B) {
	data, _ := jsonv2.Marshal(getMediumStruct())
	for b.Loop() {
		var s MediumEvent
		_ = jsonv2.Unmarshal(data, &s)
	}
}

func BenchmarkUnmarshal_Medium_Proto_Static(b *testing.B) {
	msg := getMediumVanillaMsg()
	data, _ := proto.Marshal(msg)
	for b.Loop() {
		var m vanillapb.MediumEvent
		_ = proto.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Medium_Proto_Value(b *testing.B) {
	pbVal, _ := structpb.NewValue(getMediumMap())
	data, _ := proto.Marshal(pbVal)
	for b.Loop() {
		var pb structpb.Value
		_ = proto.Unmarshal(data, &pb)
	}
}

func BenchmarkUnmarshal_Medium_Proto_Any(b *testing.B) {
	anyVal, _ := anypb.New(getMediumVanillaMsg())
	data, _ := proto.Marshal(anyVal)
	for b.Loop() {
		var a anypb.Any
		_ = proto.Unmarshal(data, &a)
		var msg vanillapb.MediumEvent
		_ = anypb.UnmarshalTo(&a, &msg, proto.UnmarshalOptions{})
	}
}

func BenchmarkUnmarshal_Medium_VTProto(b *testing.B) {
	msg := getMediumVTProtoMsg()
	data, _ := msg.MarshalVT()
	for b.Loop() {
		var m vtprotopb.MediumEvent
		_ = m.UnmarshalVT(data)
	}
}

func BenchmarkUnmarshal_Medium_JSONPlugin_Std(b *testing.B) {
	msg := getMediumJSONPluginMsg()
	data, _ := json.Marshal(msg)
	for b.Loop() {
		var m jsonpluginpb.MediumEvent
		_ = json.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Medium_JSONPlugin_V2(b *testing.B) {
	msg := getMediumJSONPluginMsg()
	data, _ := jsonv2.Marshal(msg)
	for b.Loop() {
		var m jsonpluginpb.MediumEvent
		_ = jsonv2.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Medium_Proto_Dynamic(b *testing.B) {
	msg := getMediumVanillaMsg()
	data, _ := proto.Marshal(msg)
	desc := dynamicMediumDesc
	b.ResetTimer()
	for b.Loop() {
		m := dynamicpb.NewMessage(desc)
		_ = proto.Unmarshal(data, m)
	}
}

func BenchmarkUnmarshal_Medium_Proto_HyperPB(b *testing.B) {
	msg := getMediumVanillaMsg()
	data, _ := proto.Marshal(msg)
	mType := hyperMediumType
	b.ResetTimer()
	for b.Loop() {
		m := hyperpb.NewMessage(mType)
		_ = proto.Unmarshal(data, m)
	}
}

func BenchmarkUnmarshal_Medium_Proto_HyperPB_Shared(b *testing.B) {
	msg := getMediumVanillaMsg()
	data, _ := proto.Marshal(msg)
	mType := hyperMediumType
	shared := new(hyperpb.Shared)
	b.ResetTimer()
	for b.Loop() {
		m := shared.NewMessage(mType)
		_ = proto.Unmarshal(data, m)
		shared.Free()
	}
}

// Large Marshal
func BenchmarkMarshal_Large_JSON_Std_Map(b *testing.B) {
	data := getLargeMap()
	for b.Loop() {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkMarshal_Large_JSON_Std_Struct(b *testing.B) {
	data := getLargeStruct()
	for b.Loop() {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkMarshal_Large_JSON_V2_Map(b *testing.B) {
	data := getLargeMap()
	for b.Loop() {
		_, _ = jsonv2.Marshal(data)
	}
}

func BenchmarkMarshal_Large_JSON_V2_Struct(b *testing.B) {
	data := getLargeStruct()
	for b.Loop() {
		_, _ = jsonv2.Marshal(data)
	}
}

func BenchmarkMarshal_Large_Proto_Static(b *testing.B) {
	data := getLargeVanillaMsgPayload()
	for b.Loop() {
		_, _ = proto.Marshal(data)
	}
}

func BenchmarkMarshal_Large_Proto_Value(b *testing.B) {
	data := getLargeMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(pbVal)
	}
}

func BenchmarkMarshal_Large_Proto_Any(b *testing.B) {
	msgList := getLargeVanillaMsg()
	largeAnyList := make([]*anypb.Any, 100)
	for j := range 100 {
		largeAnyList[j], _ = anypb.New(msgList[j])
	}
	b.ResetTimer()
	for b.Loop() {
		for j := range 100 {
			_, _ = proto.Marshal(largeAnyList[j])
		}
	}
}

func BenchmarkMarshal_Large_VTProto(b *testing.B) {
	data := getLargeVTProtoMsgPayload()
	for b.Loop() {
		_, _ = data.MarshalVT()
	}
}

func BenchmarkMarshal_Large_JSONPlugin_Std(b *testing.B) {
	msg := getLargeJSONPluginMsgPayload()
	for b.Loop() {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMarshal_Large_JSONPlugin_V2(b *testing.B) {
	msg := getLargeJSONPluginMsgPayload()
	for b.Loop() {
		_, _ = jsonv2.Marshal(msg)
	}
}

func BenchmarkMarshal_Large_Proto_Dynamic(b *testing.B) {
	orig := getLargeVanillaMsgPayload()
	data, _ := proto.Marshal(orig)
	msg := dynamicpb.NewMessage(dynamicLargeDesc)
	_ = proto.Unmarshal(data, msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_Large_Proto_HyperPB(b *testing.B) {
	orig := getLargeVanillaMsgPayload()
	data, _ := proto.Marshal(orig)
	msg := hyperpb.NewMessage(hyperLargeType)
	_ = proto.Unmarshal(data, msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = proto.Marshal(msg)
	}
}

// Large Unmarshal
func BenchmarkUnmarshal_Large_JSON_Std_Map(b *testing.B) {
	data, _ := json.Marshal(getLargeMap())
	for b.Loop() {
		var m []any
		_ = json.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Large_JSON_Std_Struct(b *testing.B) {
	data, _ := json.Marshal(getLargeStruct())
	for b.Loop() {
		var s []MediumEvent
		_ = json.Unmarshal(data, &s)
	}
}

func BenchmarkUnmarshal_Large_JSON_V2_Map(b *testing.B) {
	data, _ := jsonv2.Marshal(getLargeMap())
	for b.Loop() {
		var m []any
		_ = jsonv2.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Large_JSON_V2_Struct(b *testing.B) {
	data, _ := jsonv2.Marshal(getLargeStruct())
	for b.Loop() {
		var s []MediumEvent
		_ = jsonv2.Unmarshal(data, &s)
	}
}

func BenchmarkUnmarshal_Large_Proto_Static(b *testing.B) {
	data, _ := proto.Marshal(getLargeVanillaMsgPayload())
	for b.Loop() {
		var p vanillapb.LargeEventPayload
		_ = proto.Unmarshal(data, &p)
	}
}

func BenchmarkUnmarshal_Large_Proto_Value(b *testing.B) {
	pbVal, _ := structpb.NewValue(getLargeMap())
	data, _ := proto.Marshal(pbVal)
	for b.Loop() {
		var pb structpb.Value
		_ = proto.Unmarshal(data, &pb)
	}
}

func BenchmarkUnmarshal_Large_Proto_Any(b *testing.B) {
	msgList := getLargeVanillaMsg()
	largeAnyList := make([]*anypb.Any, 100)
	var dataBytes [][]byte
	for j := range 100 {
		largeAnyList[j], _ = anypb.New(msgList[j])
		bytes, _ := proto.Marshal(largeAnyList[j])
		dataBytes = append(dataBytes, bytes)
	}
	for b.Loop() {
		for j := range 100 {
			var a anypb.Any
			_ = proto.Unmarshal(dataBytes[j], &a)
			var msg vanillapb.MediumEvent
			_ = anypb.UnmarshalTo(&a, &msg, proto.UnmarshalOptions{})
		}
	}
}

func BenchmarkUnmarshal_Large_VTProto(b *testing.B) {
	msg := getLargeVTProtoMsgPayload()
	data, _ := msg.MarshalVT()
	for b.Loop() {
		var p vtprotopb.LargeEventPayload
		_ = p.UnmarshalVT(data)
	}
}

func BenchmarkUnmarshal_Large_JSONPlugin_Std(b *testing.B) {
	msg := getLargeJSONPluginMsgPayload()
	data, _ := json.Marshal(msg)
	for b.Loop() {
		var p jsonpluginpb.LargeEventPayload
		_ = json.Unmarshal(data, &p)
	}
}

func BenchmarkUnmarshal_Large_JSONPlugin_V2(b *testing.B) {
	msg := getLargeJSONPluginMsgPayload()
	data, _ := jsonv2.Marshal(msg)
	for b.Loop() {
		var p jsonpluginpb.LargeEventPayload
		_ = jsonv2.Unmarshal(data, &p)
	}
}

func BenchmarkUnmarshal_Large_Proto_Dynamic(b *testing.B) {
	msg := getLargeVanillaMsgPayload()
	data, _ := proto.Marshal(msg)
	desc := dynamicLargeDesc
	b.ResetTimer()
	for b.Loop() {
		m := dynamicpb.NewMessage(desc)
		_ = proto.Unmarshal(data, m)
	}
}

func BenchmarkUnmarshal_Large_Proto_HyperPB(b *testing.B) {
	msg := getLargeVanillaMsgPayload()
	data, _ := proto.Marshal(msg)
	mType := hyperLargeType
	b.ResetTimer()
	for b.Loop() {
		m := hyperpb.NewMessage(mType)
		_ = proto.Unmarshal(data, m)
	}
}

func BenchmarkUnmarshal_Large_Proto_HyperPB_Shared(b *testing.B) {
	msg := getLargeVanillaMsgPayload()
	data, _ := proto.Marshal(msg)
	mType := hyperLargeType
	shared := new(hyperpb.Shared)
	b.ResetTimer()
	for b.Loop() {
		m := shared.NewMessage(mType)
		_ = proto.Unmarshal(data, m)
		shared.Free()
	}
}

// --- ProtoJSON Benchmarks ---

// Small Marshal ProtoJSON
func BenchmarkMarshal_Small_ProtoJSON_Static(b *testing.B) {
	msg := getSmallVanillaMsg()
	for b.Loop() {
		_, _ = protojson.Marshal(msg)
	}
}

func BenchmarkMarshal_Small_ProtoJSON_Value(b *testing.B) {
	data := getSmallMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = protojson.Marshal(pbVal)
	}
}

func BenchmarkMarshal_Small_ProtoJSON_Any(b *testing.B) {
	msg := getSmallVanillaMsg()
	anyVal, _ := anypb.New(msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = protojson.Marshal(anyVal)
	}
}

// Small Unmarshal ProtoJSON
func BenchmarkUnmarshal_Small_ProtoJSON_Static(b *testing.B) {
	msg := getSmallVanillaMsg()
	data, _ := protojson.Marshal(msg)
	for b.Loop() {
		var m vanillapb.SmallObject
		_ = protojson.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Small_ProtoJSON_Value(b *testing.B) {
	pbVal, _ := structpb.NewValue(getSmallMap())
	data, _ := protojson.Marshal(pbVal)
	for b.Loop() {
		var pb structpb.Value
		_ = protojson.Unmarshal(data, &pb)
	}
}

func BenchmarkUnmarshal_Small_ProtoJSON_Any(b *testing.B) {
	anyVal, _ := anypb.New(getSmallVanillaMsg())
	data, _ := protojson.Marshal(anyVal)
	for b.Loop() {
		var a anypb.Any
		_ = protojson.Unmarshal(data, &a)
		var msg vanillapb.SmallObject
		_ = anypb.UnmarshalTo(&a, &msg, proto.UnmarshalOptions{})
	}
}

// Medium Marshal ProtoJSON
func BenchmarkMarshal_Medium_ProtoJSON_Static(b *testing.B) {
	msg := getMediumVanillaMsg()
	for b.Loop() {
		_, _ = protojson.Marshal(msg)
	}
}

func BenchmarkMarshal_Medium_ProtoJSON_Value(b *testing.B) {
	data := getMediumMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = protojson.Marshal(pbVal)
	}
}

func BenchmarkMarshal_Medium_ProtoJSON_Any(b *testing.B) {
	msg := getMediumVanillaMsg()
	anyVal, _ := anypb.New(msg)
	b.ResetTimer()
	for b.Loop() {
		_, _ = protojson.Marshal(anyVal)
	}
}

// Medium Unmarshal ProtoJSON
func BenchmarkUnmarshal_Medium_ProtoJSON_Static(b *testing.B) {
	msg := getMediumVanillaMsg()
	data, _ := protojson.Marshal(msg)
	for b.Loop() {
		var m vanillapb.MediumEvent
		_ = protojson.Unmarshal(data, &m)
	}
}

func BenchmarkUnmarshal_Medium_ProtoJSON_Value(b *testing.B) {
	pbVal, _ := structpb.NewValue(getMediumMap())
	data, _ := protojson.Marshal(pbVal)
	for b.Loop() {
		var pb structpb.Value
		_ = protojson.Unmarshal(data, &pb)
	}
}

func BenchmarkUnmarshal_Medium_ProtoJSON_Any(b *testing.B) {
	anyVal, _ := anypb.New(getMediumVanillaMsg())
	data, _ := protojson.Marshal(anyVal)
	for b.Loop() {
		var a anypb.Any
		_ = protojson.Unmarshal(data, &a)
		var msg vanillapb.MediumEvent
		_ = anypb.UnmarshalTo(&a, &msg, proto.UnmarshalOptions{})
	}
}

// Large Marshal ProtoJSON
func BenchmarkMarshal_Large_ProtoJSON_Static(b *testing.B) {
	data := getLargeVanillaMsgPayload()
	for b.Loop() {
		_, _ = protojson.Marshal(data)
	}
}

func BenchmarkMarshal_Large_ProtoJSON_Value(b *testing.B) {
	data := getLargeMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = protojson.Marshal(pbVal)
	}
}

func BenchmarkMarshal_Large_ProtoJSON_Any(b *testing.B) {
	msgList := getLargeVanillaMsg()
	largeAnyList := make([]*anypb.Any, 100)
	for j := range 100 {
		largeAnyList[j], _ = anypb.New(msgList[j])
	}
	b.ResetTimer()
	for b.Loop() {
		for j := range 100 {
			_, _ = protojson.Marshal(largeAnyList[j])
		}
	}
}

// Large Unmarshal ProtoJSON
func BenchmarkUnmarshal_Large_ProtoJSON_Static(b *testing.B) {
	data, _ := protojson.Marshal(getLargeVanillaMsgPayload())
	for b.Loop() {
		var p vanillapb.LargeEventPayload
		_ = protojson.Unmarshal(data, &p)
	}
}

func BenchmarkUnmarshal_Large_ProtoJSON_Value(b *testing.B) {
	pbVal, _ := structpb.NewValue(getLargeMap())
	data, _ := protojson.Marshal(pbVal)
	for b.Loop() {
		var pb structpb.Value
		_ = protojson.Unmarshal(data, &pb)
	}
}

func BenchmarkUnmarshal_Large_ProtoJSON_Any(b *testing.B) {
	msgList := getLargeVanillaMsg()
	largeAnyList := make([]*anypb.Any, 100)
	var dataBytes [][]byte
	for j := range 100 {
		largeAnyList[j], _ = anypb.New(msgList[j])
		bytes, _ := protojson.Marshal(largeAnyList[j])
		dataBytes = append(dataBytes, bytes)
	}
	for b.Loop() {
		for j := range 100 {
			var a anypb.Any
			_ = protojson.Unmarshal(dataBytes[j], &a)
			var msg vanillapb.MediumEvent
			_ = anypb.UnmarshalTo(&a, &msg, proto.UnmarshalOptions{})
		}
	}
}

// --- Protobuf + JSON (Opaque JSON Packaging) Benchmarks ---

func BenchmarkMarshal_Small_Proto_JSON(b *testing.B) {
	data := getSmallStruct()
	for b.Loop() {
		jsonBytes, _ := json.Marshal(&data)
		envelope := &vanillapb.JSONEnvelope{JsonData: string(jsonBytes)}
		_, _ = proto.Marshal(envelope)
	}
}

func BenchmarkUnmarshal_Small_Proto_JSON(b *testing.B) {
	jsonBytes, _ := json.Marshal(getSmallStruct())
	envelope := &vanillapb.JSONEnvelope{JsonData: string(jsonBytes)}
	data, _ := proto.Marshal(envelope)
	for b.Loop() {
		var env vanillapb.JSONEnvelope
		_ = proto.Unmarshal(data, &env)
		var dest SmallObject
		_ = json.Unmarshal([]byte(env.JsonData), &dest)
	}
}

func BenchmarkMarshal_Medium_Proto_JSON(b *testing.B) {
	data := getMediumStruct()
	for b.Loop() {
		jsonBytes, _ := json.Marshal(&data)
		envelope := &vanillapb.JSONEnvelope{JsonData: string(jsonBytes)}
		_, _ = proto.Marshal(envelope)
	}
}

func BenchmarkUnmarshal_Medium_Proto_JSON(b *testing.B) {
	jsonBytes, _ := json.Marshal(getMediumStruct())
	envelope := &vanillapb.JSONEnvelope{JsonData: string(jsonBytes)}
	data, _ := proto.Marshal(envelope)
	for b.Loop() {
		var env vanillapb.JSONEnvelope
		_ = proto.Unmarshal(data, &env)
		var dest MediumEvent
		_ = json.Unmarshal([]byte(env.JsonData), &dest)
	}
}

func BenchmarkMarshal_Large_Proto_JSON(b *testing.B) {
	data := getLargeStruct()
	for b.Loop() {
		jsonBytes, _ := json.Marshal(&data)
		envelope := &vanillapb.JSONEnvelope{JsonData: string(jsonBytes)}
		_, _ = proto.Marshal(envelope)
	}
}

func BenchmarkUnmarshal_Large_Proto_JSON(b *testing.B) {
	jsonBytes, _ := json.Marshal(getLargeStruct())
	envelope := &vanillapb.JSONEnvelope{JsonData: string(jsonBytes)}
	data, _ := proto.Marshal(envelope)
	for b.Loop() {
		var env vanillapb.JSONEnvelope
		_ = proto.Unmarshal(data, &env)
		var dest []MediumEvent
		_ = json.Unmarshal([]byte(env.JsonData), &dest)
	}
}

// --- Construction & Conversion (Appendix) Benchmarks ---

func BenchmarkConstruction_Small_Proto_Value(b *testing.B) {
	data := getSmallMap()
	for b.Loop() {
		_, _ = structpb.NewValue(data)
	}
}

func BenchmarkConversion_Small_Proto_Value(b *testing.B) {
	data := getSmallMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_ = pbVal.AsInterface()
	}
}

func BenchmarkConstruction_Medium_Proto_Value(b *testing.B) {
	data := getMediumMap()
	for b.Loop() {
		_, _ = structpb.NewValue(data)
	}
}

func BenchmarkConversion_Medium_Proto_Value(b *testing.B) {
	data := getMediumMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_ = pbVal.AsInterface()
	}
}

func BenchmarkConstruction_Large_Proto_Value(b *testing.B) {
	data := getLargeMap()
	for b.Loop() {
		_, _ = structpb.NewValue(data)
	}
}

func BenchmarkConversion_Large_Proto_Value(b *testing.B) {
	data := getLargeMap()
	pbVal, _ := structpb.NewValue(data)
	b.ResetTimer()
	for b.Loop() {
		_ = pbVal.AsInterface()
	}
}


