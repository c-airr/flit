package ui

import (
	"testing"

	"github.com/c-airr/flit/draw"
)

// paintRoot lays out root in a 200x100 viewport and paints it.
func paintRoot(root Widget) *draw.List {
	tr := newTestTree(root)
	tr.layout(Size{W: 200, H: 100})
	var out draw.List
	tr.paint(&out)
	return &out
}

func checkPaint(t *testing.T, root Widget, want string) {
	t.Helper() // failures point at the caller's line, not this one
	if got := paintRoot(root).String(); got != want {
		t.Errorf("paint mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPaintCard(t *testing.T) {
	root := Container(Padding(All(16), Column(Text("Clicks"), Text("5").Size(20)).Gap(8))).
		Background(DefaultTheme().Surface).
		Radius(8)
	checkPaint(t, root, `FillRect x=0 y=0 w=200 h=100 color=#ffffffff radius=8
Text x=16 y=16 w=168 color=#18181bff size=14 weight=400 "Clicks"
Text x=16 y=40 w=168 color=#18181bff size=20 weight=400 "5"
`)
}

func TestPaintClipAndBorder(t *testing.T) {
	root := Container(Text("x")).Radius(4).Clip().Border(DefaultTheme().Outline, 1)
	checkPaint(t, root, `PushClip x=0 y=0 w=200 h=100 radius=4
Text x=0 y=0 w=200 color=#18181bff size=14 weight=400 "x"
PopClip
StrokeRect x=0 y=0 w=200 h=100 color=#e4e4e7ff radius=4 width=1
`)
}

func TestPaintTextOptions(t *testing.T) {
	root := Row(Text("a").Color(draw.Hex(0x4F46E5)).Weight(600))
	checkPaint(t, root, `Text x=0 y=0 w=0 color=#4f46e5ff size=14 weight=600 "a"
`)
}

func TestPaintBuildPaintsItsChild(t *testing.T) {
	root := Build(func(*Ctx) Widget { return Padding(All(4), Row(Text("a"))) })
	checkPaint(t, root, `Text x=4 y=4 w=0 color=#18181bff size=14 weight=400 "a"
`)
}

func TestPaintTransparentBackgroundEmitsNothing(t *testing.T) {
	checkPaint(t, Container(nil), "")
}

func TestPaintResetsTheList(t *testing.T) {
	tr := newTestTree(Row())
	tr.layout(Size{W: 200, H: 100})
	var out draw.List
	out.FillRect(draw.Rect{W: 1, H: 1}, draw.Hex(0), 0) // left over from an earlier frame

	tr.paint(&out)

	if got := out.String(); got != "" {
		t.Errorf("list after paint = %q, want empty", got)
	}
	if out.Version != draw.FormatVersion {
		t.Errorf("Version = %d, want %d", out.Version, draw.FormatVersion)
	}
}
