package ui

import (
	"strings"
	"testing"

	"github.com/c-airr/flit/draw"
)

// firstLine is the first command of a painted list: the button's background.
func firstLine(list string) string {
	line, _, _ := strings.Cut(list, "\n")
	return line
}

func TestButtonDefaultLook(t *testing.T) {
	clicks := 0
	a := NewApp(Row(Button("+", func() { clicks++ })))

	// "+" is 8x16; padding is 2 and 4 grid units (8, 16): 40x32.
	want := `FillRect x=0 y=0 w=40 h=32 color=#4f46e5ff radius=8
Text x=16 y=8 w=0 color=#ffffffff size=14 weight=600 "+"
`
	if got := frame(a); got != want {
		t.Errorf("idle:\n got:\n%s\nwant:\n%s", got, want)
	}

	a.Pointer(at(Move, 10, 10))
	if got := firstLine(frame(a)); !strings.Contains(got, "color=#4338caff") {
		t.Errorf("hovered background: %s, want PrimaryHover #4338caff", got)
	}
	a.Pointer(at(Press, 10, 10))
	if got := firstLine(frame(a)); !strings.Contains(got, "color=#3730a3ff") {
		t.Errorf("pressed background: %s, want PrimaryPressed #3730a3ff", got)
	}
	a.Pointer(at(Release, 10, 10))
	if clicks != 1 {
		t.Errorf("clicks = %d, want 1", clicks)
	}
	if c := a.Cursor(); c != CursorPointer {
		t.Errorf("Cursor() = %d, want CursorPointer", c)
	}
}

func TestButtonOptions(t *testing.T) {
	b := Button("Go", nil).
		Background(draw.Hex(0x000000)).
		TextColor(draw.Hex(0xFF0000)).
		Radius(2).
		Padding(All(4)).
		FontSize(18).
		Weight(400)
	a := NewApp(Row(b))

	want := `FillRect x=0 y=0 w=24 h=24 color=#000000ff radius=2
Text x=4 y=4 w=0 color=#ff0000ff size=18 weight=400 "Go"
`
	if got := frame(a); got != want {
		t.Errorf("\n got:\n%s\nwant:\n%s", got, want)
	}
	click(a, 10, 10) // a nil onClick must not panic
}

func TestButtonCustomBackgroundHover(t *testing.T) {
	a := NewApp(Row(Button("x", nil).Background(draw.Hex(0x000000))))
	frame(a)

	a.Pointer(at(Move, 5, 5))
	if got := firstLine(frame(a)); !strings.Contains(got, "color=#000000ff") {
		t.Errorf("hovered: %s, want the set background, not the theme's hover color", got)
	}
}
