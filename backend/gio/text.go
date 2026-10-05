// Package gio is flit's backend on top of Gio (gioui.org): it opens the
// window, turns Gio input into ui.PointerEvents, measures text and draws
// draw.Lists. Only this package knows about Gio.
package gio

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/c-airr/flit/backend/gio/fonts"
	"github.com/c-airr/flit/draw"
)

// unbounded is a pixel size larger than any window, used for "no limit".
const unbounded = 1 << 24

// newShaper returns a text shaper that knows only the embedded Inter, so
// text looks the same on every machine.
func newShaper() *text.Shaper {
	return text.NewShaper(text.NoSystemFonts(), text.WithCollection(fonts.Collection()))
}

// layoutText is the one place text is laid out, both to draw it and to
// measure it, so a measured size always matches what is drawn. It records
// the text's drawing operations into gtx.Ops at the current offset.
//
// maxWidth is in dp; 0 means no wrapping. gtx.Metric must have
// PxPerSp == PxPerDp, because flit sizes fonts in dp (spec §7).
func layoutText(gtx layout.Context, sh *text.Shaper, s string, style draw.FontStyle, maxWidth float32, material op.CallOp) layout.Dimensions {
	// gtx is a copy (layout.Context is a struct passed by value), so these
	// constraint changes stay inside this function.
	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max = image.Pt(unbounded, unbounded)
	if maxWidth > 0 {
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(maxWidth))
	}
	// Gio weights are CSS weights minus 400: Normal is 0, SemiBold is 200.
	f := font.Font{Typeface: fonts.Typeface, Weight: font.Weight(int(style.Weight) - 400)}
	return widget.Label{}.Layout(gtx, sh, f, unit.Sp(style.Size), s, material)
}

// measurer is the draw.TextMeasurer for ui's layout. It lays the text out
// for real into scratch operations that are never drawn and reports the
// size in dp.
type measurer struct {
	shaper *text.Shaper
	ops    op.Ops
	scale  float32 // pixels per dp
}

var _ draw.TextMeasurer = (*measurer)(nil)

func newMeasurer(sh *text.Shaper) *measurer {
	return &measurer{shaper: sh, scale: 1}
}

// setScale sets the window's pixels per dp; call it every frame, since the
// window can move to a monitor with another scale.
func (m *measurer) setScale(pxPerDp float32) {
	m.scale = pxPerDp
}

func (m *measurer) Measure(s string, style draw.FontStyle, maxWidth float32) draw.Size {
	m.ops.Reset()
	gtx := layout.Context{
		Ops:    &m.ops,
		Metric: unit.Metric{PxPerDp: m.scale, PxPerSp: m.scale},
	}
	dims := layoutText(gtx, m.shaper, s, style, maxWidth, op.CallOp{})
	return draw.Size{W: float32(dims.Size.X) / m.scale, H: float32(dims.Size.Y) / m.scale}
}
