package fakebackend

import (
	"math"
	"testing"

	"github.com/c-airr/flit/draw"
)

func TestMeasure(t *testing.T) {
	tests := []struct {
		text     string
		maxWidth float32
		want     draw.Size
	}{
		{"", 0, draw.Size{W: 0, H: 16}},
		{"Clicks", 0, draw.Size{W: 48, H: 16}},
		{"Zażółć", 0, draw.Size{W: 48, H: 16}}, // 6 runes, 10 bytes: width counts runes
		{"ab\ncde", 0, draw.Size{W: 24, H: 32}},
		{"abcdef", 0, draw.Size{W: 48, H: 16}},                    // 0 = no wrapping
		{"abcdef", 20, draw.Size{W: 16, H: 48}},                   // 2 chars per line
		{"abcdef", 100, draw.Size{W: 48, H: 16}},                  // fits on one line
		{"abc", 4, draw.Size{W: 8, H: 48}},                        // narrower than one char: still 1 per line
		{"abc\n\nd", 16, draw.Size{W: 16, H: 64}},                 // an empty line is still one line when wrapping
		{"abcdef", float32(math.Inf(1)), draw.Size{W: 48, H: 16}}, // Inf = no wrapping, like 0
	}
	var m Measurer
	// The fake ignores the style; pass a non-default one to prove it.
	style := draw.FontStyle{Size: 99, Weight: 700}
	for _, tt := range tests {
		if got := m.Measure(tt.text, style, tt.maxWidth); got != tt.want {
			t.Errorf("Measure(%q, maxWidth=%g) = %+v, want %+v", tt.text, tt.maxWidth, got, tt.want)
		}
	}
}
