package gio

import (
	"testing"

	"gioui.org/f32"
	"gioui.org/io/pointer"

	"github.com/c-airr/flit/ui"
)

func TestConvertPointer(t *testing.T) {
	ev := func(k pointer.Kind, b pointer.Buttons) pointer.Event {
		return pointer.Event{Kind: k, Buttons: b, Position: f32.Pt(30, 60)}
	}
	at := ui.Point{X: 20, Y: 40} // 30,60 px at 1.5 px per dp
	steps := []struct {
		name string
		in   pointer.Event
		want ui.PointerEvent
		ok   bool
	}{
		{"secondary press", ev(pointer.Press, pointer.ButtonSecondary), ui.PointerEvent{}, false},
		{"secondary release", ev(pointer.Release, 0), ui.PointerEvent{}, false},
		{"primary press", ev(pointer.Press, pointer.ButtonPrimary), ui.PointerEvent{Kind: ui.Press, Pos: at}, true},
		{"secondary press while primary is down", ev(pointer.Press, pointer.ButtonPrimary|pointer.ButtonSecondary), ui.PointerEvent{}, false},
		{"secondary release while primary is down", ev(pointer.Release, pointer.ButtonPrimary), ui.PointerEvent{}, false},
		{"primary release", ev(pointer.Release, 0), ui.PointerEvent{Kind: ui.Release, Pos: at}, true},
		{"move", ev(pointer.Move, 0), ui.PointerEvent{Kind: ui.Move, Pos: at}, true},
		{"drag", ev(pointer.Drag, pointer.ButtonPrimary), ui.PointerEvent{Kind: ui.Move, Pos: at}, true},
		{"enter", ev(pointer.Enter, 0), ui.PointerEvent{Kind: ui.Move, Pos: at}, true},
		{"leave", ev(pointer.Leave, 0), ui.PointerEvent{Kind: ui.Leave, Pos: at}, true},
		{"scroll", ev(pointer.Scroll, 0), ui.PointerEvent{}, false},
		{"press before cancel", ev(pointer.Press, pointer.ButtonPrimary), ui.PointerEvent{Kind: ui.Press, Pos: at}, true},
		{"cancel", ev(pointer.Cancel, 0), ui.PointerEvent{Kind: ui.Leave, Pos: at}, true},
		{"release after cancel", ev(pointer.Release, 0), ui.PointerEvent{}, false},
	}
	// One converter for all steps: it remembers whether the primary button is down.
	var c pointerConverter
	for _, s := range steps {
		got, ok := c.convert(s.in, 1.5)
		if ok != s.ok || (ok && got != s.want) {
			t.Errorf("%s: got %+v, %v; want %+v, %v", s.name, got, ok, s.want, s.ok)
		}
	}
}

func TestGioCursor(t *testing.T) {
	tests := []struct {
		in   ui.Cursor
		want pointer.Cursor
	}{
		{0, pointer.CursorDefault},
		{ui.CursorDefault, pointer.CursorDefault},
		{ui.CursorPointer, pointer.CursorPointer},
		{ui.CursorText, pointer.CursorText},
		{ui.CursorCrosshair, pointer.CursorCrosshair},
		{ui.CursorGrab, pointer.CursorGrab},
		{ui.CursorNotAllowed, pointer.CursorNotAllowed},
	}
	for _, tt := range tests {
		if got := gioCursor(tt.in); got != tt.want {
			t.Errorf("gioCursor(%d) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
