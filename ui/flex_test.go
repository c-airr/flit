package ui

import (
	"slices"
	"testing"
)

func childOffsets(n *node) []Point {
	var out []Point
	for _, c := range n.children {
		out = append(out, c.offset)
	}
	return out
}

// Under Constraints{0, 200, 0, 100}: Text("ab") is 16x16, Text("abcd") 32x16.
func TestFlexColumnAlignment(t *testing.T) {
	ab, abcd := Text("ab"), Text("abcd")
	tests := []struct {
		name        string
		w           FlexWidget
		wantOffsets []Point
		wantSize    Size
	}{
		{"gap; main = max, cross = widest", Column(ab, abcd).Gap(8),
			[]Point{{X: 0, Y: 0}, {X: 0, Y: 24}}, Size{W: 32, H: 100}},
		{"main center", Column(ab, abcd).Gap(8).MainAlign(Center),
			[]Point{{X: 0, Y: 30}, {X: 0, Y: 54}}, Size{W: 32, H: 100}},
		{"main end", Column(ab, abcd).Gap(8).MainAlign(End),
			[]Point{{X: 0, Y: 60}, {X: 0, Y: 84}}, Size{W: 32, H: 100}},
		{"main space between", Column(ab, abcd).Gap(8).MainAlign(SpaceBetween),
			[]Point{{X: 0, Y: 0}, {X: 0, Y: 84}}, Size{W: 32, H: 100}},
		{"main stretch acts as start", Column(ab, abcd).Gap(8).MainAlign(Stretch),
			[]Point{{X: 0, Y: 0}, {X: 0, Y: 24}}, Size{W: 32, H: 100}},
		{"cross center", Column(ab, abcd).Gap(8).CrossAlign(Center),
			[]Point{{X: 8, Y: 0}, {X: 0, Y: 24}}, Size{W: 32, H: 100}},
		{"cross end", Column(ab, abcd).Gap(8).CrossAlign(End),
			[]Point{{X: 16, Y: 0}, {X: 0, Y: 24}}, Size{W: 32, H: 100}},
		{"cross space between acts as start", Column(ab, abcd).Gap(8).CrossAlign(SpaceBetween),
			[]Point{{X: 0, Y: 0}, {X: 0, Y: 24}}, Size{W: 32, H: 100}},
		{"cross stretch takes the max", Column(ab, abcd).Gap(8).CrossAlign(Stretch),
			[]Point{{X: 0, Y: 0}, {X: 0, Y: 24}}, Size{W: 200, H: 100}},
		{"no children", Column(), nil, Size{W: 0, H: 100}},
		{"nil children are dropped", Column(ab, nil, abcd).Gap(8),
			[]Point{{X: 0, Y: 0}, {X: 0, Y: 24}}, Size{W: 32, H: 100}},
	}
	for _, tt := range tests {
		n := layoutUnder(tt.w, Constraints{0, 200, 0, 100})
		if got := childOffsets(n); !slices.Equal(got, tt.wantOffsets) {
			t.Errorf("%s: offsets = %v, want %v", tt.name, got, tt.wantOffsets)
		}
		if n.size != tt.wantSize {
			t.Errorf("%s: size = %+v, want %+v", tt.name, n.size, tt.wantSize)
		}
	}
}

func TestFlexStretchTightensChildren(t *testing.T) {
	n := layoutUnder(Column(Text("ab"), Text("abcd")).CrossAlign(Stretch), Constraints{0, 200, 0, 100})
	for i, c := range n.children {
		if c.size.W != 200 {
			t.Errorf("child %d width = %v, want 200", i, c.size.W)
		}
	}
}

func TestFlexOverflowHasNoNegativeGaps(t *testing.T) {
	var xs []Widget
	for range 8 { // range over an int: runs 8 times (Go 1.22+)
		xs = append(xs, Text("x"))
	}
	// Variadic call with a slice: xs... passes the slice as the children.
	n := layoutUnder(Column(xs...).MainAlign(SpaceBetween), Constraints{0, 200, 0, 100})

	want := []Point{{Y: 0}, {Y: 16}, {Y: 32}, {Y: 48}, {Y: 64}, {Y: 80}, {Y: 96}, {Y: 112}}
	if got := childOffsets(n); !slices.Equal(got, want) {
		t.Errorf("offsets = %v, want %v", got, want)
	}
	if n.size != (Size{W: 8, H: 100}) {
		t.Errorf("size = %+v, want {8 100} (clamped to max)", n.size)
	}
}

func TestFlexUnbounded(t *testing.T) {
	tests := []struct {
		name string
		w    FlexWidget
		c    Constraints
		want Size
	}{
		{"main falls back to the total", Column(Text("ab")), Constraints{0, Inf, 0, Inf}, Size{W: 16, H: 16}},
		{"main never below min", Column(Text("ab")), Constraints{0, Inf, 50, Inf}, Size{W: 16, H: 50}},
		{"stretch with unbounded cross takes the widest", Column(Text("ab")).CrossAlign(Stretch),
			Constraints{0, Inf, 0, Inf}, Size{W: 16, H: 16}},
	}
	for _, tt := range tests {
		if got := layoutUnder(tt.w, tt.c).size; got != tt.want {
			t.Errorf("%s: size = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestFlexRow(t *testing.T) {
	// Text("a\nb") is two lines: 8x32.
	n := layoutUnder(Row(Text("ab"), Text("a\nb")).Gap(4).CrossAlign(End), Constraints{0, 200, 0, 100})

	want := []Point{{X: 0, Y: 16}, {X: 20, Y: 0}}
	if got := childOffsets(n); !slices.Equal(got, want) {
		t.Errorf("offsets = %v, want %v", got, want)
	}
	if n.size != (Size{W: 200, H: 32}) {
		t.Errorf("size = %+v, want {200 32}", n.size)
	}
}

func TestFlexShrinkingDisposesExtraChildren(t *testing.T) {
	tr := newTestTree(Column(Text("a"), Text("b"), Text("c")))
	first, second, third := tr.root.children[0], tr.root.children[1], tr.root.children[2]

	tr.setRoot(Column(Text("a")))

	if len(tr.root.children) != 1 || tr.root.children[0] != first {
		t.Fatal("first child node was not kept")
	}
	if !second.disposed || !third.disposed {
		t.Errorf("disposed: second=%v third=%v, want both", second.disposed, third.disposed)
	}
}
