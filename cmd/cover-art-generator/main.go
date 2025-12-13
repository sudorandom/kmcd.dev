package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
	"github.com/fogleman/primitive/primitive"
)

// --- Configuration ---
const (
	WIDTH              = 1200
	HEIGHT             = 630
	GRID_SPACING       = 60
	MIN_WIRES          = 0
	MAX_WIRES          = 4
	MIN_COMPONENTS     = 7
	MAX_COMPONENTS     = 20
	COMPONENT_MIN_SIZE = 30
	COMPONENT_MAX_SIZE = 400
)

// --- Color Palettes ---
var PALETTES = [][]string{
	{"#d8f3dc", "#b7e4c7", "#95d5b2", "#74c69d", "#52b788", "#40916c", "#2d6a4f"}, // Green
	{"#fde2e4", "#fad2e1", "#fbc3d4", "#f9b4c8", "#f8a5bc", "#f796b0", "#f687a3"}, // Pink/Red

	{"#ADD8E6", "#87CEEB", "#6495ED", "#4169E1", "#1E90FF"}, // Blue
	{"#F2E7FE", "#E6CCFB", "#D1ACF6", "#BB8CEF", "#A36EE8"}, // Purple
	{"#FFFAD3", "#FFECB3", "#FFDD88", "#FFCE5A", "#FFBF2B"}, // Yellow/Orange
	{"#FFCDD2", "#EF9A9A", "#E57373", "#EF5350", "#F44336"}, // Red
	{"#F5F5F5", "#E0E0E0", "#BDBDBD", "#9E9E9E", "#757575"}, // Grey
	{"#D7CCC8", "#BCAAA4", "#A1887F", "#8D6E63", "#795548"}, // Brown
	{"#E0F7FA", "#B2EBF2", "#80DEEA", "#4DD0E1", "#00BCD4"}, // Cyan
	{"#C5CAE9", "#9FA8DA", "#7986CB", "#5C6BC0", "#3F51B5"}, // Indigo/Blue
	{"#A8DADC", "#83C5BE", "#6D9F9D", "#548A85", "#3C756F"}, // Teal

	// complementary palettes
	{"#F44336", "#00BCD4", "#EF5350", "#4DD0E1", "#E57373", "#80DEEA", "#EF9A9A", "#B2EBF2", "#FFCDD2", "#E0F7FA"}, // Red & Cyan (Complementary)
	{"#2d6a4f", "#FF0066", "#40916c", "#FF3399", "#52b788", "#FF66B2", "#74c69d", "#FF99CC", "#95d5b2", "#FFCCE5"}, // Green & Magenta (Complementary)
	{"#1E90FF", "#FFBF2B", "#4169E1", "#FFCE5A", "#6495ED", "#FFDD88", "#87CEEB", "#FFECB3", "#ADD8E6", "#FFFAD3"}, // Blue & Orange (Complementary)
	{"#FFBF2B", "#A36EE8", "#FFCE5A", "#BB8CEF", "#FFDD88", "#D1ACF6", "#FFECB3", "#E6CCFB", "#FFFAD3", "#F2E7FE"}, // Yellow & Purple (Complementary)
	{"#2d6a4f", "#A36EE8", "#40916c", "#BB8CEF", "#52b788", "#D1ACF6", "#74c69d", "#E6CCFB", "#95d5b2", "#F2E7FE"}, // Green & Purple (Complementary - using existing purple for violet)
	{"#3C756F", "#f687a3", "#548A85", "#f796b0", "#6D9F9D", "#f8a5bc", "#83C5BE", "#f9b4c8", "#A8DADC", "#fbc3d4"}, // Teal & Pink/Red (Complementary)
	{"#3F51B5", "#FFBF2B", "#5C6BC0", "#FFCE5A", "#7986CB", "#FFDD88", "#9FA8DA", "#FFECB3", "#C5CAE9", "#FFFAD3"}, // Indigo & Gold (Complementary)
	{"#795548", "#1E90FF", "#8D6E63", "#4169E1", "#A1887F", "#6495ED", "#BCAAA4", "#87CEEB", "#D7CCC8", "#ADD8E6"}, // Brown & Blue (Complementary)
	{"#757575", "#F44336", "#9E9E9E", "#EF5350", "#BDBDBD", "#E57373", "#E0E0E0", "#EF9A9A", "#F5F5F5", "#FFCDD2"}, // Grey & Red (Complementary)

	{"#FFBF2B", "#008080", "#FFCE5A", "#00AAAA", "#FFDD88", "#00DADA"}, // Orange & Teal (Complementary)
	{"#6A5ACD", "#FFA07A", "#8470FF", "#FFC48C", "#9370DB", "#FFDBA4"}, // SlateBlue & LightSalmon (Complementary)
	{"#FF69B4", "#3CB371", "#FF8DC8", "#5CD791", "#FFA2DC", "#7CEBB1"}, // HotPink & MediumSeaGreen (Complementary)
	{"#4682B4", "#D2B48C", "#6A9EC8", "#E6C9A4", "#8EBADA", "#FADEC0"}, // SteelBlue & Tan (Complementary)
	{"#00FFFF", "#FFD700", "#33FFFF", "#FFE47A", "#66FFFF", "#FFF1A4"}, // Aqua & Gold (Complementary)
	{"#7FFFD4", "#B0E0E6", "#90EE90", "#ADD8E6", "#C0FFC0", "#E0FFFF"}, // Aquamarine & PowderBlue (Analogous/Complementary)
	{"#FF4500", "#4169E1", "#FF6347", "#6A5ACD", "#FF7F50", "#8A2BE2"}, // OrangeRed & RoyalBlue (Complementary)
	{"#20B2AA", "#FF6347", "#3CB371", "#FF7F50", "#66CDAA", "#FFA07A"}, // LightSeaGreen & Tomato (Complementary)
	{"#8A2BE2", "#FFD700", "#9370DB", "#FFE47A", "#BA55D3", "#FFF1A4"}, // BlueViolet & Gold (Complementary)
	{"#FF8C00", "#4682B4", "#FFA07A", "#6A9EC8", "#FFB6C1", "#8EBADA"}, // DarkOrange & SteelBlue (Complementary)
	{"#00FA9A", "#FF69B4", "#3CB371", "#FF8DC8", "#66CDAA", "#FFA2DC"}, // MediumSpringGreen & HotPink (Complementary)
	{"#483D8B", "#F0E68C", "#6A5ACD", "#FFFACD", "#8470FF", "#FFFFE0"}, // DarkSlateBlue & Khaki (Complementary)
	{"#FFDAB9", "#6B8E23", "#FFE4B5", "#8FBC8F", "#FFEFD5", "#ADFF2F"}, // PeachPuff & OliveDrab (Complementary)
	{"#CD5C5C", "#4682B4", "#F08080", "#6A9EC8", "#E9967A", "#8EBADA"}, // IndianRed & SteelBlue (Complementary)
	{"#DAA520", "#87CEFA", "#BDB76B", "#ADD8E6", "#F0E68C", "#E0FFFF"}, // Goldenrod & LightSkyBlue (Complementary)
	{"#800080", "#ADFF2F", "#BA55D3", "#B2EE67", "#DDA0DD", "#CAFF70"}, // Purple & GreenYellow (Complementary)
	{"#FF6347", "#40E0D0", "#FF7F50", "#64DCDC", "#FFA07A", "#88EEEC"}, // Tomato & Turquoise (Complementary)
	{"#2F4F4F", "#FFDEAD", "#708090", "#FFE4C4", "#A9A9A9", "#FFEFD5"}, // DarkSlateGray & NavajoWhite (Complementary)
}

var backgroundColors = []string{"#111111", "#0D1B2A", "#1B263B", "#22223B", "#0A0A14", "#201E1F"}

type Point struct {
	X, Y float64
}

type Neighbor struct {
	Point    Point
	Distance float64
}

func main() {
	// Art generation flags
	style := flag.String("style", "grid", "The generation style. Choices: 'grid', 'radial', 'flow', 'random'.")
	seed := flag.Int64("seed", 0, "Random seed. If 0, a random seed is used.")
	network := flag.Bool("network", false, "Add a network graph overlay connecting components.")

	// Primitive flags
	output := flag.String("o", "cover.svg", "Output SVG file path.")
	pngOutput := flag.String("png", "", "Output PNG file path for the intermediate raster image.")
	numShapes := flag.Int("n", 100, "Number of shapes to use in the primitive output.")
	mode := flag.Int("m", 1, "Mode for primitive shape generation (0-8).")

	flag.Parse()

	if *seed == 0 {
		*seed = rand.Int63()
	}
	fmt.Printf("Using seed: %d\n", *seed)

	// 1. Generate the raster image in memory
	artImage := generateArt(*style, *seed, *network)

	// Save the intermediate PNG if requested
	if *pngOutput != "" {
		file, err := os.Create(*pngOutput)
		if err != nil {
			log.Fatalf("Failed to create PNG file: %v", err)
		}
		defer file.Close()
		if err := png.Encode(file, artImage); err != nil {
			log.Fatalf("Failed to save PNG: %v", err)
		}
		fmt.Printf("Intermediate art saved to %s\n", *pngOutput)
	}

	// 2. Convert to SVG using primitive
	svgContent, err := primitivize(artImage, *numShapes, *mode)
	if err != nil {
		log.Fatalf("Failed to generate SVG: %v", err)
	}

	// 3. Save the SVG file
	finalOutputPath := *output
	if info, err := os.Stat(finalOutputPath); err == nil && info.IsDir() {
		finalOutputPath = filepath.Join(finalOutputPath, "cover.svg")
	}

	err = os.WriteFile(finalOutputPath, []byte(svgContent), 0644)
	if err != nil {
		log.Fatalf("Failed to save SVG: %v", err)
	}

	fmt.Printf("Art saved to %s\n", finalOutputPath)
}

func clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func toShapeType(mode int) primitive.ShapeType {
	switch mode {
	case 1:
		return primitive.ShapeTypeTriangle
	case 2:
		return primitive.ShapeTypeRectangle
	case 3:
		return primitive.ShapeTypeEllipse
	case 4:
		return primitive.ShapeTypeCircle
	case 5:
		return primitive.ShapeTypeRotatedRectangle
	case 6:
		return primitive.ShapeTypeQuadratic
	case 7:
		return primitive.ShapeTypeRotatedEllipse
	case 8:
		return primitive.ShapeTypePolygon
	default:
		return primitive.ShapeTypeAny
	}
}

func primitivize(inputImage image.Image, numShapes int, mode int) (string, error) {
	// Use all available cores
	workers := runtime.GOMAXPROCS(0)
	mode = clamp(mode, 0, 8)
	shapeType := toShapeType(mode)

	bg := primitive.MakeColor(primitive.AverageImageColor(inputImage))
	model := primitive.NewModel(inputImage, bg, inputImage.Bounds().Dx(), workers)

	for i := 1; i <= numShapes; i++ {
		model.Step(shapeType, 128, 0)
	}

	svg := model.SVG()
	return svg, nil
}

func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex color length")
	}
	r, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	g, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	b, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, nil
}

func generateArt(style string, seed int64, addNetwork bool) image.Image {
	rnd := rand.New(rand.NewSource(seed))

	// 1. --- Setup ---
	bg_color := backgroundColors[rnd.Intn(len(backgroundColors))]
	dc := gg.NewContext(WIDTH, HEIGHT)
	dc.SetHexColor(bg_color)
	dc.Clear()

	// Draw background grid
	if bgColorRGB, err := parseHexColor(bg_color); err == nil {
		dc.SetRGBA(
			math.Min(1, float64(bgColorRGB.R)/255.0+0.08),
			math.Min(1, float64(bgColorRGB.G)/255.0+0.08),
			math.Min(1, float64(bgColorRGB.B)/255.0+0.08),
			1,
		)
		for x := 0; x < WIDTH; x += GRID_SPACING {
			dc.DrawLine(float64(x), 0, float64(x), float64(HEIGHT))
		}
		for y := 0; y < HEIGHT; y += GRID_SPACING {
			dc.DrawLine(0, float64(y), float64(WIDTH), float64(y))
		}
		dc.Stroke()
	}

	palette := PALETTES[rnd.Intn(len(PALETTES))]

	// 2. --- Create Grid & Points ---
	jitterAmount := 0.0
	if style != "grid" {
		jitterAmount = 5.0
	}
	var gridPoints []Point
	for x := 0; x <= WIDTH+GRID_SPACING; x += GRID_SPACING {
		for y := 0; y <= HEIGHT+GRID_SPACING; y += GRID_SPACING {
			jitterX := rnd.Float64()*2*jitterAmount - jitterAmount
			jitterY := rnd.Float64()*2*jitterAmount - jitterAmount
			gridPoints = append(gridPoints, Point{X: float64(x) + jitterX, Y: float64(y) + jitterY})
		}
	}

	numComponents := rnd.Intn(MAX_COMPONENTS-MIN_COMPONENTS+1) + MIN_COMPONENTS

	// Calculate base component size inversely proportional to the number of components
	// More components -> smaller size, Fewer components -> larger size
	numComponentsRange := float64(MAX_COMPONENTS - MIN_COMPONENTS)
	normalizedNumComponents := float64(numComponents-MIN_COMPONENTS) / numComponentsRange

	componentSizeRange := float64(COMPONENT_MAX_SIZE - COMPONENT_MIN_SIZE)
	// Invert the normalized value: 0 components (min_n) maps to 1 (max_s), max components (max_n) maps to 0 (min_s)
	inverseNormalizedNumComponents := math.Sqrt(1.0 - normalizedNumComponents)

	baseComponentSize := float64(COMPONENT_MIN_SIZE) + (inverseNormalizedNumComponents * componentSizeRange)

	var componentPoints []Point
	for i := 0; i < numComponents; i++ {
		componentPoints = append(componentPoints, gridPoints[rnd.Intn(len(gridPoints))])
	}

	// 3. --- Draw Wires based on Style ---
	numWires := rnd.Intn(MAX_WIRES-MIN_WIRES+1) + MIN_WIRES

	for range numWires {
		startPoint := gridPoints[rnd.Intn(len(gridPoints))]
		endPoint := gridPoints[rnd.Intn(len(gridPoints))]
		color := palette[rnd.Intn(len(palette))]
		width := float64(rnd.Intn(2) + 10)
		if width == 2 && rnd.Float64() < 0.5 { // more chance for width 1
			width = 1
		}

		dc.SetHexColor(color)
		dc.SetLineWidth(width)

		switch style {
		case "radial":
			hub := Point{
				X: float64(WIDTH)/2 + (rnd.Float64()-0.5)*(float64(WIDTH)/2),
				Y: float64(HEIGHT)/2 + (rnd.Float64()-0.5)*(float64(HEIGHT)/2),
			}
			startPoint = hub
			dc.DrawLine(startPoint.X, startPoint.Y, endPoint.X, endPoint.Y)
			dc.Stroke()
		case "flow":
			var flowPoints []Point
			for _, p := range gridPoints {
				if p.X < WIDTH/2 {
					flowPoints = append(flowPoints, p)
				}
			}
			if len(flowPoints) > 0 {
				startPoint = flowPoints[rnd.Intn(len(flowPoints))]
				var endPointsFiltered []Point
				for _, p := range gridPoints {
					if p.X > startPoint.X+GRID_SPACING {
						endPointsFiltered = append(endPointsFiltered, p)
					}
				}
				if len(endPointsFiltered) > 0 {
					endPoint = endPointsFiltered[rnd.Intn(len(endPointsFiltered))]
					dc.DrawLine(startPoint.X, startPoint.Y, endPoint.X, endPoint.Y)
					dc.Stroke()
				}
			}
		default: // grid or random
			if style == "grid" || rnd.Float64() < 0.2 {
				midPointX := Point{X: startPoint.X, Y: endPoint.Y}
				midPointY := Point{X: endPoint.X, Y: startPoint.Y}
				midPoint := midPointX
				if rnd.Float64() < 0.5 {
					midPoint = midPointY
				}
				dc.MoveTo(startPoint.X, startPoint.Y)
				dc.LineTo(midPoint.X, midPoint.Y)
				dc.LineTo(endPoint.X, endPoint.Y)
				dc.Stroke()
			} else {
				dc.DrawLine(startPoint.X, startPoint.Y, endPoint.X, endPoint.Y)
				dc.Stroke()
			}
		}
	}
	dc.Fill()

	// 4. --- Draw Network Overlay ---
	if addNetwork && len(componentPoints) > 1 {
		for _, p1 := range componentPoints {
			var neighbors []Neighbor
			for _, p2 := range componentPoints {
				if p1.X == p2.X && p1.Y == p2.Y {
					continue
				}
				dist := math.Hypot(p1.X-p2.X, p1.Y-p2.Y)
				neighbors = append(neighbors, Neighbor{Point: p2, Distance: dist})
			}

			sort.Slice(neighbors, func(i, j int) bool {
				return neighbors[i].Distance < neighbors[j].Distance
			})

			numNeighbors := rnd.Intn(5) // 0 to 4
			if len(neighbors) < numNeighbors {
				numNeighbors = len(neighbors)
			}

			for i := 0; i < numNeighbors; i++ {
				neighbor := neighbors[i]
				if neighbor.Distance < WIDTH/3.5 {
					dc.SetHexColor(palette[rnd.Intn(len(palette))])
					dc.SetLineWidth(float64(rnd.Intn(8) + 8))
					dc.DrawLine(p1.X, p1.Y, neighbor.Point.X, neighbor.Point.Y)
					dc.Stroke()
				}
			}
		}
	}
	dc.Fill()

	// 5. --- Draw Components ---
	for _, point := range componentPoints {
		// Apply a random jitter around the base size, ensuring it stays within defined bounds
		jitter := (rnd.Float64()*2 - 1) * (componentSizeRange * 0.15) // +/- 15% of the range
		size := baseComponentSize + jitter
		size = math.Max(float64(COMPONENT_MIN_SIZE), math.Min(float64(COMPONENT_MAX_SIZE), size))
		color := palette[rnd.Intn(len(palette))]
		dc.SetHexColor(color)

		shapeChoice := rnd.Float64()

		switch {
		case shapeChoice < 0.15: // Rectangle
			dc.Push()
			dc.RotateAbout(gg.Radians(rnd.Float64()*180), point.X, point.Y)
			dc.DrawRectangle(point.X-size/2, point.Y-size/2, size, size)
			dc.Pop()
		case shapeChoice < 0.30: // Ellipse
			dc.Push()
			rx := size / 2
			ry := rx * (rnd.Float64()*0.7 + 0.3) // 30% to 100% of rx
			dc.RotateAbout(gg.Radians(rnd.Float64()*180), point.X, point.Y)
			dc.DrawEllipse(point.X, point.Y, rx, ry)
			dc.Pop()
		case shapeChoice < 0.45: // Pie Slice
			dc.Push()
			dc.RotateAbout(gg.Radians(rnd.Float64()*360), point.X, point.Y)
			start := rnd.Float64() * 360
			end := start + rnd.Float64()*255 + 45
			dc.DrawEllipticalArc(point.X, point.Y, size/2, size/2, gg.Radians(start), gg.Radians(end))
			dc.LineTo(point.X, point.Y)
			dc.ClosePath()
			dc.Pop()
		case shapeChoice < 0.60: // Hexagon
			dc.DrawRegularPolygon(6, point.X, point.Y, size/2, gg.Radians(rnd.Float64()*60))
		case shapeChoice < 0.70: // Star
			dc.Push()
			dc.RotateAbout(gg.Radians(rnd.Float64()*360), point.X, point.Y)
			points := rnd.Intn(3) + 5 // 5, 6, or 7 points
			drawStar(dc, float64(points), point.X, point.Y, size/2)
			dc.Pop()
		case shapeChoice < 0.85: // Right Triangle
			dc.Push()
			dc.RotateAbout(gg.Radians(rnd.Float64()*360), point.X, point.Y)
			p1 := point
			quadrant := rnd.Intn(4) + 1
			var p2, p3 Point
			switch quadrant {
			case 1:
				p2 = Point{X: point.X + size, Y: point.Y}
				p3 = Point{X: point.X, Y: point.Y + size}
			case 2:
				p2 = Point{X: point.X - size, Y: point.Y}
				p3 = Point{X: point.X, Y: point.Y + size}
			case 3:
				p2 = Point{X: point.X - size, Y: point.Y}
				p3 = Point{X: point.X, Y: point.Y - size}
			default:
				p2 = Point{X: point.X + size, Y: point.Y}
				p3 = Point{X: point.X, Y: point.Y - size}
			}
			dc.MoveTo(p1.X, p1.Y)
			dc.LineTo(p2.X, p2.Y)
			dc.LineTo(p3.X, p3.Y)
			dc.ClosePath()
			dc.Pop()
		case shapeChoice < 0.95: // Parabola
			dc.SetHexColor(color)
			dc.SetLineWidth(float64(rnd.Intn(3) + 1))
			flip := rnd.Float64() < 0.5
			for xLocal := -size; xLocal <= size; xLocal++ {
				yLocal := math.Pow(xLocal/size, 2) * size
				if flip {
					dc.LineTo(point.X+xLocal, point.Y-size+yLocal)
				} else {
					dc.LineTo(point.X-size+yLocal, point.Y+xLocal)
				}
			}
			dc.Stroke()
			continue // Use continue to not fill
		default: // Random Triangle
			p1 := Point{X: point.X + rnd.Float64()*size*2 - size, Y: point.Y + rnd.Float64()*size*2 - size}
			p2 := Point{X: point.X + rnd.Float64()*size*2 - size, Y: point.Y + rnd.Float64()*size*2 - size}
			p3 := Point{X: point.X + rnd.Float64()*size*2 - size, Y: point.Y + rnd.Float64()*size*2 - size}
			dc.MoveTo(p1.X, p1.Y)
			dc.LineTo(p2.X, p2.Y)
			dc.LineTo(p3.X, p3.Y)
			dc.ClosePath()
		}
		dc.Fill()
	}

	// 6. --- Post-processing (on the raster image) ---
	img := dc.Image()
	img = imaging.Blur(img, 1.1) // Soften
	img = addVignette(img)       // Add vignette

	return img
}

func drawStar(dc *gg.Context, points, centerX, centerY, outerRadius float64) {
	innerRadius := outerRadius * 0.4
	dc.NewSubPath()
	for i := 0.0; i < points*2; i++ {
		r := outerRadius
		if int(i)%2 != 0 {
			r = innerRadius
		}
		angle := (math.Pi*2/(points*2))*i - math.Pi/2
		x := centerX + r*math.Cos(angle)
		y := centerY + r*math.Sin(angle)
		dc.LineTo(x, y)
	}
	dc.ClosePath()
}

func addVignette(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := float64(bounds.Dx()), float64(bounds.Dy())

	// Create a new image to draw the vignette on
	resultImg := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get original color
			r, g, b, a := img.At(x, y).RGBA()

			// Calculate distance from center, normalized to [0, 1]
			dx := (float64(x) - width/2) / (width / 2)
			dy := (float64(y) - height/2) / (height / 2)
			dist := math.Sqrt(dx*dx+dy*dy) / math.Sqrt(2) // Divide by sqrt(2) to ensure it's in [0, 1] for corners

			// Simple quadratic fade
			fade := math.Max(0, 1-dist*dist*0.9)

			// Apply the fade
			resultImg.SetNRGBA(x, y, color.NRGBA{
				R: uint8((float64(r>>8) * fade)),
				G: uint8((float64(g>>8) * fade)),
				B: uint8((float64(b>>8) * fade)),
				A: uint8(a >> 8),
			})
		}
	}
	return resultImg
}
