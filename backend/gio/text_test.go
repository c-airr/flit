package gio

import (
	"testing"

	"github.com/c-airr/flit/draw"
)

var (
	regular  = draw.FontStyle{Size: 14, Weight: 400}
	semibold = draw.FontStyle{Size: 14, Weight: 600}
)

func newTestMeasurer() *measurer {
	return newMeasurer(newShaper())
}

func TestMeasureEmpty(t *testing.T) {
	m := newTestMeasurer()
	empty := m.Measure("", regular, 0)
	line := m.Measure("A", regular, 0)
	if empty.W != 0 || empty.H != line.H || line.H <= 0 {
		t.Errorf(`Measure("") = %+v, want W 0 and H %v (one line)`, empty, line.H)
	}
}

func TestMeasureWraps(t *testing.T) {
	m := newTestMeasurer()
	s := "The quick brown fox jumps over the lazy dog"
	one := m.Measure(s, regular, 0)
	wrapped := m.Measure(s, regular, 100)
	if wrapped.W > 100 || wrapped.H < 2*one.H {
		t.Errorf("wrapped at 100: %+v, want W <= 100 and H >= %v (single line %+v)", wrapped, 2*one.H, one)
	}
}

func TestMeasureNewline(t *testing.T) {
	m := newTestMeasurer()
	a, ab := m.Measure("a", regular, 0), m.Measure("a\nb", regular, 0)
	// About two lines, not exactly: one line's box is ascent+descent, while
	// a second line adds the line spacing, which is a bit smaller.
	if ab.H < 1.8*a.H || ab.H > 2.2*a.H {
		t.Errorf(`"a\nb" height %v, want about 2x %v`, ab.H, a.H)
	}
}

func TestMeasureWeightAndSize(t *testing.T) {
	m := newTestMeasurer()
	reg, bold := m.Measure("Hello", regular, 0), m.Measure("Hello", semibold, 0)
	if bold.W <= reg.W {
		t.Errorf("semibold width %v, want more than regular %v", bold.W, reg.W)
	}
	big := m.Measure("Hello", draw.FontStyle{Size: 28, Weight: 400}, 0)
	if ratio := big.W / reg.W; ratio < 1.8 || ratio > 2.2 {
		t.Errorf("size 28 is %.2fx as wide as size 14, want about 2x", ratio)
	}
}

func TestMeasureScaleInvariant(t *testing.T) {
	m := newTestMeasurer()
	s := "Zażółć gęślą jaźń"
	at1 := m.Measure(s, regular, 0)
	m.setScale(1.5)
	at15 := m.Measure(s, regular, 0)
	if abs(at1.W-at15.W) > 1 || abs(at1.H-at15.H) > 1 {
		t.Errorf("scale 1: %+v, scale 1.5: %+v; want within 1 dp", at1, at15)
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
