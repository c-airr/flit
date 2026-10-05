// Package ui is flit's public API: widgets, layout, state and the app loop.
package ui

import (
	"math"

	"github.com/c-airr/flit/draw"
)

// Point and Size are aliases: ui.Size and draw.Size are the same type, not
// two types with the same fields. The "=" is what makes it an alias, like
// `using Size = draw::Size;` in C++. Without it Go would create a new
// distinct type and every hand-off to draw would need a conversion.
type (
	Point = draw.Point
	Size  = draw.Size
)

// Inf is an unbounded constraint: "take as much as you need".
// Go constants cannot be infinite, so this is a variable set at start-up.
var Inf = float32(math.Inf(1))

// Constraints are the smallest and largest size a parent allows a child.
// Layout passes constraints down the tree and sizes back up: the child picks
// its size inside these bounds, then the parent decides where to put it.
type Constraints struct {
	MinW, MaxW, MinH, MaxH float32
}

// Tight constraints allow exactly one size.
func Tight(s Size) Constraints {
	return Constraints{MinW: s.W, MaxW: s.W, MinH: s.H, MaxH: s.H}
}

// Loosen keeps the maximums and drops the minimums to zero.
//
// c is a value receiver: the method gets a copy, so changing c here does not
// touch the caller's value. It works like a const member function in C++
// that returns a modified copy.
func (c Constraints) Loosen() Constraints {
	c.MinW, c.MinH = 0, 0
	return c
}

// Constrain clamps each dimension of s into [min, max].
func (c Constraints) Constrain(s Size) Size {
	return Size{W: clamp(s.W, c.MinW, c.MaxW), H: clamp(s.H, c.MinH, c.MaxH)}
}

// Deflate shrinks the constraints by the insets, for laying out what sits
// inside a padding. Nothing goes below zero, and Inf stays Inf, because in
// IEEE floats Inf minus a finite number is still Inf.
//
// min and max are built into the language since Go 1.21 and work on any
// ordered type, so there is no std::min / std::max import.
func (c Constraints) Deflate(in Insets) Constraints {
	dw, dh := in.Left+in.Right, in.Top+in.Bottom
	return Constraints{
		MinW: max(0, c.MinW-dw),
		MaxW: max(0, c.MaxW-dw),
		MinH: max(0, c.MinH-dh),
		MaxH: max(0, c.MaxH-dh),
	}
}

func clamp(v, lo, hi float32) float32 {
	return min(max(v, lo), hi)
}

func isInf(v float32) bool {
	return math.IsInf(float64(v), 1)
}

// finiteOr returns v, or fallback when v is Inf. Layout uses it so that an
// unbounded constraint never leaks out as an Inf size.
func finiteOr(v, fallback float32) float32 {
	if isInf(v) {
		return fallback
	}
	return v
}

// Insets are the space on each side of a box, in CSS order.
type Insets struct {
	Top, Right, Bottom, Left float32
}

// All puts the same inset on every side.
func All(v float32) Insets {
	return Insets{Top: v, Right: v, Bottom: v, Left: v}
}

// Symmetric puts vertical on the top and bottom, horizontal on the left and right.
func Symmetric(vertical, horizontal float32) Insets {
	return Insets{Top: vertical, Right: horizontal, Bottom: vertical, Left: horizontal}
}
