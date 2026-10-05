// Command spike-gio is a throwaway check that a hand-built draw.List can be
// drawn with Gio: rounded rectangles, outlines, clipping and Inter text.
// The real backend (spec step 7) replaces it.
package main

import (
	"image"
	"image/color"
	"log"
	"math"
	"os"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/c-airr/flit/backend/gio/fonts"
	"github.com/c-airr/flit/draw"
)

func main() {
	// Gio needs the main goroutine for the OS event loop (app.Main never
	// returns), so the window loop runs in a goroutine of its own.
	go func() {
		w := new(app.Window)
		w.Option(app.Title("flit spike"), app.Size(unit.Dp(400), unit.Dp(360)))
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	// Only the embedded Inter, no system fonts: text looks the same everywhere.
	shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection(fonts.Collection()))
	var ops op.Ops
	var list draw.List
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			s := gtx.Metric.PxPerDp
			viewport := draw.Size{W: float32(gtx.Constraints.Max.X) / s, H: float32(gtx.Constraints.Max.Y) / s}
			buildScene(&list, viewport)
			render(gtx, shaper, &list)
			e.Frame(gtx.Ops)
		}
	}
}

// buildScene fills l by hand with what ui will produce later.
// Coordinates are logical pixels (Gio's dp).
func buildScene(l *draw.List, vp draw.Size) {
	var (
		background = draw.Hex(0xF7F7F8)
		surface    = draw.Hex(0xFFFFFF)
		primary    = draw.Hex(0x4F46E5)
		onPrimary  = draw.Hex(0xFFFFFF)
		textColor  = draw.Hex(0x18181B)
		muted      = draw.Hex(0x71717A)
		outline    = draw.Hex(0xE4E4E7)
		regular    = draw.FontStyle{Size: 14, Weight: 400}
		semibold   = draw.FontStyle{Size: 14, Weight: 600}
	)
	l.Reset()
	l.FillRect(draw.Rect{W: vp.W, H: vp.H}, background, 0)

	card := draw.Rect{X: 24, Y: 24, W: 352, H: 184}
	l.FillRect(card, surface, 12)
	l.StrokeRect(card, outline, 12, 1)
	l.Text(draw.Rect{X: 48, Y: 44}, "Hello, flit", textColor, draw.FontStyle{Size: 20, Weight: 600})
	l.Text(draw.Rect{X: 48, Y: 80}, "Zażółć gęślą jaźń 123", muted, regular)
	l.FillRect(draw.Rect{X: 48, Y: 132, W: 120, H: 40}, primary, 8)
	l.Text(draw.Rect{X: 78, Y: 142}, "Click me", onPrimary, semibold)

	// A clipped card: the indigo block sticks out past the top-right corner
	// and must be cut along the rounded edge; the text wraps at 200.
	box := draw.Rect{X: 24, Y: 232, W: 352, H: 96}
	l.PushClip(box, 12)
	l.FillRect(box, surface, 0)
	l.FillRect(draw.Rect{X: 270, Y: 200, W: 200, H: 90}, primary, 24)
	l.Text(draw.Rect{X: 48, Y: 252, W: 200}, "This card clips its content. The block overflows its corner.", textColor, regular)
	l.PopClip()
	l.StrokeRect(box, outline, 12, 1)
}

// render translates l into Gio operations. s converts dp to pixels.
func render(gtx layout.Context, shaper *text.Shaper, l *draw.List) {
	if l.Version != draw.FormatVersion {
		log.Fatalf("draw.List version %d, renderer knows %d", l.Version, draw.FormatVersion)
	}
	s := gtx.Metric.PxPerDp
	var clips []clip.Stack
	for _, c := range l.Cmds {
		switch c.Kind {
		case draw.FillRect:
			rr := clip.UniformRRect(px(c.Rect, s), round(c.Radius*s))
			paint.FillShape(gtx.Ops, nrgba(c.Color), rr.Op(gtx.Ops))
		case draw.StrokeRect:
			rr := clip.UniformRRect(px(c.Rect, s), round(c.Radius*s))
			paint.FillShape(gtx.Ops, nrgba(c.Color), clip.Stroke{Path: rr.Path(gtx.Ops), Width: c.Width * s}.Op())
		case draw.Text:
			drawText(gtx, shaper, l.TextOf(c), c)
		case draw.PushClip:
			clips = append(clips, clip.UniformRRect(px(c.Rect, s), round(c.Radius*s)).Push(gtx.Ops))
		case draw.PopClip:
			clips[len(clips)-1].Pop()
			clips = clips[:len(clips)-1]
		}
	}
}

// drawText lays out one Text command with a Gio label. gtx is a copy (a
// struct passed by value), so changing its constraints here does not
// affect the caller.
func drawText(gtx layout.Context, shaper *text.Shaper, s string, c draw.Cmd) {
	scale := gtx.Metric.PxPerDp
	defer op.Offset(image.Pt(round(c.Rect.X*scale), round(c.Rect.Y*scale))).Push(gtx.Ops).Pop()

	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max = image.Pt(1<<20, 1<<20) // W 0 means "do not wrap"
	if c.Rect.W > 0 {
		gtx.Constraints.Max.X = round(c.Rect.W * scale)
	}

	// The label paints glyphs with whatever "material" it is given:
	// here a recorded color operation.
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: nrgba(c.Color)}.Add(gtx.Ops)
	material := rec.Stop()

	// Gio weights are CSS weights minus 400: Normal is 0, SemiBold is 200.
	f := font.Font{Typeface: fonts.Typeface, Weight: font.Weight(int(c.Font.Weight) - 400)}
	widget.Label{}.Layout(gtx, shaper, f, unit.Sp(c.Font.Size), s, material)
}

func px(r draw.Rect, s float32) image.Rectangle {
	return image.Rect(round(r.X*s), round(r.Y*s), round((r.X+r.W)*s), round((r.Y+r.H)*s))
}

func round(v float32) int {
	return int(math.Round(float64(v)))
}

func nrgba(c draw.RGBA) color.NRGBA {
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}
