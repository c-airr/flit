// Package draw defines the boundary between flit and a renderer: a flat,
// versioned list of draw commands produced once per frame, and the
// TextMeasurer interface a renderer provides back for layout.
//
// Everything here is plain data with no Go pointers inside the elements,
// so the command array and the text buffer can be handed to a C/C++
// renderer as they are. This package has no dependencies on purpose.
package draw

import (
	"fmt"
	"unsafe"
)

// FormatVersion identifies the layout and meaning of Cmd. Bump it on any
// change to Cmd or to how a Kind uses its fields. A renderer must refuse
// a List whose Version it does not know.
const FormatVersion uint32 = 1

// Point is a position in logical pixels.
type Point struct{ X, Y float32 }

// Size is a width and height in logical pixels.
type Size struct{ W, H float32 }

// Rect is an axis-aligned rectangle in logical pixels; X, Y is the top-left corner.
type Rect struct{ X, Y, W, H float32 }

// RGBA is a color with straight (non-premultiplied) alpha.
// In memory it is 4 bytes in R, G, B, A order, the same as uint8_t[4] in C.
type RGBA struct{ R, G, B, A uint8 }

// Hex builds an opaque color from 0xRRGGBB, e.g. Hex(0x4F46E5).
func Hex(rgb uint32) RGBA {
	return RGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xFF}
}

// Kind says which command a Cmd is and therefore which of its fields are used.
//
// Go has no enum keyword. The idiom is a named integer type plus a block
// of typed constants; iota counts up from 0 within the block.
type Kind uint32

const (
	FillRect   Kind = iota + 1 // Rect, Color, Radius. Zero is left unused, so a zeroed Cmd is invalid.
	StrokeRect                 // Rect, Color, Radius, Width
	Text                       // Rect (X, Y top-left; W max width, 0 = unlimited; H laid-out height), Color, TextOff/TextLen, Font
	PushClip                   // Rect, Radius
	PopClip                    // no fields
)

var kindNames = [...]string{
	FillRect:   "FillRect",
	StrokeRect: "StrokeRect",
	Text:       "Text",
	PushClip:   "PushClip",
	PopClip:    "PopClip",
}

// String makes Kind print by name in fmt and in test failures.
// Any type with a String() string method satisfies fmt.Stringer;
// Go interfaces are implemented implicitly, no "implements" keyword needed.
func (k Kind) String() string {
	if k > 0 && int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("Kind(%d)", uint32(k))
}

// FontStyle selects size (logical pixels) and weight (400 regular, 600 semibold, ...).
type FontStyle struct {
	Size   float32
	Weight uint32
}

// Cmd is one draw command. Which fields matter depends on Kind; the rest are zero.
//
// The layout is fixed: every field is 4 bytes wide (RGBA is 4 × uint8),
// so the compiler inserts no padding and Cmd is exactly 48 bytes.
// A C/C++ renderer declares the same struct and checks it with static_assert.
type Cmd struct {
	Kind    Kind      //  0
	Rect    Rect      //  4
	Color   RGBA      // 20
	Radius  float32   // 24  corner radius
	Width   float32   // 28  stroke width
	TextOff uint32    // 32  byte offset into List.TextBuf
	TextLen uint32    // 36  byte length; UTF-8, not NUL-terminated
	Font    FontStyle // 40
} // 48

// Compile-time size check, Go's version of static_assert(sizeof(Cmd) == 48).
// unsafe.Sizeof is a constant of the unsigned type uintptr, so if Cmd ever
// grows or shrinks, one of these subtractions goes below zero, the constant
// overflows uintptr and the package stops compiling.
var (
	_ [48 - unsafe.Sizeof(Cmd{})]byte
	_ [unsafe.Sizeof(Cmd{}) - 48]byte
)
