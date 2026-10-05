package ui

// Cursor is a mouse cursor shape. v1 has the system cursors only; cursors
// made from images come after v1. The zero value means "not set".
type Cursor uint8

const (
	CursorDefault    Cursor = iota + 1 // the usual arrow
	CursorPointer                      // a hand, for things you can click
	CursorText                         // an I-beam, for text
	CursorCrosshair                    // a cross, for precise picking
	CursorGrab                         // an open hand, for things you can drag
	CursorNotAllowed                   // a crossed circle, for disabled things
)
