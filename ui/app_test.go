package ui

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c-airr/flit/draw"
	"github.com/c-airr/flit/internal/fakebackend"
)

// frame runs one frame of a in a 200x100 viewport and returns the painted list.
func frame(a *App) string {
	var out draw.List
	a.Frame(Size{W: 200, H: 100}, fakebackend.Measurer{}, &out)
	return out.String()
}

func TestAppFramePaintsRoot(t *testing.T) {
	a := NewApp(Container(nil).Background(draw.Hex(0x112233)))
	want := "FillRect x=0 y=0 w=200 h=100 color=#112233ff radius=0\n"
	if got := frame(a); got != want {
		t.Errorf("frame = %q, want %q", got, want)
	}
}

func TestAppNeedsFrame(t *testing.T) {
	a := NewApp(Text("x"))
	if !a.NeedsFrame() {
		t.Error("before the first frame NeedsFrame() = false, want true")
	}
	frame(a)
	if a.NeedsFrame() {
		t.Error("after a frame with nothing pending NeedsFrame() = true, want false")
	}
}

func TestAppFrameAfterSet(t *testing.T) {
	count := NewSignal(0)
	a := NewApp(Build(func(*Ctx) Widget { return Text(strconv.Itoa(count.Get())) }))
	frame(a)

	count.Set(7)
	if !a.NeedsFrame() {
		t.Error("after Set NeedsFrame() = false, want true")
	}
	if got := frame(a); !strings.Contains(got, `"7"`) {
		t.Errorf("frame after Set(7) = %q, want the text \"7\"", got)
	}
	if a.NeedsFrame() {
		t.Error("after the rebuild NeedsFrame() = true, want false")
	}
}

func TestPostRunsOnNextFrameAndWakes(t *testing.T) {
	a := NewApp(Text("x"))
	wakeups := 0
	a.SetWakeup(func() { wakeups++ })
	frame(a)

	ran := false
	a.Post(func() { ran = true })

	if wakeups != 1 || ran {
		t.Errorf("right after Post: wakeups=%d ran=%v, want 1 and false", wakeups, ran)
	}
	if !a.NeedsFrame() {
		t.Error("with a posted function waiting NeedsFrame() = false, want true")
	}
	frame(a)
	if !ran {
		t.Error("posted function did not run in the next frame")
	}
}

func TestPostFromManyGoroutines(t *testing.T) {
	a := NewApp(Text("x"))
	var wakeups atomic.Int32
	a.SetWakeup(func() { wakeups.Add(1) })

	n := 0 // touched only by posted functions, which run on this goroutine in Frame
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Post(func() { n++ })
		}()
	}
	wg.Wait()
	frame(a)

	if n != 100 || wakeups.Load() != 100 {
		t.Errorf("n=%d wakeups=%d, want 100 and 100", n, wakeups.Load())
	}
}

func TestAsyncDeliversOnFrame(t *testing.T) {
	a := NewApp(Text("x"))
	woken := make(chan struct{}, 1)
	a.SetWakeup(func() {
		select { // a non-blocking send: never stall the posting goroutine
		case woken <- struct{}{}:
		default:
		}
	})
	got := 0
	Async(a, func() int { return 42 }, func(v int) { got = v })

	select {
	case <-woken:
	case <-time.After(5 * time.Second):
		t.Fatal("Async never woke the app")
	}
	frame(a)
	if got != 42 {
		t.Errorf("done got %d, want 42", got)
	}
}

func TestSetThemeRebuilds(t *testing.T) {
	a := NewApp(Build(func(ctx *Ctx) Widget {
		return Container(nil).Background(ctx.Theme().Primary)
	}))
	if got := frame(a); !strings.Contains(got, "#4f46e5ff") {
		t.Fatalf("first frame = %q, want the default Primary", got)
	}

	th := DefaultTheme()
	th.Primary = draw.Hex(0x00FF00)
	a.SetTheme(th)

	if !a.NeedsFrame() {
		t.Error("after SetTheme NeedsFrame() = false, want true")
	}
	if got := frame(a); !strings.Contains(got, "#00ff00ff") {
		t.Errorf("frame after SetTheme = %q, want the new Primary", got)
	}
}

func TestFrameZeroViewport(t *testing.T) {
	a := NewApp(Container(Padding(All(16), Text("x"))).Background(DefaultTheme().Surface))
	var out draw.List
	a.Frame(Size{}, fakebackend.Measurer{}, &out)
	if got := out.String(); !strings.HasPrefix(got, "FillRect x=0 y=0 w=0 h=0 ") {
		t.Errorf("frame = %q, want a 0x0 FillRect first", got)
	}
}
