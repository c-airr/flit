package gio

import (
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/c-airr/flit/draw"
)

func TestStrokeInset(t *testing.T) {
	r, radius := strokeInset(draw.Rect{X: 0, Y: 0, W: 100, H: 40}, 8, 2)
	if r != (draw.Rect{X: 1, Y: 1, W: 98, H: 38}) || radius != 7 {
		t.Errorf("strokeInset = %+v, %v; want {1 1 98 38}, 7", r, radius)
	}
	if _, radius := strokeInset(draw.Rect{W: 10, H: 10}, 0.5, 2); radius != 0 {
		t.Errorf("radius = %v, want 0 (never negative)", radius)
	}
}

func testContext() layout.Context {
	return layout.Context{Ops: new(op.Ops), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
}

func TestRenderRejectsUnknownVersion(t *testing.T) {
	l := draw.List{Version: 99}
	if err := render(testContext(), newShaper(), &l); err == nil {
		t.Error("render accepted version 99, want an error")
	}
}

func TestRenderUnbalancedPop(t *testing.T) {
	var l draw.List
	l.Reset()
	l.PopClip()
	if err := render(testContext(), newShaper(), &l); err == nil {
		t.Error("render accepted a PopClip without a PushClip, want an error")
	}
}

func TestRenderLeftoverClip(t *testing.T) {
	var l draw.List
	l.Reset()
	l.PushClip(draw.Rect{W: 10, H: 10}, 0)
	l.FillRect(draw.Rect{W: 5, H: 5}, draw.Hex(0), 0)
	if err := render(testContext(), newShaper(), &l); err != nil {
		t.Errorf("render = %v, want nil (leftover clips are popped)", err)
	}
}
