package varintbench

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/klauspost/compress/zstd"
)

type codec struct {
	name   string
	encode func([]byte, []uint64) []byte
	decode func([]byte, []uint64) int
	maxVal uint64
}

var allCodecs = []codec{
	{"LEB128", EncodeLEB128, DecodeLEB128, math.MaxUint64},
	{"LEB128Protowire", EncodeLEB128Protowire, DecodeLEB128Protowire, math.MaxUint64},
	{"LEB128Stdlib", EncodeLEB128Stdlib, DecodeLEB128Stdlib, math.MaxUint64},
	{"QUIC", EncodeQUIC, DecodeQUIC, (1 << 62) - 1},
	{"OmniBuf", EncodeOmniBuf, DecodeOmniBuf, math.MaxUint64},
	{"Prefix9", EncodePrefix9, DecodePrefix9Branchless, math.MaxUint64},
	{"Prefix9Fast1B", EncodePrefix9, DecodePrefix9Fast1B, math.MaxUint64},
	{"MsgPack", EncodeMsgPack, DecodeMsgPack, math.MaxUint64},
	{"Fixed64", EncodeFixed64, DecodeFixed64, math.MaxUint64},
	{"Fixed64Unsafe", EncodeFixed64Unsafe, DecodeFixed64Unsafe, math.MaxUint64},
}

func TestRoundTrip(t *testing.T) {
	edgeVals := []uint64{
		0, 1, 62, 63, 64, 126, 127, 128, 255, 256,
		16382, 16383, 16384, 65535, 65536,
		(1 << 21) - 1, 1 << 21,
		(1 << 28) - 1, 1 << 28,
		(1 << 29) - 1, 1 << 29,
		(1 << 30) - 1, 1 << 30,
		(1 << 32) - 1, 1 << 32,
		(1 << 35) - 1, 1 << 35,
		(1 << 42) - 1, 1 << 42,
		(1 << 49) - 1, 1 << 49,
		(1 << 56) - 1, 1 << 56,
		(1 << 62) - 1,
		1 << 62,
		math.MaxUint64 - 1,
		math.MaxUint64,
	}

	for _, c := range allCodecs {
		t.Run(c.name, func(t *testing.T) {
			var input []uint64
			for _, v := range edgeVals {
				if v <= c.maxVal {
					input = append(input, v)
				}
			}
			encoded := c.encode(nil, input)
			got := make([]uint64, len(input))
			n := c.decode(encoded, got)
			if n != len(input) {
				t.Fatalf("decoded %d values, want %d", n, len(input))
			}
			for i := range input {
				if got[i] != input[i] {
					t.Fatalf("index %d: got %d (0x%x), want %d (0x%x)", i, got[i], got[i], input[i], input[i])
				}
			}
		})
	}
}

const benchCount = 10_000

type dataset struct {
	name   string
	vals   []uint64
	maxVal uint64
}

func makeDatasets() []dataset {
	rng := rand.New(rand.NewPCG(42, 1024))

	// 1. Small: 0..63 (fits in 1 byte for every variable-length codec)
	small := make([]uint64, benchCount)
	for i := range small {
		small[i] = uint64(rng.IntN(64))
	}

	// 2. Large: 2^40 .. 2^40 + 100000 (41-bit numbers)
	large := make([]uint64, benchCount)
	for i := range large {
		large[i] = (1 << 40) + uint64(rng.IntN(100_000))
	}

	// 3. FullWidth: uniformly random 64-bit integers (exceeds QUIC's 62-bit limit)
	fullWidth := make([]uint64, benchCount)
	for i := range fullWidth {
		fullWidth[i] = rng.Uint64()
	}

	// 4. Skewed: 70% small (0..127), 20% medium, 8% large, 2% huge, shuffled
	skewed := make([]uint64, benchCount)
	for i := range skewed {
		p := rng.IntN(100)
		switch {
		case p < 70:
			skewed[i] = uint64(rng.IntN(128))
		case p < 90:
			skewed[i] = 128 + uint64(rng.IntN(16000))
		case p < 98:
			skewed[i] = 65536 + uint64(rng.IntN(100_000_000))
		default:
			skewed[i] = (1 << 40) + uint64(rng.IntN(1<<20))
		}
	}

	// 5. MixedRandom: equal 25% mix across 4 tiers, shuffled
	mixed := make([]uint64, benchCount)
	for i := range mixed {
		switch rng.IntN(4) {
		case 0:
			mixed[i] = uint64(rng.IntN(128))
		case 1:
			mixed[i] = 128 + uint64(rng.IntN(16000))
		case 2:
			mixed[i] = 65536 + uint64(rng.IntN(100_000_000))
		case 3:
			mixed[i] = (1 << 40) + uint64(rng.IntN(1<<20))
		}
	}

	// 6. MixedSorted: identical values to MixedRandom, sorted ascending
	mixedSorted := slices.Clone(mixed)
	slices.Sort(mixedSorted)

	return []dataset{
		{"Small", small, 63},
		{"Large", large, (1 << 40) + 100_000},
		{"FullWidth", fullWidth, math.MaxUint64},
		{"Skewed", skewed, (1 << 40) + (1 << 20)},
		{"MixedRandom", mixed, (1 << 40) + (1 << 20)},
		{"MixedSorted", mixedSorted, (1 << 40) + (1 << 20)},
	}
}

func TestPrintSizes(t *testing.T) {
	datasets := makeDatasets()
	fmt.Printf("%-16s", "Codec")
	for _, ds := range datasets {
		fmt.Printf(" | %11s", ds.name)
	}
	fmt.Println()
	for _, c := range allCodecs {
		fmt.Printf("%-16s", c.name)
		for _, ds := range datasets {
			if ds.maxVal > c.maxVal {
				fmt.Printf(" | %11s", "N/A")
				continue
			}
			enc := c.encode(nil, ds.vals)
			fmt.Printf(" | %9d B", len(enc))
		}
		fmt.Println()
	}
}

func TestCompression(t *testing.T) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()

	datasets := makeDatasets()
	selectedDatasets := []string{"Small", "Skewed", "FullWidth"}
	selectedCodecs := []codec{
		{"LEB128", EncodeLEB128, DecodeLEB128, math.MaxUint64},
		{"OmniBuf", EncodeOmniBuf, DecodeOmniBuf, math.MaxUint64},
		{"Fixed64", EncodeFixed64, DecodeFixed64, math.MaxUint64},
	}

	fmt.Println("\nZstd Compressed Sizes (10,000 uint64 values):")
	fmt.Printf("%-10s", "Codec")
	for _, name := range selectedDatasets {
		fmt.Printf(" | %18s", name+" (raw->zstd)")
	}
	fmt.Println()
	for _, c := range selectedCodecs {
		fmt.Printf("%-10s", c.name)
		for _, name := range selectedDatasets {
			for _, ds := range datasets {
				if ds.name == name {
					raw := c.encode(nil, ds.vals)
					comp := enc.EncodeAll(raw, nil)
					fmt.Printf(" | %7d -> %7d", len(raw), len(comp))
				}
			}
		}
		fmt.Println()
	}
}

func BenchmarkEncode(b *testing.B) {
	datasets := makeDatasets()
	buf := make([]byte, 0, benchCount*10)
	for _, ds := range datasets {
		for _, c := range allCodecs {
			if ds.maxVal > c.maxVal {
				continue
			}
			b.Run(ds.name+"/"+c.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					buf = c.encode(buf[:0], ds.vals)
				}
			})
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	datasets := makeDatasets()
	dst := make([]uint64, benchCount)
	for _, ds := range datasets {
		for _, c := range allCodecs {
			if ds.maxVal > c.maxVal {
				continue
			}
			encoded := c.encode(nil, ds.vals)
			b.Run(ds.name+"/"+c.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = c.decode(encoded, dst)
				}
			})
		}
	}
}

func BenchmarkZstdDecompressAndDecode(b *testing.B) {
	enc, _ := zstd.NewWriter(nil)
	defer enc.Close()
	dec, _ := zstd.NewReader(nil)
	defer dec.Close()

	datasets := makeDatasets()
	var skewed []uint64
	for _, ds := range datasets {
		if ds.name == "Skewed" {
			skewed = ds.vals
		}
	}
	dst := make([]uint64, benchCount)
	tmp := make([]byte, 0, benchCount*10)

	cases := []struct {
		name   string
		encode func([]byte, []uint64) []byte
		decode func([]byte, []uint64) int
	}{
		{"LEB128Protowire", EncodeLEB128Protowire, DecodeLEB128Protowire},
		{"OmniBuf", EncodeOmniBuf, DecodeOmniBuf},
		{"Fixed64Unsafe", EncodeFixed64Unsafe, DecodeFixed64Unsafe},
	}

	for _, tc := range cases {
		comp := enc.EncodeAll(tc.encode(nil, skewed), nil)
		b.Run("Skewed/"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var err error
				tmp, err = dec.DecodeAll(comp, tmp[:0])
				if err != nil {
					b.Fatal(err)
				}
				_ = tc.decode(tmp, dst)
			}
		})
	}
}

func BenchmarkZstdEncodeAndCompress(b *testing.B) {
	enc, _ := zstd.NewWriter(nil)
	defer enc.Close()

	datasets := makeDatasets()
	var skewed []uint64
	for _, ds := range datasets {
		if ds.name == "Skewed" {
			skewed = ds.vals
		}
	}
	rawBuf := make([]byte, 0, benchCount*10)
	compBuf := make([]byte, 0, benchCount*10)

	cases := []struct {
		name   string
		encode func([]byte, []uint64) []byte
	}{
		{"LEB128Protowire", EncodeLEB128Protowire},
		{"OmniBuf", EncodeOmniBuf},
		{"Fixed64Unsafe", EncodeFixed64Unsafe},
	}

	for _, tc := range cases {
		b.Run("Skewed/"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rawBuf = tc.encode(rawBuf[:0], skewed)
				compBuf = enc.EncodeAll(rawBuf, compBuf[:0])
			}
		})
	}
}
