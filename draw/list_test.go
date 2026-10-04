package draw

import "testing"

func TestListText(t *testing.T) {
	var l List
	l.Reset()
	style := FontStyle{Size: 14, Weight: 400}
	l.Text(Rect{}, "ab", RGBA{}, style)
	l.Text(Rect{}, "Zażółć", RGBA{}, style)

	if l.Version != FormatVersion {
		t.Errorf("Version = %d, want %d", l.Version, FormatVersion)
	}
	second := l.Cmds[1]
	// Offsets and lengths count bytes, not characters: "Zażółć" is 6 runes
	// but 10 bytes in UTF-8 (each of ż, ó, ł, ć takes two bytes).
	if second.TextOff != 2 || second.TextLen != 10 {
		t.Errorf("second text at off=%d len=%d, want off=2 len=10", second.TextOff, second.TextLen)
	}
	if got := l.TextOf(l.Cmds[0]); got != "ab" {
		t.Errorf("TextOf(first) = %q, want %q", got, "ab")
	}
	if got := l.TextOf(second); got != "Zażółć" {
		t.Errorf("TextOf(second) = %q, want %q", got, "Zażółć")
	}
}

func TestListResetKeepsCapacity(t *testing.T) {
	var l List
	l.Reset()
	l.FillRect(Rect{W: 10, H: 10}, RGBA{A: 255}, 0)
	l.Text(Rect{}, "hello", RGBA{}, FontStyle{})
	cmdCap, bufCap := cap(l.Cmds), cap(l.TextBuf)

	l.Reset()

	if len(l.Cmds) != 0 || len(l.TextBuf) != 0 {
		t.Errorf("after Reset len(Cmds)=%d len(TextBuf)=%d, want 0 and 0", len(l.Cmds), len(l.TextBuf))
	}
	if cap(l.Cmds) != cmdCap || cap(l.TextBuf) != bufCap {
		t.Errorf("Reset dropped capacity: Cmds %d -> %d, TextBuf %d -> %d",
			cmdCap, cap(l.Cmds), bufCap, cap(l.TextBuf))
	}
}

func TestNonTextCmdsHaveNoText(t *testing.T) {
	var l List
	l.Reset()
	l.Text(Rect{}, "abc", RGBA{}, FontStyle{})
	l.FillRect(Rect{}, RGBA{}, 0)
	if c := l.Cmds[1]; c.TextOff != 0 || c.TextLen != 0 {
		t.Errorf("FillRect has TextOff=%d TextLen=%d, want 0 and 0", c.TextOff, c.TextLen)
	}
}

func TestListString(t *testing.T) {
	var l List
	l.Reset()
	r := Rect{X: 0, Y: 0, W: 100, H: 40}
	l.FillRect(r, Hex(0x4F46E5), 8)
	l.StrokeRect(r, Hex(0xE4E4E7), 8, 1)
	l.Text(Rect{X: 16, Y: 12}, "Clicks", Hex(0xFFFFFF), FontStyle{Size: 14, Weight: 400})
	l.PushClip(r, 8)
	l.PopClip()

	want := `FillRect x=0 y=0 w=100 h=40 color=#4f46e5ff radius=8
StrokeRect x=0 y=0 w=100 h=40 color=#e4e4e7ff radius=8 width=1
Text x=16 y=12 w=0 color=#ffffffff size=14 weight=400 "Clicks"
PushClip x=0 y=0 w=100 h=40 radius=8
PopClip
`
	if got := l.String(); got != want {
		t.Errorf("String() mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}
