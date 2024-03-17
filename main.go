package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"syscall/js"

	"github.com/signintech/gopdf"
)

//go:embed fonts/*
var fonts embed.FS

type pdfRequest struct {
	Size        string  `json:"size"`
	Orientation string  `json:"orientation"`
	Pattern     pattern `json:"pattern"`
}

type pattern struct {
	Name      string `json:"name"`
	Width     string `json:"width"`
	Height    string `json:"height"`
	Color     string `json:"color"`
	LineWidth string `json:"lineWidth"`

	width     float64
	height    float64
	lineWidth float64

	Size string `json:"size"` // deprecated
}

var errUnknownPattern = fmt.Errorf("unknown pattern")

func main() {
	js.Global().Set("generatePDF", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return generatePDF(args[0].String())
	}))

	select {}
}

func generatePDF(request string) string {
	console := js.Global().Get("console")

	defer func() {
		if r := recover(); r != nil {
			console.Call("error", fmt.Sprintf("recovered: %v", r))
			return
		}
	}()

	req, err := parsePDFRequest(request)
	if err != nil {
		console.Call("error", fmt.Sprintf("error parsing request: %v", err))
		return ""
	}

	pageSize, err := getPageSize(req.Size, req.Orientation)
	if err != nil {
		console.Call("error", fmt.Sprintf("error getting page size: %v", err))
		return ""
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: *pageSize,
		Unit:     gopdf.Unit_MM,
	})
	pdf.AddPage()

	// block below adds an empty text cell to the PDF;
	// it's a mystery why this is needed for this library,
	// but if it's not done, the PDF will be invalid
	data, err := fonts.ReadFile("fonts/empty.ttf")
	if err != nil {
		console.Call("error", fmt.Sprintf("read ttf font error: %v", err))
		return ""
	}
	if err := pdf.AddTTFFontData("empty", data); err != nil {
		console.Call("error", fmt.Sprintf("add ttf font error: %v", err))
		return ""
	}
	if err := pdf.SetFont("empty", "", 14); err != nil {
		console.Call("error", fmt.Sprintf("set font error: %v", err))
		return ""
	}
	pdf.Cell(nil, "") // print text

	if err := drawPattern(&pdf, pageSize, req.Pattern); err != nil {
		console.Call("error", fmt.Sprintf("error drawing pattern: %v", err))
		return ""
	}

	var b bytes.Buffer
	if _, err := pdf.WriteTo(&b); err != nil {
		console.Call("error", fmt.Sprintf("error writing PDF: %v", err))
		return ""
	}

	console.Call("log", "PDF generated")

	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func parsePDFRequest(request string) (*pdfRequest, error) {
	var req pdfRequest
	err := json.Unmarshal([]byte(request), &req)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling request: %w", err)
	}

	// convert pattern size from string to float64
	width := req.Pattern.Width
	if width == "" { // fallback to deprecated field
		width = req.Pattern.Size
	}
	req.Pattern.width, err = strconv.ParseFloat(width, 64)
	if err != nil {
		return nil, fmt.Errorf("error converting pattern width: %w", err)
	}
	req.Pattern.height, err = strconv.ParseFloat(req.Pattern.Height, 64)
	if err != nil {
		return nil, fmt.Errorf("error converting pattern height: %w", err)
	}

	req.Pattern.lineWidth, err = strconv.ParseFloat(req.Pattern.LineWidth, 64)
	if err != nil {
		return nil, fmt.Errorf("error converting pattern line width: %w", err)
	}

	return &req, nil
}

func getPageSize(size, orientation string) (*gopdf.Rect, error) {
	var w, h int
	switch size {
	case "Letter":
		w, h = 216, 279
	case "Legal":
		w, h = 216, 356
	case "A1":
		w, h = 594, 841
	case "A2":
		w, h = 420, 594
	case "A3":
		w, h = 297, 420
	case "A4":
		w, h = 210, 297
	case "A5":
		w, h = 148, 210
	default:
		return nil, fmt.Errorf("unknown page size: %s", size)
	}

	if orientation == "L" {
		w, h = h, w
	}

	return &gopdf.Rect{W: float64(w), H: float64(h)}, nil
}

func drawPattern(pdf *gopdf.GoPdf, pageSize *gopdf.Rect, pattern pattern) error {
	r, g, b, a := convertColor(pattern.Color)

	pdf.SetStrokeColor(r, g, b)

	if a != 1.0 {
		t, err := gopdf.NewTransparency(a, string(gopdf.Overlay))
		if err != nil {
			return fmt.Errorf("error creating transparency: %w", err)
		}

		pdf.SetTransparency(t)
	}

	w := pageSize.W
	h := pageSize.H
	patternWidth := pattern.width
	patternHeight := pattern.height

	pdf.SetLineWidth(pattern.lineWidth / 1000)

	switch pattern.Name {
	case "rect":
		for x := 0.0; x < w; x += patternWidth {
			for y := 0.0; y < h; y += patternHeight {
				pdf.Rectangle(x, y, x+patternWidth, y+patternHeight, "D", 0, 0)
			}
		}
	case "lines":
		for y := 0.0; y < h; y += patternHeight {
			pdf.Line(0, y, w, y)
		}
	case "dot":
		pdf.SetFillColor(r, g, b)
		r := pattern.lineWidth / 1000
		for x := 0.0; x < w; x += patternWidth {
			for y := 0.0; y < h; y += patternHeight {
				pdf.Rectangle(
					x-r, y-r,
					x+r, y+r,
					"F", r, 10,
				)
			}
		}
	case "diamond": // not used
		for x := 0.0; x < w; x += patternWidth {
			for y := 0.0; y < h; y += patternHeight {
				pdf.Polygon(
					[]gopdf.Point{
						{X: x + patternWidth/2, Y: y},
						{X: x, Y: y + patternHeight/2},
						{X: x + patternWidth/2, Y: y + patternHeight},
						{X: x + patternWidth, Y: y + patternHeight/2},
						{X: x + patternWidth/2, Y: y},
					},
					"D",
				)
			}
		}
	case "rhombus":
		for x := 0.0; x < w; x += patternWidth {
			for y := 0.0; y < h; y += patternHeight {
				pdf.Polygon(
					[]gopdf.Point{
						{X: x + patternWidth/2, Y: y},
						{X: x, Y: y + patternHeight/2},
						{X: x + patternWidth/2, Y: y + patternHeight},
						{X: x + patternWidth, Y: y + patternHeight/2},
						{X: x + patternWidth/2, Y: y},
					},
					"D",
				)
			}
		}
	case "triangles":
		for x := 0.0; x < w; x += patternWidth {
			pdf.Line(x, 0, x, h)
		}

		for x := 0.0; x < w; x += patternWidth {
			for y := 0.0; y < h; y += patternHeight {
				pdf.Polygon(
					[]gopdf.Point{
						{X: x + patternWidth/2, Y: y},
						{X: x, Y: y + patternHeight/2},
						{X: x + patternWidth/2, Y: y + patternHeight},
						{X: x + patternWidth/2, Y: y},
						{X: x + patternWidth, Y: y + patternHeight/2},
						{X: x + patternWidth/2, Y: y + patternHeight},
					},
					"D",
				)
			}
		}
	case "hexdot":
		pdf.SetFillColor(r, g, b)
		r := pattern.lineWidth / 1000
		for x := 0.0; x < w; x += patternWidth {
			for y := 0.0; y < h; y += patternHeight {
				pdf.Rectangle(
					x+patternWidth/2-r, y-r,
					x+patternWidth/2+r, y+r,
					"F", r, 10,
				)
				pdf.Rectangle(
					x-r, y+patternHeight/2-r,
					x+r, y+patternHeight/2+r,
					"F", r, 10,
				)
			}
		}

	default:
		return fmt.Errorf("%w: %s", errUnknownPattern, pattern.Name)
	}

	return nil
}

func convertColor(hex string) (uint8, uint8, uint8, float64) {
	var r, g, b, a int64

	if len(hex) == 7 {
		r, _ = strconv.ParseInt(hex[1:3], 16, 0)
		g, _ = strconv.ParseInt(hex[3:5], 16, 0)
		b, _ = strconv.ParseInt(hex[5:7], 16, 0)
		return uint8(r), uint8(g), uint8(b), 1.0
	}

	if len(hex) == 9 {
		r, _ = strconv.ParseInt(hex[1:3], 16, 0)
		g, _ = strconv.ParseInt(hex[3:5], 16, 0)
		b, _ = strconv.ParseInt(hex[5:7], 16, 0)
		a, _ = strconv.ParseInt(hex[7:9], 16, 0)
		return uint8(r), uint8(g), uint8(b), float64(a) / 255.0
	}

	return 0, 0, 0, 0
}
