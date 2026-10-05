package ui

import (
	"sync"

	"github.com/c-airr/flit/draw"
)

// App runs a widget tree: it rebuilds what changed, lays it out and paints
// it, one frame at a time. A backend (such as backend/gio) owns the window
// and drives the App through SetWakeup, Pointer, Frame and NeedsFrame.
//
// Threading: Post and Async may be called from any goroutine. Everything
// else, and all widget building, happens on the UI goroutine, the one the
// backend calls Frame from.
type App struct {
	root         Widget
	theme        Theme
	tree         *tree // created by the first Frame, which has a measurer
	themeChanged bool

	hover   *node // the Pressable under the pointer
	pressed *node // the Pressable a press started on

	// mu guards the fields below it, which other goroutines touch through
	// Post. sync.Mutex is like std::mutex; its zero value is unlocked and
	// ready to use, so it needs no constructor.
	mu     sync.Mutex
	posted []func()
	wakeup func()
}

// NewApp makes an app showing root with the default theme. root is usually
// a Build. Nothing is built until the first Frame.
func NewApp(root Widget) *App {
	return &App{root: root, theme: DefaultTheme()}
}

// SetTheme replaces the theme. After the first frame it rebuilds the whole
// tree in the next frame, so every build sees the new theme.
func (a *App) SetTheme(th Theme) {
	a.theme = th
	if a.tree != nil {
		a.themeChanged = true
	}
}

// SetWakeup sets the function Post calls to make the backend schedule a
// frame. It must be safe to call from any goroutine.
func (a *App) SetWakeup(fn func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.wakeup = fn
}

// Post runs fn on the UI goroutine at the start of the next frame and wakes
// the backend so that frame happens. It is safe to call from any goroutine
// and never blocks for long: it only appends to a queue. The queue is
// unbounded (spec §11).
func (a *App) Post(fn func()) {
	a.mu.Lock()
	a.posted = append(a.posted, fn)
	wake := a.wakeup
	a.mu.Unlock()
	// Called outside the lock, so a wakeup that calls back into the App
	// (or simply takes a while) cannot deadlock or hold up other posters.
	if wake != nil {
		wake()
	}
}

// Async runs work in a new goroutine and then done(result) on the UI
// goroutine, where it may safely Set signals.
//
// It is a function, not a method, because Go methods cannot have type
// parameters of their own.
func Async[T any](a *App, work func() T, done func(T)) {
	// "go f()" starts f in a new goroutine: a lightweight thread managed by
	// the Go runtime. The closure captures a, work and done.
	go func() {
		v := work()
		a.Post(func() { done(v) })
	}()
}

// drainPosted runs the posted functions. The queue is swapped out under the
// lock and run outside it, so a posted function may Post again; that one
// waits for the next frame.
func (a *App) drainPosted() {
	a.mu.Lock()
	fns := a.posted
	a.posted = nil
	a.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// Frame produces one frame into out: run posted functions, rebuild dirty
// nodes, lay the tree out to fill viewport, and paint it.
func (a *App) Frame(viewport Size, m draw.TextMeasurer, out *draw.List) {
	a.drainPosted()
	if a.tree == nil {
		a.tree = newTree(a.root, m, a.theme)
	} else {
		a.tree.env.measurer = m
		if a.themeChanged {
			a.themeChanged = false
			a.tree.env.theme = a.theme
			a.tree.setRoot(a.root) // same root: every node is kept, every build reruns
		}
	}
	a.tree.rebuildDirty()
	a.tree.layout(viewport)
	a.tree.paint(out)
}

// NeedsFrame reports whether there is work for another frame: nothing has
// been drawn yet, or nodes are dirty, functions are posted, or the theme
// changed. The backend asks after each frame.
func (a *App) NeedsFrame() bool {
	if a.tree == nil || len(a.tree.dirty) > 0 || a.themeChanged {
		return true
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.posted) > 0
}
