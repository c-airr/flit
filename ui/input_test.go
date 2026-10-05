package ui

import (
	"slices"
	"strconv"
	"testing"

	"github.com/c-airr/flit/draw"
	"github.com/c-airr/flit/internal/fakebackend"
)

// pad is a 100x40 Pressable that shows its label and state as text,
// e.g. "a:hover", and counts its clicks.
func pad(label string, clicks *int) PressableWidget {
	return Pressable(func(_ *Ctx, s PressState) Widget {
		state := "idle"
		if s.Pressed {
			state = "pressed"
		} else if s.Hovered {
			state = "hover"
		}
		return Container(Text(label + ":" + state)).Width(100).Height(40)
	}).OnClick(func() { *clicks++ })
}

// texts runs a frame and returns the strings of all Text commands.
func texts(a *App) []string {
	var out draw.List
	a.Frame(Size{W: 200, H: 100}, fakebackend.Measurer{}, &out)
	var got []string
	for _, c := range out.Cmds {
		if c.Kind == draw.Text {
			got = append(got, out.TextOf(c))
		}
	}
	return got
}

func at(k PointerKind, x, y float32) PointerEvent {
	return PointerEvent{Kind: k, Pos: Point{X: x, Y: y}}
}

// click sends a press and a release at the same point.
func click(a *App, x, y float32) {
	a.Pointer(at(Press, x, y))
	a.Pointer(at(Release, x, y))
}

func checkTexts(t *testing.T, a *App, want ...string) {
	t.Helper()
	if got := texts(a); !slices.Equal(got, want) {
		t.Errorf("texts = %q, want %q", got, want)
	}
}

// Root for most tests: pad "a" at y 0-40, pad "b" at y 40-80, both x 0-100.
func twoPads(ca, cb *int) *App {
	a := NewApp(Column(pad("a", ca), pad("b", cb)))
	texts(a) // first frame: builds and lays out the tree
	return a
}

func TestHoverAndLeave(t *testing.T) {
	var ca, cb int
	a := twoPads(&ca, &cb)

	a.Pointer(at(Move, 50, 20))
	checkTexts(t, a, "a:hover", "b:idle")
	if c := a.Cursor(); c != CursorPointer {
		t.Errorf("over a: Cursor() = %d, want CursorPointer", c)
	}

	a.Pointer(at(Move, 50, 60))
	checkTexts(t, a, "a:idle", "b:hover")

	a.Pointer(at(Move, 150, 20)) // right of the pads
	checkTexts(t, a, "a:idle", "b:idle")
	if c := a.Cursor(); c != CursorDefault {
		t.Errorf("over nothing: Cursor() = %d, want CursorDefault", c)
	}

	a.Pointer(at(Move, 50, 20))
	a.Pointer(at(Leave, 0, 0))
	checkTexts(t, a, "a:idle", "b:idle")
	if c := a.Cursor(); c != CursorDefault {
		t.Errorf("after Leave: Cursor() = %d, want CursorDefault", c)
	}
}

func TestClick(t *testing.T) {
	var ca, cb int
	a := twoPads(&ca, &cb)

	a.Pointer(at(Move, 50, 20))
	a.Pointer(at(Press, 50, 20))
	checkTexts(t, a, "a:pressed", "b:idle")

	a.Pointer(at(Release, 50, 20))
	if ca != 1 || cb != 0 {
		t.Errorf("clicks a=%d b=%d, want 1 and 0", ca, cb)
	}
	checkTexts(t, a, "a:hover", "b:idle")
}

func TestReleaseElsewhereIsNotAClick(t *testing.T) {
	var ca, cb int
	a := twoPads(&ca, &cb)

	a.Pointer(at(Press, 50, 20))
	a.Pointer(at(Release, 50, 60)) // on b
	a.Pointer(at(Press, 50, 20))
	a.Pointer(at(Release, 150, 20)) // on nothing

	if ca != 0 || cb != 0 {
		t.Errorf("clicks a=%d b=%d, want none", ca, cb)
	}
}

func TestDeepestPressableWins(t *testing.T) {
	var outerClicks, innerClicks int
	inner := Pressable(func(*Ctx, PressState) Widget {
		return Container(nil).Width(50).Height(20)
	}).OnClick(func() { innerClicks++ })
	outer := Pressable(func(*Ctx, PressState) Widget {
		return Padding(All(10), inner)
	}).OnClick(func() { outerClicks++ })
	a := NewApp(Column(outer))
	texts(a)

	click(a, 20, 15)
	if innerClicks != 1 || outerClicks != 0 {
		t.Errorf("click on inner: inner=%d outer=%d, want 1 and 0", innerClicks, outerClicks)
	}
	click(a, 5, 5)
	if innerClicks != 1 || outerClicks != 1 {
		t.Errorf("click on outer's padding: inner=%d outer=%d, want 1 and 1", innerClicks, outerClicks)
	}
}

func TestClipHidesPressable(t *testing.T) {
	// The pad sits at y 80-120 inside a 50 high box, so (50,90) is on the
	// pad but outside the box.
	tree := func(clip bool, clicks *int) Widget {
		box := Container(Column(Container(nil).Height(80), pad("a", clicks))).Height(50)
		if clip {
			box = box.Clip()
		}
		return Column(box)
	}

	var clipped, unclipped int
	a := NewApp(tree(true, &clipped))
	texts(a)
	click(a, 50, 90)
	b := NewApp(tree(false, &unclipped))
	texts(b)
	click(b, 50, 90)

	if clipped != 0 || unclipped != 1 {
		t.Errorf("clicks: clipped=%d unclipped=%d, want 0 and 1", clipped, unclipped)
	}
}

func TestPointerBeforeFirstFrame(t *testing.T) {
	var ca int
	a := NewApp(Column(pad("a", &ca)))
	a.Pointer(at(Move, 50, 20))
	click(a, 50, 20)
	if ca != 0 {
		t.Errorf("clicks = %d, want 0 before the first frame", ca)
	}
	texts(a)
}

func TestDisposedWhilePressed(t *testing.T) {
	var ca int
	show := NewSignal(true)
	a := NewApp(Build(func(*Ctx) Widget {
		if show.Get() {
			return Column(pad("a", &ca))
		}
		return Column()
	}))
	texts(a)

	a.Pointer(at(Move, 50, 20))
	a.Pointer(at(Press, 50, 20))
	show.Set(false)
	texts(a)
	a.Pointer(at(Release, 50, 20))

	if ca != 0 {
		t.Errorf("clicks = %d, want 0: the pad was removed before the release", ca)
	}
	if c := a.Cursor(); c != CursorDefault {
		t.Errorf("Cursor() = %d, want CursorDefault", c)
	}
	a.Pointer(at(Move, 50, 20))
	if c := a.Cursor(); c != CursorDefault {
		t.Errorf("after a move: Cursor() = %d, want CursorDefault", c)
	}
}

func TestClickHandlerSetsSignal(t *testing.T) {
	count := NewSignal(0)
	a := NewApp(Column(Pressable(func(*Ctx, PressState) Widget {
		return Container(Build(func(*Ctx) Widget {
			return Text(strconv.Itoa(count.Get()))
		})).Width(100).Height(40)
	}).OnClick(func() { count.Update(func(v int) int { return v + 1 }) })))
	texts(a)

	click(a, 50, 20)

	if !a.NeedsFrame() {
		t.Error("after the click NeedsFrame() = false, want true")
	}
	checkTexts(t, a, "1")
}

func TestPressableCursorOption(t *testing.T) {
	a := NewApp(Column(Pressable(func(*Ctx, PressState) Widget {
		return Container(nil).Width(100).Height(40)
	}).Cursor(CursorText)))
	texts(a)

	a.Pointer(at(Move, 50, 20))

	if c := a.Cursor(); c != CursorText {
		t.Errorf("Cursor() = %d, want CursorText", c)
	}
}
