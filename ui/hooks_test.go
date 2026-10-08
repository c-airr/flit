package ui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// panicMessage runs fn and returns the value it panicked with, as text,
// or "" if it did not panic. recover only works inside a deferred
// function; it stops the panic and returns its value, a bit like catch(...).
func panicMessage(fn func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprint(r)
		}
	}()
	fn()
	return ""
}

func TestUseSignalSurvivesRebuild(t *testing.T) {
	var count *Signal[int]
	root := Build(func(ctx *Ctx) Widget {
		count = UseSignal(ctx, 0)
		return Text(strconv.Itoa(count.Get()))
	})
	tr := newTestTree(root)
	first := count

	count.Set(3)
	tr.rebuildDirty()
	if count != first {
		t.Fatal("after a rebuild UseSignal returned a different signal")
	}
	if got := textOf(tr.root.children[0]); got != "3" {
		t.Errorf("text after Set(3) = %q, want %q", got, "3")
	}

	tr.setRoot(root) // the parent rebuilds: the same node, its build reruns
	if count != first || count.Get() != 3 {
		t.Errorf("after a parent rebuild: same signal %v, value %d; want true, 3", count == first, count.Get())
	}
}

func TestRememberInitRunsOnce(t *testing.T) {
	inits := 0
	tick := NewSignal(0)
	var got int
	tr := newTestTree(Build(func(ctx *Ctx) Widget {
		tick.Get()
		got = Remember(ctx, func() int { inits++; return 7 })
		return nil
	}))
	for i := range 3 {
		tick.Set(i + 1)
		tr.rebuildDirty()
	}
	if inits != 1 || got != 7 {
		t.Errorf("init ran %d times, value %d; want 1 and 7", inits, got)
	}
}

func TestStateDroppedWhenNodeReplaced(t *testing.T) {
	show := NewSignal(true)
	var inner *Signal[int]
	component := Build(func(ctx *Ctx) Widget {
		inner = UseSignal(ctx, 0)
		return Text(strconv.Itoa(inner.Get()))
	})
	tr := newTestTree(Build(func(*Ctx) Widget {
		if show.Get() {
			return component
		}
		return Text("x")
	}))

	inner.Set(5)
	tr.rebuildDirty()
	show.Set(false) // a Text takes the component's place: its node is disposed
	tr.rebuildDirty()
	show.Set(true) // a fresh component node with fresh state
	tr.rebuildDirty()

	if got := inner.Get(); got != 0 {
		t.Errorf("state after the node was replaced = %d, want 0 (a new node starts over)", got)
	}
}

func TestNestedBuildsHaveOwnSlots(t *testing.T) {
	tick := NewSignal(0)
	var parent *Signal[string]
	var child *Signal[int]
	tr := newTestTree(Build(func(ctx *Ctx) Widget {
		tick.Get()
		parent = UseSignal(ctx, "p")
		return Build(func(ctx *Ctx) Widget {
			child = UseSignal(ctx, 1)
			return nil
		})
	}))
	for i := range 2 {
		tick.Set(i + 1)
		tr.rebuildDirty()
	}
	if parent.Get() != "p" || child.Get() != 1 {
		t.Errorf("parent %q, child %d; want \"p\" and 1", parent.Get(), child.Get())
	}
}

func TestHookInsidePressable(t *testing.T) {
	seen := map[*Signal[int]]bool{}
	builds := 0
	a := NewApp(Column(Pressable(func(ctx *Ctx, _ PressState) Widget {
		builds++
		seen[UseSignal(ctx, 0)] = true
		return Container(nil).Width(100).Height(40)
	})))
	texts(a)
	a.Pointer(at(Move, 50, 20)) // hover: the Pressable rebuilds
	texts(a)
	a.Pointer(at(Move, 150, 20)) // unhover: it rebuilds again
	texts(a)

	if builds < 3 || len(seen) != 1 {
		t.Errorf("%d builds saw %d different signals, want >= 3 builds and 1 signal", builds, len(seen))
	}
}

func TestHookPanics(t *testing.T) {
	signals := func(n int) func(ctx *Ctx) {
		return func(ctx *Ctx) {
			for range n {
				UseSignal(ctx, 0)
			}
		}
	}
	tests := []struct {
		name          string
		before, after func(ctx *Ctx)
		want          string
	}{
		{"type changed",
			func(ctx *Ctx) { UseSignal(ctx, 0) },
			func(ctx *Ctx) { UseSignal(ctx, "x") },
			"ui: hook #0 was UseSignal[int] in the previous build and is UseSignal[string] now; call hooks in the same order every build, never inside if or for"},
		{"hook changed",
			func(ctx *Ctx) { UseSignal(ctx, 0) },
			func(ctx *Ctx) { Remember(ctx, func() int { return 0 }) },
			"ui: hook #0 was UseSignal[int] in the previous build and is Remember[int] now; call hooks in the same order every build, never inside if or for"},
		{"more calls", signals(2), signals(3),
			"ui: hook #2 is new in this build (the previous build made 2 hook calls); call hooks in the same order every build, never inside if or for"},
		{"fewer calls", signals(3), signals(2),
			"ui: this build made 2 hook calls, the previous build made 3; call hooks in the same order every build, never inside if or for"},
	}
	for _, tt := range tests {
		flip := NewSignal(false)
		tr := newTestTree(Build(func(ctx *Ctx) Widget {
			if flip.Get() {
				tt.after(ctx)
			} else {
				tt.before(ctx)
			}
			return nil
		}))
		flip.Set(true)
		if got := panicMessage(tr.rebuildDirty); got != tt.want {
			t.Errorf("%s:\n got: %q\nwant: %q", tt.name, got, tt.want)
		}
	}
}

func TestHookOutsideBuild(t *testing.T) {
	var saved *Ctx
	newTestTree(Build(func(ctx *Ctx) Widget {
		saved = ctx
		return nil
	}))

	want := "ui: UseSignal called outside a build; hooks only work while a Build or Pressable function runs"
	if got := panicMessage(func() { UseSignal(saved, 0) }); got != want {
		t.Errorf("UseSignal:\n got: %q\nwant: %q", got, want)
	}
	want = "ui: Remember called outside a build; hooks only work while a Build or Pressable function runs"
	if got := panicMessage(func() { Remember(saved, func() int { return 0 }) }); got != want {
		t.Errorf("Remember:\n got: %q\nwant: %q", got, want)
	}
}

func TestBuildPanicSkipsCountCheck(t *testing.T) {
	flip := NewSignal(false)
	tr := newTestTree(Build(func(ctx *Ctx) Widget {
		if flip.Get() {
			panic("boom")
		}
		UseSignal(ctx, 0)
		return nil
	}))
	flip.Set(true)
	if got := panicMessage(tr.rebuildDirty); got != "boom" {
		t.Errorf("panic = %q, want the build's own %q", got, "boom")
	}
}

func TestHookPanicFromClick(t *testing.T) {
	extra := NewSignal(false)
	a := NewApp(Column(Build(func(ctx *Ctx) Widget {
		UseSignal(ctx, 0)
		if extra.Get() {
			UseSignal(ctx, 0)
		}
		return Pressable(func(*Ctx, PressState) Widget {
			return Container(nil).Width(100).Height(40)
		}).OnClick(func() { extra.Set(true) })
	})))
	texts(a)
	click(a, 50, 20)

	got := panicMessage(func() { texts(a) })
	if !strings.HasPrefix(got, "ui: hook #1 is new") {
		t.Errorf("panic from the frame after the click = %q, want the hook-order message", got)
	}
}
