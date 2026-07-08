package benchmark

import (
	"fmt"
	"testing"

	pb "ints-bench/proto"

	"buf.build/go/hyperpb"
	"google.golang.org/protobuf/proto"
)

var (
	hyperInt64VarintType *hyperpb.MessageType
	hyperInt64SintType   *hyperpb.MessageType
	hyperInt64FixedType  *hyperpb.MessageType
)

func init() {
	hyperInt64VarintType = hyperpb.CompileMessageDescriptor((*pb.Int64Varint)(nil).ProtoReflect().Descriptor())
	hyperInt64SintType = hyperpb.CompileMessageDescriptor((*pb.Int64Sint)(nil).ProtoReflect().Descriptor())
	hyperInt64FixedType = hyperpb.CompileMessageDescriptor((*pb.Int64Fixed)(nil).ProtoReflect().Descriptor())
}

// --- Test Data Generation (64-bit) ---

func makeSmallPositive64() []int64 {
	res := make([]int64, 1000)
	for i := range res {
		res[i] = int64(i % 100)
	}
	return res
}

func makeLargePositive64() []int64 {
	res := make([]int64, 1000)
	for i := range res {
		res[i] = int64((1 << 50) + i)
	}
	return res
}

func makeNegative64() []int64 {
	res := make([]int64, 1000)
	for i := range res {
		res[i] = int64(-(i % 100) - 1)
	}
	return res
}

func makeSmallPositiveU64() []uint64 {
	res := make([]uint64, 1000)
	for i := range res {
		res[i] = uint64(i % 100)
	}
	return res
}

func makeLargePositiveU64() []uint64 {
	res := make([]uint64, 1000)
	for i := range res {
		res[i] = uint64((1 << 50) + i)
	}
	return res
}

// --- Test Data Generation (32-bit) ---

func makeSmallPositive32() []int32 {
	res := make([]int32, 1000)
	for i := range res {
		res[i] = int32(i % 100)
	}
	return res
}

func makeLargePositive32() []int32 {
	res := make([]int32, 1000)
	for i := range res {
		res[i] = int32((1 << 28) + i)
	}
	return res
}

func makeNegative32() []int32 {
	res := make([]int32, 1000)
	for i := range res {
		res[i] = int32(-(i % 100) - 1)
	}
	return res
}

func makeSmallPositiveU32() []uint32 {
	res := make([]uint32, 1000)
	for i := range res {
		res[i] = uint32(i % 100)
	}
	return res
}

func makeLargePositiveU32() []uint32 {
	res := make([]uint32, 1000)
	for i := range res {
		res[i] = uint32((1 << 28) + i)
	}
	return res
}

// --- Size Print Helper ---

func TestPrintSizes(t *testing.T) {
	fmt.Println("=== SERIALIZED DATA SIZES (1000 elements) ===")

	// 64-bit
	sp64Varint, _ := proto.Marshal(&pb.Int64Varint{Values: makeSmallPositive64()})
	sp64Sint, _ := proto.Marshal(&pb.Int64Sint{Values: makeSmallPositive64()})
	sp64Fixed, _ := proto.Marshal(&pb.Int64Fixed{Values: makeSmallPositive64()})
	sp64Ufixed, _ := proto.Marshal(&pb.Int64Ufixed{Values: makeSmallPositiveU64()})

	lp64Varint, _ := proto.Marshal(&pb.Int64Varint{Values: makeLargePositive64()})
	lp64Sint, _ := proto.Marshal(&pb.Int64Sint{Values: makeLargePositive64()})
	lp64Fixed, _ := proto.Marshal(&pb.Int64Fixed{Values: makeLargePositive64()})
	lp64Ufixed, _ := proto.Marshal(&pb.Int64Ufixed{Values: makeLargePositiveU64()})

	n64Varint, _ := proto.Marshal(&pb.Int64Varint{Values: makeNegative64()})
	n64Sint, _ := proto.Marshal(&pb.Int64Sint{Values: makeNegative64()})
	n64Fixed, _ := proto.Marshal(&pb.Int64Fixed{Values: makeNegative64()})

	fmt.Printf("64-bit Small Positive: Varint=%d B, Sint=%d B, Fixed=%d B, Ufixed=%d B\n", len(sp64Varint), len(sp64Sint), len(sp64Fixed), len(sp64Ufixed))
	fmt.Printf("64-bit Large Positive: Varint=%d B, Sint=%d B, Fixed=%d B, Ufixed=%d B\n", len(lp64Varint), len(lp64Sint), len(lp64Fixed), len(lp64Ufixed))
	fmt.Printf("64-bit Negative:       Varint=%d B, Sint=%d B, Fixed=%d B\n", len(n64Varint), len(n64Sint), len(n64Fixed))

	// 32-bit
	sp32Varint, _ := proto.Marshal(&pb.Int32Varint{Values: makeSmallPositive32()})
	sp32Sint, _ := proto.Marshal(&pb.Int32Sint{Values: makeSmallPositive32()})
	sp32Fixed, _ := proto.Marshal(&pb.Int32Fixed{Values: makeSmallPositive32()})
	sp32Ufixed, _ := proto.Marshal(&pb.Int32Ufixed{Values: makeSmallPositiveU32()})

	lp32Varint, _ := proto.Marshal(&pb.Int32Varint{Values: makeLargePositive32()})
	lp32Sint, _ := proto.Marshal(&pb.Int32Sint{Values: makeLargePositive32()})
	lp32Fixed, _ := proto.Marshal(&pb.Int32Fixed{Values: makeLargePositive32()})
	lp32Ufixed, _ := proto.Marshal(&pb.Int32Ufixed{Values: makeLargePositiveU32()})

	n32Varint, _ := proto.Marshal(&pb.Int32Varint{Values: makeNegative32()})
	n32Sint, _ := proto.Marshal(&pb.Int32Sint{Values: makeNegative32()})
	n32Fixed, _ := proto.Marshal(&pb.Int32Fixed{Values: makeNegative32()})

	fmt.Printf("32-bit Small Positive: Varint=%d B, Sint=%d B, Fixed=%d B, Ufixed=%d B\n", len(sp32Varint), len(sp32Sint), len(sp32Fixed), len(sp32Ufixed))
	fmt.Printf("32-bit Large Positive: Varint=%d B, Sint=%d B, Fixed=%d B, Ufixed=%d B\n", len(lp32Varint), len(lp32Sint), len(lp32Fixed), len(lp32Ufixed))
	fmt.Printf("32-bit Negative:       Varint=%d B, Sint=%d B, Fixed=%d B\n", len(n32Varint), len(n32Sint), len(n32Fixed))
	fmt.Println("=============================================")
}

// --- 64-Bit Benchmarks ---

// Marshal

func BenchmarkMarshal_64_SmallPositive_Varint(b *testing.B) {
	msg := &pb.Int64Varint{Values: makeSmallPositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_SmallPositive_Varint_VT(b *testing.B) {
	msg := &pb.Int64Varint{Values: makeSmallPositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_SmallPositive_Sint(b *testing.B) {
	msg := &pb.Int64Sint{Values: makeSmallPositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_SmallPositive_Sint_VT(b *testing.B) {
	msg := &pb.Int64Sint{Values: makeSmallPositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_SmallPositive_Fixed(b *testing.B) {
	msg := &pb.Int64Fixed{Values: makeSmallPositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_SmallPositive_Fixed_VT(b *testing.B) {
	msg := &pb.Int64Fixed{Values: makeSmallPositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_LargePositive_Varint(b *testing.B) {
	msg := &pb.Int64Varint{Values: makeLargePositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_LargePositive_Varint_VT(b *testing.B) {
	msg := &pb.Int64Varint{Values: makeLargePositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_LargePositive_Sint(b *testing.B) {
	msg := &pb.Int64Sint{Values: makeLargePositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_LargePositive_Sint_VT(b *testing.B) {
	msg := &pb.Int64Sint{Values: makeLargePositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_LargePositive_Fixed(b *testing.B) {
	msg := &pb.Int64Fixed{Values: makeLargePositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_LargePositive_Fixed_VT(b *testing.B) {
	msg := &pb.Int64Fixed{Values: makeLargePositive64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_Negative_Varint(b *testing.B) {
	msg := &pb.Int64Varint{Values: makeNegative64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_Negative_Varint_VT(b *testing.B) {
	msg := &pb.Int64Varint{Values: makeNegative64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_Negative_Sint(b *testing.B) {
	msg := &pb.Int64Sint{Values: makeNegative64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_Negative_Sint_VT(b *testing.B) {
	msg := &pb.Int64Sint{Values: makeNegative64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

func BenchmarkMarshal_64_Negative_Fixed(b *testing.B) {
	msg := &pb.Int64Fixed{Values: makeNegative64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMarshal_64_Negative_Fixed_VT(b *testing.B) {
	msg := &pb.Int64Fixed{Values: makeNegative64()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalVT()
	}
}

// Unmarshal

func BenchmarkUnmarshal_64_SmallPositive_Varint(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeSmallPositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Varint{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Varint_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeSmallPositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Varint{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Varint_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeSmallPositive64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64VarintType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Sint(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeSmallPositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Sint{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Sint_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeSmallPositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Sint{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Sint_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeSmallPositive64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64SintType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Fixed(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeSmallPositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Fixed{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Fixed_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeSmallPositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Fixed{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_SmallPositive_Fixed_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeSmallPositive64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64FixedType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_LargePositive_Varint(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeLargePositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Varint{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_LargePositive_Varint_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeLargePositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Varint{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_LargePositive_Varint_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeLargePositive64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64VarintType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_LargePositive_Sint(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeLargePositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Sint{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_LargePositive_Sint_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeLargePositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Sint{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_LargePositive_Sint_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeLargePositive64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64SintType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_LargePositive_Fixed(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeLargePositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Fixed{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_LargePositive_Fixed_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeLargePositive64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Fixed{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_LargePositive_Fixed_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeLargePositive64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64FixedType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_Negative_Varint(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeNegative64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Varint{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_Negative_Varint_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeNegative64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Varint{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_Negative_Varint_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Varint{Values: makeNegative64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64VarintType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_Negative_Sint(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeNegative64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Sint{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_Negative_Sint_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeNegative64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Sint{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_Negative_Sint_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Sint{Values: makeNegative64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64SintType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}

func BenchmarkUnmarshal_64_Negative_Fixed(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeNegative64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Fixed{}
		_ = proto.Unmarshal(payload, msg)
	}
}

func BenchmarkUnmarshal_64_Negative_Fixed_VT(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeNegative64()})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &pb.Int64Fixed{}
		_ = msg.UnmarshalVT(payload)
	}
}

func BenchmarkUnmarshal_64_Negative_Fixed_HyperPB_Shared(b *testing.B) {
	payload, _ := proto.Marshal(&pb.Int64Fixed{Values: makeNegative64()})
	shared := new(hyperpb.Shared)
	mType := hyperInt64FixedType
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := shared.NewMessage(mType)
		_ = proto.Unmarshal(payload, msg)
		shared.Free()
	}
}
