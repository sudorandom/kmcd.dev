package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	WIDTH  = 1200
	HEIGHT = 630
)

type Point struct {
	X, Y float64
}

type PostMeta struct {
	Title      string
	Slug       string
	Date       string
	Series     []string
	Tags       []string
	Categories []string
}

type Palette struct {
	Name    string
	Bg      string
	Surface string
	Border  string
	Shadow1 string
	Shadow2 string
	Inks    []string // 4 spot inks
}

// Palettes rooted in kmcd.dev's Midnight Ocean Riso & Spot-Ink system
var Palettes = []Palette{
	{
		Name:    "midnight-ocean-triad",
		Bg:      "#0d121f",
		Surface: "#141c2e",
		Border:  "#2b3b5e",
		Shadow1: "#070a12",
		Shadow2: "#163b4a",
		Inks:    []string{"#2dd4bf", "#a78bfa", "#38bdf8", "#fb7185"}, // Teal, Violet, Sky, Coral
	},
	{
		Name:    "riso-cyber-sunset",
		Bg:      "#0e1124",
		Surface: "#171c36",
		Border:  "#313b6e",
		Shadow1: "#080914",
		Shadow2: "#2b2450",
		Inks:    []string{"#ff5449", "#ffbe0b", "#2dd4bf", "#a78bfa"}, // Riso Coral, Sunflower, Teal, Violet
	},
	{
		Name:    "deep-sea-telemetry",
		Bg:      "#0a131f",
		Surface: "#112033",
		Border:  "#234063",
		Shadow1: "#050a10",
		Shadow2: "#0f3a42",
		Inks:    []string{"#38bdf8", "#2dd4bf", "#818cf8", "#fbbf24"}, // Sky, Teal, Indigo, Amber
	},
	{
		Name:    "indigo-press",
		Bg:      "#111122",
		Surface: "#1a1a35",
		Border:  "#353566",
		Shadow1: "#080812",
		Shadow2: "#2d1b4e",
		Inks:    []string{"#c084fc", "#38bdf8", "#f472b6", "#2dd4bf"}, // Purple, Sky, Pink, Teal
	},
	{
		Name:    "emerald-packet",
		Bg:      "#0b1518",
		Surface: "#122429",
		Border:  "#244752",
		Shadow1: "#050b0d",
		Shadow2: "#1b3329",
		Inks:    []string{"#34d399", "#38bdf8", "#fbbf24", "#a78bfa"}, // Emerald, Sky, Sunflower, Violet
	},
}

func main() {
	style := flag.String("style", "auto", "The generation style: 'auto', 'constellation', 'flow', 'orbital', 'bauhaus', 'isometric', 'random'.")
	seed := flag.Int64("seed", 0, "Random seed. If 0, derived deterministically from post path or random.")
	network := flag.Bool("network", true, "Add network graph overlay.")
	output := flag.String("o", "cover.svg", "Output SVG file path or post directory.")
	pngOutput := flag.String("png", "", "Optional flag kept for CLI compatibility.")
	numShapes := flag.Int("n", 100, "Complexity parameter (kept for CLI compatibility).")
	mode := flag.Int("m", 0, "Palette override (kept for CLI compatibility).")
	batch := flag.Bool("batch-regenerate", false, "Regenerate all generated cover.svg files across content/posts.")

	flag.Parse()
	_ = pngOutput
	_ = numShapes

	if *batch {
		regenerateAllGeneratedCovers()
		return
	}

	finalOutputPath, postDir := resolvePaths(*output)
	meta := parsePostMeta(postDir)

	actualSeed := *seed
	if actualSeed == 0 {
		if meta.Slug != "" || postDir != "" {
			key := meta.Slug
			if key == "" {
				key = filepath.Base(postDir)
			}
			actualSeed = hashSeed(key)
		} else {
			actualSeed = rand.Int63()
		}
	}

	chosenStyle := resolveStyle(*style, meta, actualSeed)
	fmt.Printf("Generating cover: style=%s seed=%d path=%s\n", chosenStyle, actualSeed, finalOutputPath)

	svgContent := generateArtSVG(chosenStyle, actualSeed, meta, *network, *mode)

	if err := os.WriteFile(finalOutputPath, []byte(svgContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save SVG: %v\n", err)
		os.Exit(1)
	}
}

func resolvePaths(out string) (svgPath string, postDir string) {
	info, err := os.Stat(out)
	if err == nil && info.IsDir() {
		return filepath.Join(out, "cover.svg"), out
	}
	if strings.HasSuffix(out, ".svg") {
		return out, filepath.Dir(out)
	}
	return out, filepath.Dir(out)
}

func hashSeed(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte("kmcd-riso-art-v3:" + s))
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, h.Sum64())
	v := int64(binary.LittleEndian.Uint64(b) & 0x7fffffffffffffff)
	if v == 0 {
		v = 42
	}
	return v
}

func parsePostMeta(dir string) PostMeta {
	meta := PostMeta{}
	if dir == "" || dir == "." {
		return meta
	}
	meta.Slug = filepath.Base(dir)
	indexPath := filepath.Join(dir, "index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return meta
	}
	content := string(data)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return meta
	}
	fm := parts[1]

	if m := regexp.MustCompile(`(?m)^title:\s*["']?([^"'\n]+)["']?`).FindStringSubmatch(fm); len(m) > 1 {
		meta.Title = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`(?m)^slug:\s*["']?([^"'\n]+)["']?`).FindStringSubmatch(fm); len(m) > 1 {
		meta.Slug = strings.TrimSpace(m[1])
	}
	meta.Tags = extractList(fm, "tags")
	meta.Series = extractList(fm, "series")
	meta.Categories = extractList(fm, "categories")
	return meta
}

func extractList(fm, key string) []string {
	var res []string
	re := regexp.MustCompile(`(?m)^` + key + `:\s*\[([^\]]*)\]`)
	if m := re.FindStringSubmatch(fm); len(m) > 1 {
		items := strings.Split(m[1], ",")
		for _, it := range items {
			clean := strings.Trim(strings.TrimSpace(it), `"'`)
			if clean != "" {
				res = append(res, strings.ToLower(clean))
			}
		}
	}
	return res
}

func resolveStyle(requested string, meta PostMeta, seed int64) string {
	styles := []string{"constellation", "flow", "orbital", "bauhaus", "isometric"}
	switch requested {
	case "constellation", "flow", "orbital", "bauhaus", "isometric":
		return requested
	case "grid":
		return "isometric"
	case "radial":
		return "orbital"
	}

	rnd := rand.New(rand.NewSource(seed))
	return styles[rnd.Intn(len(styles))]
}

func generateArtSVG(style string, seed int64, meta PostMeta, addNetwork bool, modeOverride int) string {
	rnd := rand.New(rand.NewSource(seed))

	palIdx := int(math.Abs(float64(seed/7))) % len(Palettes)
	if modeOverride > 0 {
		palIdx = modeOverride % len(Palettes)
	}
	pal := Palettes[palIdx]

	// Shuffle a working copy of the 4 spot inks so each seed gets a unique primary/secondary balance
	inks := make([]string, len(pal.Inks))
	copy(inks, pal.Inks)
	rnd.Shuffle(len(inks), func(i, j int) { inks[i], inks[j] = inks[j], inks[i] })

	var buf bytes.Buffer
	w := func(format string, args ...any) {
		fmt.Fprintf(&buf, format+"\n", args...)
	}

	w(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="100%%" height="100%%">`, WIDTH, HEIGHT)
	w(`<defs>`)
	// Subtle Risograph paper dot grain
	w(`  <pattern id="bg-dots" width="24" height="24" patternUnits="userSpaceOnUse">`)
	w(`    <circle cx="6" cy="6" r="1.1" fill="%s" fill-opacity="0.13"/>`, inks[0])
	w(`    <circle cx="18" cy="18" r="1.1" fill="%s" fill-opacity="0.11"/>`, inks[1])
	w(`  </pattern>`)
	// Coarse Halftone dot fills for geometric layers
	for i := 0; i < 4; i++ {
		angle := -20 + i*15
		w(`  <pattern id="ht-%d" width="10" height="10" patternUnits="userSpaceOnUse" patternTransform="rotate(%d)">`, i, angle)
		w(`    <circle cx="5" cy="5" r="2.6" fill="%s" fill-opacity="0.55"/>`, inks[i])
		w(`  </pattern>`)
		w(`  <pattern id="lines-%d" width="8" height="8" patternUnits="userSpaceOnUse" patternTransform="rotate(%d)">`, i, 45-i*30)
		w(`    <line x1="0" y1="0" x2="0" y2="8" stroke="%s" stroke-width="2.2" stroke-opacity="0.45"/>`, inks[i])
		w(`  </pattern>`)
	}
	// Radial vignette gradient so edges feel deep and grounded
	w(`  <radialGradient id="vignette" cx="50%%" cy="50%%" r="65%%">`)
	w(`    <stop offset="55%%" stop-color="%s" stop-opacity="0"/>`, pal.Bg)
	w(`    <stop offset="100%%" stop-color="#04060c" stop-opacity="0.55"/>`)
	w(`  </radialGradient>`)
	w(`</defs>`)

	// 1. Base Canvas + Subtle Grid + Halftone Grain
	w(`<rect width="%d" height="%d" fill="%s"/>`, WIDTH, HEIGHT, pal.Bg)
	drawSubtleGrid(&buf, pal)
	w(`<rect width="%d" height="%d" fill="url(#bg-dots)"/>`, WIDTH, HEIGHT)

	// 2. Ambient Halftone Geometry (soft geometric fields behind main composition)
	drawAmbientHalftones(&buf, rnd, inks)

	// 3. Main Generative Art Composition
	switch style {
	case "constellation":
		drawConstellationArt(&buf, rnd, pal, inks)
	case "flow":
		drawFlowArt(&buf, rnd, pal, inks)
	case "orbital":
		drawOrbitalArt(&buf, rnd, pal, inks)
	case "isometric":
		drawIsometricArt(&buf, rnd, pal, inks)
	default: // "bauhaus"
		drawBauhausArt(&buf, rnd, pal, inks)
	}

	// 4. Soft Vignette
	w(`<rect width="%d" height="%d" fill="url(#vignette)" pointer-events="none"/>`, WIDTH, HEIGHT)
	w(`</svg>`)
	return buf.String()
}

func drawSubtleGrid(buf *bytes.Buffer, pal Palette) {
	fmt.Fprintf(buf, `<g opacity="0.22">`+"\n")
	step := 60
	for x := step; x < WIDTH; x += step {
		fmt.Fprintf(buf, `  <line x1="%d" y1="0" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`+"\n", x, x, HEIGHT, pal.Border)
	}
	for y := step; y < HEIGHT; y += step {
		fmt.Fprintf(buf, `  <line x1="0" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`+"\n", y, WIDTH, y, pal.Border)
	}
	fmt.Fprintf(buf, `</g>`+"\n")
}

func drawAmbientHalftones(buf *bytes.Buffer, rnd *rand.Rand, inks []string) {
	fmt.Fprintf(buf, `<g class="ambient-halftones">`+"\n")
	count := 3 + rnd.Intn(2)
	for i := 0; i < count; i++ {
		cx := 140.0 + rnd.Float64()*float64(WIDTH-280)
		cy := 140.0 + rnd.Float64()*float64(HEIGHT-280)
		r := 110.0 + rnd.Float64()*140.0
		pat := fmt.Sprintf("url(#ht-%d)", i%4)
		if i%2 == 1 {
			pat = fmt.Sprintf("url(#lines-%d)", i%4)
		}
		if rnd.Float64() < 0.65 {
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", cx, cy, r, pat)
		} else {
			ang := rnd.Float64() * 360.0
			fmt.Fprintf(buf, `  <g transform="translate(%.1f %.1f) rotate(%.1f)">`+"\n", cx, cy, ang)
			drawPolygonPath(buf, 6, 0, 0, r, pat, "", 0)
			fmt.Fprintf(buf, `  </g>`+"\n")
		}
	}
	fmt.Fprintf(buf, `</g>`+"\n")
}

// Style 1: Constellation & Geodesic Network Art (Submarine cable arcs, Delaunay-like triangles, glowing nodes)
func drawConstellationArt(buf *bytes.Buffer, rnd *rand.Rand, pal Palette, inks []string) {
	fmt.Fprintf(buf, `<g class="art-constellation">`+"\n")

	// Generate 22 well-spaced points across the canvas, concentrated toward the middle horizontal band (y=150..480)
	var pts []Point
	cols, rows := 7, 4
	cellW := float64(WIDTH-120) / float64(cols)
	cellH := float64(HEIGHT-100) / float64(rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if rnd.Float64() < 0.82 {
				px := 60.0 + (float64(c)+0.15+rnd.Float64()*0.7)*cellW
				py := 50.0 + (float64(r)+0.15+rnd.Float64()*0.7)*cellH
				pts = append(pts, Point{X: px, Y: py})
			}
		}
	}

	// 1. Translucent geometric facets (triangles between close neighbors)
	for i := 0; i < len(pts); i++ {
		// find 2 nearest neighbors
		type nb struct {
			idx  int
			dist float64
		}
		var nbs []nb
		for j := 0; j < len(pts); j++ {
			if i == j {
				continue
			}
			d := math.Hypot(pts[i].X-pts[j].X, pts[i].Y-pts[j].Y)
			if d < 320 {
				nbs = append(nbs, nb{j, d})
			}
		}
		sort.Slice(nbs, func(a, b int) bool { return nbs[a].dist < nbs[b].dist })
		if len(nbs) >= 2 && rnd.Float64() < 0.65 {
			p1, p2, p3 := pts[i], pts[nbs[0].idx], pts[nbs[1].idx]
			ink := inks[i%len(inks)]
			op := 0.08 + rnd.Float64()*0.16
			fmt.Fprintf(buf, `  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" fill-opacity="%.2f" stroke="%s" stroke-width="1.5" stroke-opacity="0.35"/>`+"\n",
				p1.X, p1.Y, p2.X, p2.Y, p3.X, p3.Y, ink, op, ink)
		}
	}

	// 2. Sweeping great-circle submarine arcs & crisp geodesic edges
	for i := 0; i < len(pts); i++ {
		for j := i + 1; j < len(pts); j++ {
			d := math.Hypot(pts[i].X-pts[j].X, pts[i].Y-pts[j].Y)
			if d > 90 && d < 360 && rnd.Float64() < 0.45 {
				p1, p2 := pts[i], pts[j]
				ink := inks[(i+j)%len(inks)]
				if (i+j)%3 == 0 {
					// Sweeping bezier arc (like map.kmcd.dev cable arcs)
					midX, midY := (p1.X+p2.X)/2, (p1.Y+p2.Y)/2
					dx, dy := p2.X-p1.X, p2.Y-p1.Y
					bend := (rnd.Float64() - 0.5) * 0.65
					cx, cy := midX-dy*bend, midY+dx*bend
					w := 2.0 + rnd.Float64()*2.5
					fmt.Fprintf(buf, `  <path d="M %.1f %.1f Q %.1f %.1f %.1f %.1f" stroke="%s" stroke-width="%.1f" stroke-opacity="0.8" fill="none" stroke-linecap="round"/>`+"\n",
						p1.X, p1.Y, cx, cy, p2.X, p2.Y, ink, w)
				} else {
					fmt.Fprintf(buf, `  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1.8" stroke-opacity="0.55"/>`+"\n",
						p1.X, p1.Y, p2.X, p2.Y, ink)
				}
			}
		}
	}

	// 3. Nodes: Layered Risograph rings, diamonds, and glowing cores
	for i, pt := range pts {
		ink := inks[i%len(inks)]
		ink2 := inks[(i+1)%len(inks)]
		r := 6.0 + rnd.Float64()*16.0

		switch i % 4 {
		case 0: // Concentric Ring Hub with Risograph misregistration offset
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="2" stroke-opacity="0.45"/>`+"\n",
				pt.X, pt.Y, r*2.1, ink)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="0.85"/>`+"\n",
				pt.X+3.5, pt.Y+3.5, r, pal.Shadow1)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n",
				pt.X, pt.Y, r, ink)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n",
				pt.X, pt.Y, r*0.42, pal.Bg)
		case 1: // Rotated Diamond Node (IXP marker style)
			s := r * 1.25
			fmt.Fprintf(buf, `  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
				pt.X+3, pt.Y-s+3, pt.X+s+3, pt.Y+3, pt.X+3, pt.Y+s+3, pt.X-s+3, pt.Y+3, pal.Shadow1)
			fmt.Fprintf(buf, `  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" stroke="%s" stroke-width="2"/>`+"\n",
				pt.X, pt.Y-s, pt.X+s, pt.Y, pt.X, pt.Y+s, pt.X-s, pt.Y, ink, pal.Bg)
		case 2: // Misregistered Dual-Ink Riso Circle
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="0.75"/>`+"\n",
				pt.X+4, pt.Y-3, r, ink2)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="0.85"/>`+"\n",
				pt.X, pt.Y, r, ink)
		default: // Crisp Satellite Dot
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n",
				pt.X, pt.Y, r*0.65, ink)
		}
	}

	fmt.Fprintf(buf, `</g>`+"\n")
}

// Style 2: Harmonic Flow Fields & Multiplexed Ribbon Waves
func drawFlowArt(buf *bytes.Buffer, rnd *rand.Rand, pal Palette, inks []string) {
	fmt.Fprintf(buf, `<g class="art-flow">`+"\n")

	// Draw 18 flowing harmonic ribbons sweeping across the canvas
	numStreams := 18
	basePhase := rnd.Float64() * math.Pi * 2
	freq1 := 0.0035 + rnd.Float64()*0.0025
	freq2 := 0.0075 + rnd.Float64()*0.0035

	for s := 0; s < numStreams; s++ {
		t := float64(s) / float64(numStreams-1)
		startY := 70.0 + t*float64(HEIGHT-140)
		ink := inks[s%len(inks)]
		amp1 := 45.0 + math.Sin(t*math.Pi)*65.0
		amp2 := 20.0 + math.Cos(t*math.Pi*2)*25.0
		phaseOffset := float64(s) * 0.22

		var pts []string
		for x := -20.0; x <= float64(WIDTH+20); x += 20.0 {
			y := startY +
				math.Sin(x*freq1+basePhase+phaseOffset)*amp1 +
				math.Cos(x*freq2-basePhase+phaseOffset*1.3)*amp2
			cmd := "L"
			if x == -20.0 {
				cmd = "M"
			}
			pts = append(pts, fmt.Sprintf("%s %.1f %.1f", cmd, x, y))
		}
		d := strings.Join(pts, " ")

		strokeW := 2.0 + math.Sin(t*math.Pi)*4.5
		op := 0.45 + math.Sin(t*math.Pi)*0.45

		if s%4 == 0 {
			// Bold Riso shadow + main ribbon
			fmt.Fprintf(buf, `  <path d="%s" stroke="%s" stroke-width="%.1f" fill="none" stroke-linecap="round" transform="translate(5 5)"/>`+"\n",
				d, pal.Shadow1, strokeW+1.5)
			fmt.Fprintf(buf, `  <path d="%s" stroke="%s" stroke-width="%.1f" stroke-opacity="%.2f" fill="none" stroke-linecap="round"/>`+"\n",
				d, ink, strokeW+1.5, op)
		} else if s%3 == 0 {
			fmt.Fprintf(buf, `  <path d="%s" stroke="%s" stroke-width="%.1f" stroke-opacity="%.2f" stroke-dasharray="14 8" fill="none" stroke-linecap="round"/>`+"\n",
				d, ink, strokeW, op)
		} else {
			fmt.Fprintf(buf, `  <path d="%s" stroke="%s" stroke-width="%.1f" stroke-opacity="%.2f" fill="none" stroke-linecap="round"/>`+"\n",
				d, ink, strokeW, op)
		}

		// Place geometric nodes / spheres riding along key streams
		if s%3 == 1 {
			for x := 160.0 + float64((s*73)%180); x < float64(WIDTH-140); x += 260.0 {
				y := startY +
					math.Sin(x*freq1+basePhase+phaseOffset)*amp1 +
					math.Cos(x*freq2-basePhase+phaseOffset*1.3)*amp2
				r := 8.0 + rnd.Float64()*14.0
				fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", x+4, y+4, r, pal.Shadow1)
				fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="%s" stroke-width="2.5"/>`+"\n", x, y, r, ink, pal.Bg)
			}
		}
	}

	// Overlay 3 large translucent geometric discs intersecting the wave field
	for i := 0; i < 3; i++ {
		cx := 260.0 + float64(i)*340.0 + (rnd.Float64()-0.5)*90.0
		cy := 315.0 + (rnd.Float64()-0.5)*110.0
		r := 45.0 + rnd.Float64()*45.0
		ink := inks[(i+1)%len(inks)]
		fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="0.18" stroke="%s" stroke-width="3"/>`+"\n",
			cx, cy, r, ink, ink)
		fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n",
			cx, cy, r*0.32, inks[(i+2)%len(inks)])
	}

	fmt.Fprintf(buf, `</g>`+"\n")
}

// Style 3: Orbital & Radial Frequency Rings (Concentric arcs, radar sweeps, planetary nodes)
func drawOrbitalArt(buf *bytes.Buffer, rnd *rand.Rand, pal Palette, inks []string) {
	fmt.Fprintf(buf, `<g class="art-orbital">`+"\n")

	// 2 or 3 orbital centers distributed horizontally across the middle band (so both full and 110px strip look rich)
	centers := []Point{
		{X: 340.0 + (rnd.Float64()-0.5)*100.0, Y: 315.0 + (rnd.Float64()-0.5)*70.0},
		{X: 860.0 + (rnd.Float64()-0.5)*100.0, Y: 315.0 + (rnd.Float64()-0.5)*70.0},
	}
	if rnd.Float64() < 0.65 {
		centers = []Point{
			{X: 260.0 + (rnd.Float64()-0.5)*60.0, Y: 315.0 + (rnd.Float64()-0.5)*80.0},
			{X: 600.0 + (rnd.Float64()-0.5)*70.0, Y: 315.0 + (rnd.Float64()-0.5)*50.0},
			{X: 940.0 + (rnd.Float64()-0.5)*60.0, Y: 315.0 + (rnd.Float64()-0.5)*80.0},
		}
	}

	// Tangential connecting rays between centers
	for i := 0; i < len(centers)-1; i++ {
		c1, c2 := centers[i], centers[i+1]
		fmt.Fprintf(buf, `  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="2.5" stroke-dasharray="8 8" stroke-opacity="0.6"/>`+"\n",
			c1.X, c1.Y, c2.X, c2.Y, inks[i%len(inks)])
	}

	for cIdx, c := range centers {
		numRings := 5 + rnd.Intn(3)
		for rIdx := numRings; rIdx >= 1; rIdx-- {
			radius := float64(rIdx) * (28.0 + rnd.Float64()*8.0)
			ink := inks[(cIdx+rIdx)%len(inks)]

			// Full thin orbit track
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="1.2" stroke-opacity="0.28"/>`+"\n",
				c.X, c.Y, radius, ink)

			// Bold partial arc segment
			startAng := rnd.Float64() * 360.0
			sweepAng := 65.0 + rnd.Float64()*160.0
			strokeW := 4.0 + float64(rIdx%3)*3.0
			drawArcSegment(buf, c.X+4, c.Y+4, radius, startAng, startAng+sweepAng, pal.Shadow1, strokeW)
			drawArcSegment(buf, c.X, c.Y, radius, startAng, startAng+sweepAng, ink, strokeW)

			// Orbiting satellite node on the ring
			satAng := (startAng + sweepAng*0.5) * math.Pi / 180.0
			sx := c.X + radius*math.Cos(satAng)
			sy := c.Y + radius*math.Sin(satAng)
			satR := 5.0 + float64(rIdx%3)*3.0
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" stroke="%s" stroke-width="2"/>`+"\n",
				sx, sy, satR, inks[(cIdx+rIdx+1)%len(inks)], pal.Bg)
		}

		// Core layered sun/hub
		coreR := 24.0 + rnd.Float64()*14.0
		fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", c.X+5, c.Y+5, coreR, pal.Shadow1)
		fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", c.X, c.Y, coreR, inks[cIdx%len(inks)])
		fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", c.X, c.Y, coreR*0.45, pal.Bg)
	}

	fmt.Fprintf(buf, `</g>`+"\n")
}

// Style 4: Abstract Bauhaus & Risograph Print Composition (Layered geometric shapes, misregistered spot inks, curves)
func drawBauhausArt(buf *bytes.Buffer, rnd *rand.Rand, pal Palette, inks []string) {
	fmt.Fprintf(buf, `<g class="art-bauhaus">`+"\n")

	// Diagonal & orthogonal structural lines across canvas
	numLines := 4 + rnd.Intn(3)
	for i := 0; i < numLines; i++ {
		x1 := 80.0 + rnd.Float64()*float64(WIDTH-160)
		y1 := 60.0 + rnd.Float64()*float64(HEIGHT-120)
		ang := float64(rnd.Intn(8)) * (math.Pi / 4.0)
		length := 240.0 + rnd.Float64()*380.0
		x2 := x1 + math.Cos(ang)*length
		y2 := y1 + math.Sin(ang)*length
		ink := inks[i%len(inks)]
		fmt.Fprintf(buf, `  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="3.5" stroke-opacity="0.55" stroke-linecap="round"/>`+"\n",
			x1, y1, x2, y2, ink)
	}

	// Generate 8-10 rich geometric forms distributed across the canvas
	numShapes := 8 + rnd.Intn(4)
	for i := 0; i < numShapes; i++ {
		// Distribute X evenly across canvas with jitter, and bias Y toward the center band (140..490)
		slotW := float64(WIDTH-220) / float64(numShapes)
		px := 110.0 + (float64(i)+0.5)*slotW + (rnd.Float64()-0.5)*slotW*0.8
		py := 150.0 + rnd.Float64()*float64(HEIGHT-300)
		size := 65.0 + rnd.Float64()*95.0
		ink := inks[i%len(inks)]
		ink2 := inks[(i+1)%len(inks)]

		switch i % 5 {
		case 0: // Misregistered Risograph Circle + Halftone Overlap
			r := size * 0.6
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", px+7, py+7, r, pal.Shadow1)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s" fill-opacity="0.88"/>`+"\n", px, py, r, ink)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="url(#ht-%d)"/>`+"\n", px+r*0.35, py-r*0.25, r*0.75, (i+1)%4)

		case 1: // Semicircle / Bauhaus Arch Pair
			r := size * 0.65
			rot := float64(rnd.Intn(4)*90) + (rnd.Float64()-0.5)*25.0
			fmt.Fprintf(buf, `  <g transform="translate(%.1f %.1f) rotate(%.1f)">`+"\n", px, py, rot)
			fmt.Fprintf(buf, `    <path d="M %.1f 0 A %.1f %.1f 0 0 1 %.1f 0 Z" fill="%s" transform="translate(6 6)"/>`+"\n", -r, r, r, r, pal.Shadow1)
			fmt.Fprintf(buf, `    <path d="M %.1f 0 A %.1f %.1f 0 0 1 %.1f 0 Z" fill="%s" fill-opacity="0.9"/>`+"\n", -r, r, r, r, ink)
			fmt.Fprintf(buf, `    <circle cx="0" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", r*0.35, r*0.32, ink2)
			fmt.Fprintf(buf, `  </g>`+"\n")

		case 2: // Regular Polygon (Hexagon or Triangle) with Inner Cutout
			sides := 6
			if i%2 == 0 {
				sides = 3
			}
			rot := rnd.Float64() * 360.0
			fmt.Fprintf(buf, `  <g transform="translate(%.1f %.1f) rotate(%.1f)">`+"\n", px, py, rot)
			drawPolygonPath(buf, sides, 6, 6, size*0.65, pal.Shadow1, "", 0)
			drawPolygonPath(buf, sides, 0, 0, size*0.65, ink, "", 0)
			drawPolygonPath(buf, sides, 0, 0, size*0.32, pal.Bg, ink2, 2.5)
			fmt.Fprintf(buf, `  </g>`+"\n")

		case 3: // Bold Concentric Ring + Crosshair Star
			r := size * 0.55
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="14"/>`+"\n", px+5, py+5, r, pal.Shadow1)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="14" stroke-opacity="0.9"/>`+"\n", px, py, r, ink)
			drawStarPolygon(buf, 8, px, py, r*0.52, r*0.2, ink2)

		default: // Rotated Pill / Capsule with Striped Fill
			w, h := size*1.2, size*0.48
			rot := -35.0 + rnd.Float64()*70.0
			fmt.Fprintf(buf, `  <g transform="translate(%.1f %.1f) rotate(%.1f)">`+"\n", px, py, rot)
			fmt.Fprintf(buf, `    <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s"/>`+"\n", -w/2+6, -h/2+6, w, h, h/2, pal.Shadow1)
			fmt.Fprintf(buf, `    <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="%s" fill-opacity="0.85"/>`+"\n", -w/2, -h/2, w, h, h/2, ink)
			fmt.Fprintf(buf, `    <rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="%.1f" fill="url(#lines-%d)"/>`+"\n", -w/2, -h/2, w, h, h/2, i%4)
			fmt.Fprintf(buf, `  </g>`+"\n")
		}
	}

	fmt.Fprintf(buf, `</g>`+"\n")
}

// Style 5: Isometric Hexagonal Mesh & Signal Paths (3D geometric blocks, orthogonal circuit traces, nodes)
func drawIsometricArt(buf *bytes.Buffer, rnd *rand.Rand, pal Palette, inks []string) {
	fmt.Fprintf(buf, `<g class="art-isometric">`+"\n")

	// Isometric lattice constants
	radius := 48.0
	dx := radius * 1.5
	dy := radius * math.Sqrt(3)

	var hexCenters []Point
	for col := 1; col < int(float64(WIDTH)/dx)-1; col++ {
		for row := 1; row < int(float64(HEIGHT)/dy); row++ {
			cx := float64(col) * dx
			cy := float64(row) * dy
			if col%2 == 1 {
				cy += dy * 0.5
			}
			if cx > 100 && cx < WIDTH-100 && cy > 80 && cy < HEIGHT-80 {
				hexCenters = append(hexCenters, Point{X: cx, Y: cy})
			}
		}
	}

	// Draw isometric signal traces connecting random adjacent hex centers
	for i := 0; i < len(hexCenters); i++ {
		for j := i + 1; j < len(hexCenters); j++ {
			d := math.Hypot(hexCenters[i].X-hexCenters[j].X, hexCenters[i].Y-hexCenters[j].Y)
			if d < dy*1.15 && rnd.Float64() < 0.38 {
				ink := inks[(i+j)%len(inks)]
				fmt.Fprintf(buf, `  <line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="3" stroke-opacity="0.65" stroke-linecap="round"/>`+"\n",
					hexCenters[i].X, hexCenters[i].Y, hexCenters[j].X, hexCenters[j].Y, ink)
			}
		}
	}

	// Select subset of hex centers to render as 3-face isometric cubes or layered hexagons
	rnd.Shuffle(len(hexCenters), func(i, j int) { hexCenters[i], hexCenters[j] = hexCenters[j], hexCenters[i] })
	numCubes := min(len(hexCenters), 14+rnd.Intn(6))

	for i := 0; i < numCubes; i++ {
		pt := hexCenters[i]
		r := radius * (0.65 + rnd.Float64()*0.55)
		inkTop := inks[i%len(inks)]
		inkLeft := inks[(i+1)%len(inks)]
		inkRight := pal.Surface

		if i%3 == 0 {
			// Flat Hexagon with Halftone & Ring
			drawPolygonPath(buf, 6, pt.X+5, pt.Y+5, r, pal.Shadow1, "", 0)
			drawPolygonPath(buf, 6, pt.X, pt.Y, r, inkTop, pal.Bg, 2)
			fmt.Fprintf(buf, `  <circle cx="%.1f" cy="%.1f" r="%.1f" fill="%s"/>`+"\n", pt.X, pt.Y, r*0.35, pal.Bg)
		} else {
			// 3D Isometric Cube (Top, Left, Right rhombuses)
			drawIsometricCube(buf, pt.X+6, pt.Y+6, r, pal.Shadow1, pal.Shadow1, pal.Shadow1, "")
			drawIsometricCube(buf, pt.X, pt.Y, r, inkTop, inkLeft, inkRight, pal.Bg)
		}
	}

	fmt.Fprintf(buf, `</g>`+"\n")
}

func drawPolygonPath(buf *bytes.Buffer, sides int, cx, cy, r float64, fill, stroke string, strokeW float64) {
	var pts []string
	for i := 0; i < sides; i++ {
		ang := float64(i)*2.0*math.Pi/float64(sides) - math.Pi/2.0
		px := cx + r*math.Cos(ang)
		py := cy + r*math.Sin(ang)
		pts = append(pts, fmt.Sprintf("%.1f,%.1f", px, py))
	}
	st := ""
	if stroke != "" && strokeW > 0 {
		st = fmt.Sprintf(` stroke="%s" stroke-width="%.1f"`, stroke, strokeW)
	}
	fmt.Fprintf(buf, `    <polygon points="%s" fill="%s"%s/>`+"\n", strings.Join(pts, " "), fill, st)
}

func drawStarPolygon(buf *bytes.Buffer, points int, cx, cy, outerR, innerR float64, fill string) {
	var pts []string
	for i := 0; i < points*2; i++ {
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		ang := float64(i)*math.Pi/float64(points) - math.Pi/2.0
		px := cx + r*math.Cos(ang)
		py := cy + r*math.Sin(ang)
		pts = append(pts, fmt.Sprintf("%.1f,%.1f", px, py))
	}
	fmt.Fprintf(buf, `  <polygon points="%s" fill="%s"/>`+"\n", strings.Join(pts, " "), fill)
}

func drawArcSegment(buf *bytes.Buffer, cx, cy, r, startDeg, endDeg float64, stroke string, strokeW float64) {
	sRad := startDeg * math.Pi / 180.0
	eRad := endDeg * math.Pi / 180.0
	x1 := cx + r*math.Cos(sRad)
	y1 := cy + r*math.Sin(sRad)
	x2 := cx + r*math.Cos(eRad)
	y2 := cy + r*math.Sin(eRad)
	largeArc := 0
	if endDeg-startDeg > 180.0 {
		largeArc = 1
	}
	fmt.Fprintf(buf, `  <path d="M %.1f %.1f A %.1f %.1f 0 %d 1 %.1f %.1f" fill="none" stroke="%s" stroke-width="%.1f" stroke-linecap="round"/>`+"\n",
		x1, y1, r, r, largeArc, x2, y2, stroke, strokeW)
}

func drawIsometricCube(buf *bytes.Buffer, cx, cy, r float64, cTop, cLeft, cRight, stroke string) {
	// 6 vertices of a point-top hexagon
	var vx, vy [6]float64
	for i := 0; i < 6; i++ {
		ang := (float64(i)*60.0 - 90.0) * math.Pi / 180.0
		vx[i] = cx + r*math.Cos(ang)
		vy[i] = cy + r*math.Sin(ang)
	}
	st := ""
	if stroke != "" {
		st = fmt.Sprintf(` stroke="%s" stroke-width="1.8" stroke-linejoin="round"`, stroke)
	}
	// Top face: v5, v0, v1, center
	fmt.Fprintf(buf, `  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"%s/>`+"\n",
		vx[5], vy[5], vx[0], vy[0], vx[1], vy[1], cx, cy, cTop, st)
	// Right face: center, v1, v2, v3
	fmt.Fprintf(buf, `  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" fill-opacity="0.85"%s/>`+"\n",
		cx, cy, vx[1], vy[1], vx[2], vy[2], vx[3], vy[3], cRight, st)
	// Left face: center, v3, v4, v5
	fmt.Fprintf(buf, `  <polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" fill-opacity="0.75"%s/>`+"\n",
		cx, cy, vx[3], vy[3], vx[4], vy[4], vx[5], vy[5], cLeft, st)
}

func regenerateAllGeneratedCovers() {
	matches, err := filepath.Glob("content/posts/*/*/cover.svg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "glob error: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(matches)

	count := 0
	for _, svgPath := range matches {
		data, err := os.ReadFile(svgPath)
		if err != nil {
			continue
		}
		head := string(data)
		if len(head) > 3500 {
			head = head[:3500]
		}
		// Skip custom hand-crafted/exported maps like content/posts/2026/internet-map-2026/cover.svg
		if strings.Contains(head, "tiles-layer") || len(data) > 100000 {
			fmt.Printf("Skipping manually created SVG: %s\n", svgPath)
			continue
		}
		isGenerated := strings.Contains(head, `fill-opacity="0.501961"`) ||
			strings.Contains(head, `stroke-opacity="0.501961"`) ||
			strings.Contains(head, `id="riso-dots-bg"`) ||
			strings.Contains(head, `id="bg-dots"`) ||
			strings.Contains(head, `class="blueprint-grid"`)
		if !isGenerated {
			fmt.Printf("Skipping non-generated SVG: %s\n", svgPath)
			continue
		}

		postDir := filepath.Dir(svgPath)
		meta := parsePostMeta(postDir)
		seed := hashSeed(meta.Slug)
		style := resolveStyle("auto", meta, seed)
		svgContent := generateArtSVG(style, seed, meta, true, 0)
		if err := os.WriteFile(svgPath, []byte(svgContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", svgPath, err)
			continue
		}
		count++
		fmt.Printf("[%02d] Regenerated %-55s (style=%-13s seed=%d)\n", count, svgPath, style, seed)
	}
	fmt.Printf("Done! Regenerated %d generated cover.svg files.\n", count)
}
