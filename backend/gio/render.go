package gio

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"

	"github.com/c-airr/flit/draw"
)

// render translates l into Gio operations in gtx.Ops. Coordinates in l are
// dp; gtx.Metric.PxPerDp turns them into pixels. It refuses a list of an
// unknown format version and a PopClip without a PushClip; clips left
// pushed at the end are popped.
func render(gtx layout.Context, sh *text.Shaper, l *draw.List) error {
	if l.Version != draw.FormatVersion {
		return fmt.Errorf("draw.List has format version %d, this renderer knows %d", l.Version, draw.FormatVersion)
	}
	s := gtx.Metric.PxPerDp
	var clips []clip.Stack
	// Gio clip stacks must be popped in reverse order, like nested scopes.
	defer func() {
		for i := len(clips) - 1; i >= 0; i-- {
			clips[i].Pop()
		}
	}()

	for i, c := range l.Cmds {
		switch c.Kind {
		case draw.FillRect:
			rr := clip.UniformRRect(px(c.Rect, s), round(c.Radius*s))
			paint.FillShape(gtx.Ops, nrgba(c.Color), rr.Op(gtx.Ops))
		case draw.StrokeRect:
			r, radius := strokeInset(c.Rect, c.Radius, c.Width)
			rr := clip.UniformRRect(px(r, s), round(radius*s))
			paint.FillShape(gtx.Ops, nrgba(c.Color), clip.Stroke{Path: rr.Path(gtx.Ops), Width: c.Width * s}.Op())
		case draw.Text:
			drawText(gtx, sh, l.TextOf(c), c)
		case draw.PushClip:
			clips = append(clips, clip.UniformRRect(px(c.Rect, s), round(c.Radius*s)).Push(gtx.Ops))
		case draw.PopClip:
			if len(clips) == 0 {
				return fmt.Errorf("command %d: PopClip without PushClip", i)
			}
			clips[len(clips)-1].Pop()
			clips = clips[:len(clips)-1]
		default:
			return fmt.Errorf("command %d: unknown kind %v", i, c.Kind)
		}
	}
	return nil
}

// strokeInset moves the outline of r inwards by half the stroke width.
// Gio strokes are centered on the path, so this keeps the whole stroke
// inside r (spec §3). The radius shrinks with it, but never below zero.
func strokeInset(r draw.Rect, radius, width float32) (draw.Rect, float32) {
	h := width / 2
	return draw.Rect{X: r.X + h, Y: r.Y + h, W: r.W - width, H: r.H - width}, max(0, radius-h)
}

func drawText(gtx layout.Context, sh *text.Shaper, s string, c draw.Cmd) {
	scale := gtx.Metric.PxPerDp
	defer op.Offset(image.Pt(round(c.Rect.X*scale), round(c.Rect.Y*scale))).Push(gtx.Ops).Pop()
	// The text paints its glyphs with a "material": here a recorded
	// color operation.
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: nrgba(c.Color)}.Add(gtx.Ops)
	material := rec.Stop()
	layoutText(gtx, sh, s, c.Font, c.Rect.W, material)
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
