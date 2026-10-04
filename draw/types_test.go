package draw

import (
	"testing"
	"unsafe"
)

// TestCmdLayout pins the binary layout of Cmd. A C/C++ backend mirrors this
// struct field by field, so any change here is a breaking change to the
// boundary and must bump FormatVersion.
func TestCmdLayout(t *testing.T) {
	var c Cmd
	if got := unsafe.Sizeof(c); got != 48 {
		t.Fatalf("Sizeof(Cmd) = %d, want 48", got)
	}
	offsets := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"Kind", unsafe.Offsetof(c.Kind), 0},
		{"Rect", unsafe.Offsetof(c.Rect), 4},
		{"Color", unsafe.Offsetof(c.Color), 20},
		{"Radius", unsafe.Offsetof(c.Radius), 24},
		{"Width", unsafe.Offsetof(c.Width), 28},
		{"TextOff", unsafe.Offsetof(c.TextOff), 32},
		{"TextLen", unsafe.Offsetof(c.TextLen), 36},
		{"Font", unsafe.Offsetof(c.Font), 40},
	}
	for _, o := range offsets {
		if o.got != o.want {
			t.Errorf("Offsetof(Cmd.%s) = %d, want %d", o.name, o.got, o.want)
		}
	}
}

func TestHex(t *testing.T) {
	got := Hex(0x4F46E5)
	want := RGBA{0x4F, 0x46, 0xE5, 0xFF}
	if got != want {
		t.Errorf("Hex(0x4F46E5) = %v, want %v", got, want)
	}
}

func TestKindString(t *testing.T) {
	tests := map[Kind]string{
		FillRect:   "FillRect",
		StrokeRect: "StrokeRect",
		Text:       "Text",
		PushClip:   "PushClip",
		PopClip:    "PopClip",
		Kind(0):    "Kind(0)",
	}
	for k, want := range tests {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", uint32(k), got, want)
		}
	}
}
