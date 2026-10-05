package ui

import "testing"

// All expected sizes come from the fake measurer: 8 px per character,
// 16 px per line.

// layoutUnder mounts w on its own and lays it out under c.
func layoutUnder(w Widget, c Constraints) *node {
	e := testEnv()
	n := mount(w, nil, e)
	layoutNode(n, e, c)
	return n
}

func TestTextLayout(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		c        Constraints
		want     Size
		wantMaxW float32
	}{
		{"empty text is one line high", "", Constraints{0, 200, 0, 100}, Size{W: 0, H: 16}, 200},
		{"measured", "Clicks", Constraints{0, 200, 0, 100}, Size{W: 48, H: 16}, 200},
		{"max width reaches the measurer", "abcdef", Constraints{0, 20, 0, 100}, Size{W: 16, H: 48}, 20},
		{"tight constraints win", "Clicks", Tight(Size{W: 200, H: 100}), Size{W: 200, H: 100}, 200},
		{"unbounded width means max width 0", "Clicks", Constraints{0, Inf, 0, Inf}, Size{W: 48, H: 16}, 0},
	}
	for _, tt := range tests {
		n := layoutUnder(Text(tt.text), tt.c)
		if n.size != tt.want {
			t.Errorf("%s: size = %+v, want %+v", tt.name, n.size, tt.want)
		}
		// n.state has type any; comparing it with a float32 is true only
		// when it holds a float32 with the same value.
		if n.state != tt.wantMaxW {
			t.Errorf("%s: state = %v (%T), want float32 %v", tt.name, n.state, n.state, tt.wantMaxW)
		}
	}
}

func TestPaddingAsRootIsTight(t *testing.T) {
	tr := newTestTree(Padding(All(16), Text("Clicks")))
	tr.layout(Size{W: 200, H: 100})

	text := tr.root.children[0]
	if tr.root.size != (Size{W: 200, H: 100}) {
		t.Errorf("padding size = %+v, want {200 100}", tr.root.size)
	}
	if text.offset != (Point{X: 16, Y: 16}) || text.size != (Size{W: 168, H: 68}) {
		t.Errorf("text offset=%+v size=%+v, want {16 16} and {168 68}", text.offset, text.size)
	}
}

func TestPaddingLoose(t *testing.T) {
	in := Insets{Top: 1, Right: 2, Bottom: 3, Left: 4}
	n := layoutUnder(Padding(in, Text("ab")), Constraints{0, 200, 0, 100})

	text := n.children[0]
	if text.offset != (Point{X: 4, Y: 1}) {
		t.Errorf("text offset = %+v, want {4 1} (left, top)", text.offset)
	}
	if n.size != (Size{W: 16 + 6, H: 16 + 4}) {
		t.Errorf("padding size = %+v, want {22 20}", n.size)
	}
}

func TestPaddingWithoutChild(t *testing.T) {
	n := layoutUnder(Padding(All(8), nil), Constraints{0, 200, 0, 100})
	if n.size != (Size{W: 16, H: 16}) {
		t.Errorf("size = %+v, want {16 16}", n.size)
	}
}

func TestContainerLayout(t *testing.T) {
	tests := []struct {
		name      string
		w         ContainerWidget
		c         Constraints
		want      Size
		wantChild Size
	}{
		{"no child takes the max", Container(nil), Constraints{0, 200, 0, 100}, Size{W: 200, H: 100}, Size{}},
		{"no child, unbounded: zero, never Inf", Container(nil), Constraints{0, Inf, 0, Inf}, Size{}, Size{}},
		{"no child, unbounded: falls back to min", Container(nil), Constraints{10, Inf, 20, Inf}, Size{W: 10, H: 20}, Size{}},
		{"child size when loose", Container(Text("x")), Constraints{0, 200, 0, 100}, Size{W: 8, H: 16}, Size{W: 8, H: 16}},
		{"width and height are tight for the child", Container(Text("x")).Width(50).Height(30),
			Constraints{0, 200, 0, 100}, Size{W: 50, H: 30}, Size{W: 50, H: 30}},
		{"width is clamped into constraints", Container(nil).Width(500), Constraints{0, 200, 0, 100}, Size{W: 200, H: 100}, Size{}},
	}
	for _, tt := range tests {
		n := layoutUnder(tt.w, tt.c)
		if n.size != tt.want {
			t.Errorf("%s: size = %+v, want %+v", tt.name, n.size, tt.want)
		}
		if len(n.children) == 1 && n.children[0].size != tt.wantChild {
			t.Errorf("%s: child size = %+v, want %+v", tt.name, n.children[0].size, tt.wantChild)
		}
	}
}

func TestContainerAsRootFillsViewport(t *testing.T) {
	tr := newTestTree(Container(nil))
	tr.layout(Size{W: 200, H: 100})
	if tr.root.size != (Size{W: 200, H: 100}) {
		t.Errorf("size = %+v, want {200 100}", tr.root.size)
	}
}

func TestBuildPassesConstraintsThrough(t *testing.T) {
	tr := newTestTree(Build(func(*Ctx) Widget { return Text("Clicks") }))
	tr.layout(Size{W: 200, H: 100})

	if tr.root.size != (Size{W: 200, H: 100}) || tr.root.children[0].size != (Size{W: 200, H: 100}) {
		t.Errorf("build=%+v text=%+v, want both {200 100}", tr.root.size, tr.root.children[0].size)
	}
}
