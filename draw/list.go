package draw

import (
	"fmt"
	"strconv"
	"strings"
)

// List is everything a renderer needs to draw one frame.
//
// It is two flat arrays: Cmds, and TextBuf holding the text of all Text
// commands back to back. A renderer reads Cmds in order; a Text command
// points into TextBuf with TextOff/TextLen.
//
// The zero value is not ready to use: call Reset at the start of every
// frame. Reset sets Version and reuses the memory of the previous frame.
type List struct {
	Version uint32
	Cmds    []Cmd
	TextBuf []byte
}

// Reset prepares the list for a new frame. It keeps the allocated
// capacity of both slices, so once they have grown to the size of a
// typical frame, building a frame allocates nothing.
//
// s[:0] is a slice of length 0 over the same backing array, similar to
// std::vector::clear(), which also keeps capacity.
func (l *List) Reset() {
	l.Version = FormatVersion
	l.Cmds = l.Cmds[:0]
	l.TextBuf = l.TextBuf[:0]
}

// FillRect fills r with c; radius > 0 rounds the corners.
func (l *List) FillRect(r Rect, c RGBA, radius float32) {
	l.Cmds = append(l.Cmds, Cmd{Kind: FillRect, Rect: r, Color: c, Radius: radius})
}

// StrokeRect draws the outline of r, width logical pixels thick.
func (l *List) StrokeRect(r Rect, c RGBA, radius, width float32) {
	l.Cmds = append(l.Cmds, Cmd{Kind: StrokeRect, Rect: r, Color: c, Radius: radius, Width: width})
}

// Text draws s with its top-left corner at r.X, r.Y. r.W is the width to
// wrap at (0 = no wrapping) and r.H is the height layout measured.
// The string is copied into TextBuf.
func (l *List) Text(r Rect, s string, c RGBA, f FontStyle) {
	off := len(l.TextBuf)
	// append(bytes, str...) copies the string's bytes into the slice.
	// The "..." spreads the string as a sequence of bytes.
	l.TextBuf = append(l.TextBuf, s...)
	l.Cmds = append(l.Cmds, Cmd{
		Kind:    Text,
		Rect:    r,
		Color:   c,
		TextOff: uint32(off),
		TextLen: uint32(len(s)), // len of a string counts bytes, not characters
		Font:    f,
	})
}

// PushClip restricts drawing to r (rounded by radius) until the matching PopClip.
// Clips nest: the effective clip is the intersection of all pushed ones.
func (l *List) PushClip(r Rect, radius float32) {
	l.Cmds = append(l.Cmds, Cmd{Kind: PushClip, Rect: r, Radius: radius})
}

// PopClip removes the most recently pushed clip.
func (l *List) PopClip() {
	l.Cmds = append(l.Cmds, Cmd{Kind: PopClip})
}

// TextOf returns the text of a Text command.
func (l *List) TextOf(c Cmd) string {
	// Converting []byte to string copies the bytes. That is fine here:
	// TextOf is for debugging and backends, not for the per-frame hot path.
	return string(l.TextBuf[c.TextOff : c.TextOff+c.TextLen])
}

// String prints the list one command per line, in a stable format that
// golden tests compare against, e.g.
//
//	FillRect x=0 y=0 w=100 h=40 color=#4f46e5ff radius=8
//	Text x=16 y=12 w=0 color=#ffffffff size=14 weight=400 "Clicks"
func (l *List) String() string {
	var b strings.Builder
	for _, c := range l.Cmds {
		b.WriteString(c.Kind.String())
		switch c.Kind {
		case FillRect:
			writeRect(&b, c.Rect)
			fmt.Fprintf(&b, " color=%s radius=%s", hexColor(c.Color), num(c.Radius))
		case StrokeRect:
			writeRect(&b, c.Rect)
			fmt.Fprintf(&b, " color=%s radius=%s width=%s", hexColor(c.Color), num(c.Radius), num(c.Width))
		case Text:
			fmt.Fprintf(&b, " x=%s y=%s w=%s color=%s size=%s weight=%d %q",
				num(c.Rect.X), num(c.Rect.Y), num(c.Rect.W), hexColor(c.Color),
				num(c.Font.Size), c.Font.Weight, l.TextOf(c))
		case PushClip:
			writeRect(&b, c.Rect)
			fmt.Fprintf(&b, " radius=%s", num(c.Radius))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func writeRect(b *strings.Builder, r Rect) {
	fmt.Fprintf(b, " x=%s y=%s w=%s h=%s", num(r.X), num(r.Y), num(r.W), num(r.H))
}

// num formats a float32 as short as possible: 8, 0.5, 1e+06.
func num(v float32) string {
	return strconv.FormatFloat(float64(v), 'g', -1, 32)
}

func hexColor(c RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}
