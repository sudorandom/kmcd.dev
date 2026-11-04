package main

import (
	"strings"
	"testing"
)

func TestGenerateArtSVGStyles(t *testing.T) {
	styles := []string{"constellation", "flow", "orbital", "bauhaus", "isometric"}
	meta := PostMeta{
		Title: "HTTP/3 From Scratch",
		Slug:  "http3-from-scratch",
		Tags:  []string{"golang", "http3", "quic"},
	}

	for i, st := range styles {
		svg := generateArtSVG(st, int64(1000+i), meta, true, i)
		if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(strings.TrimSpace(svg), "</svg>") {
			t.Fatalf("style %s produced invalid SVG wrapper", st)
		}
	}
}

func BenchmarkGenerateArtSVG(b *testing.B) {
	meta := PostMeta{
		Title: "Benchmarking Protobuf",
		Slug:  "benchmarking-protojsonx",
		Tags:  []string{"protobuf", "golang"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateArtSVG("constellation", int64(i+1), meta, true, 0)
	}
}
