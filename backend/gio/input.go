package gio

import (
	"gioui.org/io/pointer"

	"github.com/c-airr/flit/ui"
)

// pointerConverter turns Gio pointer events into ui.PointerEvents. v1
// clicks with the primary (left) button only, so it remembers whether that
// button is down to pair each Release with its Press.
type pointerConverter struct {
	primaryDown bool
}

// convert returns the ui event for e, with the position in dp, and false
// for events ui does not use.
func (c *pointerConverter) convert(e pointer.Event, pxPerDp float32) (ui.PointerEvent, bool) {
	pos := ui.Point{X: e.Position.X / pxPerDp, Y: e.Position.Y / pxPerDp}
	switch e.Kind {
	case pointer.Press:
		// Buttons holds every button that is down, so a second button
		// pressed while the primary is held shows up here too.
		if c.primaryDown || !e.Buttons.Contain(pointer.ButtonPrimary) {
			return ui.PointerEvent{}, false
		}
		c.primaryDown = true
		return ui.PointerEvent{Kind: ui.Press, Pos: pos}, true
	case pointer.Release:
		if !c.primaryDown || e.Buttons.Contain(pointer.ButtonPrimary) {
			return ui.PointerEvent{}, false
		}
		c.primaryDown = false
		return ui.PointerEvent{Kind: ui.Release, Pos: pos}, true
	case pointer.Move, pointer.Drag, pointer.Enter:
		return ui.PointerEvent{Kind: ui.Move, Pos: pos}, true
	case pointer.Leave, pointer.Cancel:
		// Cancel means the system took the pointer away (e.g. the window
		// lost focus): no release will come for the press.
		if e.Kind == pointer.Cancel {
			c.primaryDown = false
		}
		return ui.PointerEvent{Kind: ui.Leave, Pos: pos}, true
	}
	return ui.PointerEvent{}, false
}

// gioCursor maps a ui cursor to Gio's; "not set" is the default arrow.
func gioCursor(c ui.Cursor) pointer.Cursor {
	switch c {
	case ui.CursorPointer:
		return pointer.CursorPointer
	case ui.CursorText:
		return pointer.CursorText
	case ui.CursorCrosshair:
		return pointer.CursorCrosshair
	case ui.CursorGrab:
		return pointer.CursorGrab
	case ui.CursorNotAllowed:
		return pointer.CursorNotAllowed
	}
	return pointer.CursorDefault
}
