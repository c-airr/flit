package gio

import (
	"log"
	"os"

	// Aliased because Run's parameter is called app, after the spec.
	gioapp "gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"

	"github.com/c-airr/flit/draw"
	"github.com/c-airr/flit/ui"
)

// Options configure the window.
type Options struct {
	Title         string  // window title; "" means "flit"
	Width, Height float32 // initial size in dp; 0 means 800 x 600
}

// Run opens the window and runs the app. It blocks and must be called from
// main (the main goroutine), because Gio's app.Main must own the main OS
// thread. When the window is closed, Run ends the process with os.Exit,
// so deferred calls in main do not run.
func Run(app *ui.App, opts Options) {
	if opts.Title == "" {
		opts.Title = "flit"
	}
	if opts.Width == 0 {
		opts.Width = 800
	}
	if opts.Height == 0 {
		opts.Height = 600
	}
	// The window loop runs in its own goroutine, which becomes flit's UI
	// goroutine; gioapp.Main keeps the main goroutine for the OS.
	go func() {
		w := new(gioapp.Window)
		w.Option(gioapp.Title(opts.Title), gioapp.Size(unit.Dp(opts.Width), unit.Dp(opts.Height)))
		if err := loop(w, app); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	gioapp.Main()
}

// pointerKinds are the Gio pointer events flit listens to.
const pointerKinds = pointer.Press | pointer.Release | pointer.Move | pointer.Drag |
	pointer.Enter | pointer.Leave | pointer.Cancel

// loop handles the window's events until it is closed.
func loop(w *gioapp.Window, app *ui.App) error {
	sh := newShaper()
	m := newMeasurer(sh)
	var (
		conv pointerConverter
		ops  op.Ops
		list draw.List
	)
	// tag identifies our input area to Gio. Any unique pointer will do.
	tag := new(int)
	// A method value: w.Invalidate bound to this w, callable later as a
	// plain func(). Invalidate is safe to call from any goroutine.
	app.SetWakeup(w.Invalidate)

	for {
		switch e := w.Event().(type) {
		case gioapp.DestroyEvent:
			return e.Err
		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)
			gtx.Metric.PxPerSp = gtx.Metric.PxPerDp // fonts are dp too (spec §7)
			s := gtx.Metric.PxPerDp
			m.setScale(s)

			// Pointer events that reached our input area since the last frame.
			for {
				ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointerKinds})
				if !ok {
					break
				}
				if pe, ok := ev.(pointer.Event); ok {
					if uev, ok := conv.convert(pe, s); ok {
						app.Pointer(uev)
					}
				}
			}

			viewport := ui.Size{W: float32(gtx.Constraints.Max.X) / s, H: float32(gtx.Constraints.Max.Y) / s}
			app.Frame(viewport, m, &list)
			if err := render(gtx, sh, &list); err != nil {
				return err
			}

			// One input area over the whole window: it receives the pointer
			// events for the next frame and sets the cursor.
			area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			event.Op(gtx.Ops, tag)
			gioCursor(app.Cursor()).Add(gtx.Ops)
			area.Pop()

			e.Frame(gtx.Ops)
			if app.NeedsFrame() {
				w.Invalidate()
			}
		}
	}
}
