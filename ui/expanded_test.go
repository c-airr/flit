package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func childWidths(n *node) []float32 {
	var out []float32
	for _, c := range n.children {
		out = append(out, c.size.W)
	}
	return out
}

func childXs(n *node) []float32 {
	var out []float32
	for _, c := range n.children {
		out = append(out, c.offset.X)
	}
	return out
}

// Under Constraints{0, 200, 0, 100}; Text("ab") is 16x16.
func TestExpandedSplits(t *testing.T) {
	// 200 * 2/3 in float32. The last flex child gets what is left, 200 - that.
	twoThirds := float32(400.0 / 3)
	tests := []struct {
		name       string
		w          FlexWidget
		wantWidths []float32
		wantXs     []float32
	}{
		{"one Expanded takes the rest", Row(Text("ab"), Expanded(nil)),
			[]float32{16, 184}, []float32{0, 16}},
		{"2:1", Row(Expanded(nil).Flex(2), Expanded(nil)),
			[]float32{twoThirds, 200 - twoThirds}, []float32{0, twoThirds}},
		{"gaps come off first", Row(Text("ab"), Expanded(nil)).Gap(10),
			[]float32{16, 174}, []float32{0, 26}},
		{"Spacer pushes to the end", Row(Text("ab"), Spacer(), Text("ab")),
			[]float32{16, 168, 16}, []float32{0, 16, 184}},
		{"Flex(0) counts as 1", Row(Expanded(nil).Flex(0), Expanded(nil)),
			[]float32{100, 100}, []float32{0, 100}},
		{"MainAlign has no effect", Row(Text("ab"), Expanded(nil)).MainAlign(End),
			[]float32{16, 184}, []float32{0, 16}},
	}
	for _, tt := range tests {
		n := layoutUnder(tt.w, Constraints{0, 200, 0, 100})
		if got := childWidths(n); !slices.Equal(got, tt.wantWidths) {
			t.Errorf("%s: widths = %v, want %v", tt.name, got, tt.wantWidths)
		}
		if got := childXs(n); !slices.Equal(got, tt.wantXs) {
			t.Errorf("%s: x = %v, want %v", tt.name, got, tt.wantXs)
		}
		if n.size.W != 200 {
			t.Errorf("%s: row width = %v, want 200", tt.name, n.size.W)
		}
	}
}

func TestExpandedPartsSumExactly(t *testing.T) {
	// 100 / 3 is not exact in float32; the last part absorbs the error.
	n := layoutUnder(Row(Expanded(nil), Expanded(nil), Expanded(nil)), Constraints{0, 100, 0, 100})
	w := childWidths(n)
	if sum := w[0] + w[1] + w[2]; sum != 100 {
		t.Errorf("widths %v sum to %v, want exactly 100", w, sum)
	}
}

func TestExpandedOverflow(t *testing.T) {
	// The first Text alone is 200 wide: nothing is left for the Expanded.
	n := layoutUnder(Row(Text(strings.Repeat("a", 25)), Expanded(nil), Text("ab")), Constraints{0, 200, 0, 100})
	if got := n.children[1].size.W; got != 0 {
		t.Errorf("Expanded width = %v, want 0 (never negative or NaN)", got)
	}
}

func TestExpandedSqueezesWideChild(t *testing.T) {
	n := layoutUnder(Row(Expanded(Text("abcdefghijklmnopqrstuvwxyz")), Text("ab")), Constraints{0, 200, 0, 100})
	exp, last := n.children[0], n.children[1]
	if exp.size.W != 184 || exp.children[0].size.W != 184 {
		t.Errorf("Expanded %v wide, its Text %v; want both 184", exp.size.W, exp.children[0].size.W)
	}
	if last.offset.X != 184 {
		t.Errorf("last Text at x = %v, want 184", last.offset.X)
	}
}

func TestExpandedThroughComposite(t *testing.T) {
	n := layoutUnder(Row(Build(func(*Ctx) Widget { return Expanded(nil) }), Text("ab")), Constraints{0, 200, 0, 100})
	if got := childWidths(n); !slices.Equal(got, []float32{184, 16}) {
		t.Errorf("widths = %v, want [184 16]", got)
	}
}

func TestExpandedUnboundedMain(t *testing.T) {
	n := layoutUnder(Row(Text("ab"), Expanded(Text("abcd"))), Constraints{0, Inf, 0, 100})
	if got := childWidths(n); !slices.Equal(got, []float32{16, 32}) {
		t.Errorf("widths = %v, want [16 32] (laid out like ordinary children)", got)
	}
}

func TestExpandedInColumnInRow(t *testing.T) {
	// The Row bounds the Column's height, so the Column can share it out.
	n := layoutUnder(Row(Column(Text("ab"), Expanded(nil))), Constraints{0, 200, 0, 100})
	exp := n.children[0].children[1]
	if exp.size.H != 84 || exp.offset.Y != 16 {
		t.Errorf("Expanded height %v at y %v, want 84 at 16", exp.size.H, exp.offset.Y)
	}
}

func TestExpandedOutsideFlex(t *testing.T) {
	if got := layoutUnder(Expanded(Text("ab")), Constraints{0, 200, 0, 100}).size; got != (Size{W: 16, H: 16}) {
		t.Errorf("Expanded(Text) alone = %+v, want its child's {16 16}", got)
	}
	if got := layoutUnder(Expanded(nil), Constraints{0, 200, 0, 100}).size; got != (Size{}) {
		t.Errorf("empty Expanded alone = %+v, want {0 0}", got)
	}
}

// captureWarnings turns debug warnings on or off for one test and records
// what would be logged. t.Cleanup restores the package variables after.
func captureWarnings(t *testing.T, enabled bool) *[]string {
	var got []string
	oldEnabled, oldLogf := debugEnabled, debugLogf
	debugEnabled = enabled
	debugLogf = func(format string, args ...any) { got = append(got, fmt.Sprintf(format, args...)) }
	t.Cleanup(func() { debugEnabled, debugLogf = oldEnabled, oldLogf })
	return &got
}

func TestDebugWarnings(t *testing.T) {
	notInFlex := "flit: Expanded is not a child of a Row or Column; it lays out like its child"
	unbounded := "flit: Expanded in a Row/Column with an unbounded main axis; it lays out like an ordinary child"

	got := captureWarnings(t, true)
	tr := newTestTree(Padding(All(0), Expanded(nil)))
	tr.layout(Size{W: 200, H: 100})
	tr.layout(Size{W: 200, H: 100})
	if !slices.Equal(*got, []string{notInFlex}) {
		t.Errorf("outside a flex, two layouts: logged %q, want it once", *got)
	}

	*got = nil
	tr = newTestTree(Row(Expanded(nil)))
	layoutNode(tr.root, &tr.env, Constraints{0, Inf, 0, 100})
	layoutNode(tr.root, &tr.env, Constraints{0, Inf, 0, 100})
	if !slices.Equal(*got, []string{unbounded}) {
		t.Errorf("unbounded main axis, two layouts: logged %q, want it once", *got)
	}
}

func TestDebugWarningsOff(t *testing.T) {
	got := captureWarnings(t, false)
	newTestTree(Padding(All(0), Expanded(nil))).layout(Size{W: 200, H: 100})
	layoutUnder(Row(Expanded(nil)), Constraints{0, Inf, 0, 100})
	if len(*got) != 0 {
		t.Errorf("with debug off, logged %q, want nothing", *got)
	}
}
