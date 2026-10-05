// Package fakebackend holds stand-ins for a real renderer, for tests.
// Its measurer gives every character the same box, so expected layout
// sizes can be worked out by hand.
//
// It lives under internal/, which Go enforces: only code inside this
// module can import it, so it never becomes part of the public API.
package fakebackend

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/c-airr/flit/draw"
)

// Every rune is CharW wide and every line LineH tall, whatever the style.
const (
	CharW = 8
	LineH = 16
)

// Measurer is a draw.TextMeasurer with fixed-size characters.
// The zero value is ready to use.
type Measurer struct{}

// A compile-time check that Measurer implements draw.TextMeasurer.
// Go interfaces are satisfied implicitly (no "implements"), so without this
// line a wrong method signature would only fail where a Measurer is first
// used as a TextMeasurer, possibly in another package.
var _ draw.TextMeasurer = Measurer{}

// Measure splits text on '\n'. With a finite maxWidth > 0 every line wraps
// after floor(maxWidth/CharW) runes, at least one; 0 or Inf means no wrapping.
// The style is ignored; "_" names a parameter the function does not use.
func (Measurer) Measure(text string, _ draw.FontStyle, maxWidth float32) draw.Size {
	wrap := maxWidth > 0 && !math.IsInf(float64(maxWidth), 1)
	perLine := 0
	if wrap {
		perLine = max(1, int(maxWidth/CharW))
	}

	var w float32
	lines := 0
	for _, line := range strings.Split(text, "\n") {
		// len(line) would count bytes; text width depends on characters.
		runes := utf8.RuneCountInString(line)
		if !wrap {
			lines++
			w = max(w, float32(runes*CharW))
			continue
		}
		lines += max(1, (runes+perLine-1)/perLine) // integer ceil(runes/perLine)
		w = max(w, float32(min(runes, perLine)*CharW))
	}
	return draw.Size{W: w, H: float32(lines * LineH)}
}
