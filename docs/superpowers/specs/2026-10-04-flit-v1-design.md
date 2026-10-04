# flit v1 – design

Date: 2026-10-04 · Status: draft, awaiting author review

## 1. Purpose

flit is a GUI library for Go with a declarative, Flutter-style API that looks
good without configuration. There is no specific target application yet: flit
is general purpose. v1 is done when `examples/counter` runs on Windows and
Linux. The architecture must allow adding lists, text fields and scrolling
later without rewriting the core.

Non-goals for v1: animations, hot reload, keyboard input, text fields,
scrolling, images, flex/Expanded, component-local state, keys, dark theme,
c-shared embedding, any C++ code.

## 2. Architecture overview

flit is **retained mode**: a persistent tree of nodes survives between frames
and only dirty parts are rebuilt. The user writes a declarative tree of cheap
widget values. Gio, the backend, is immediate mode, but only the backend
package sees that.

```
 user code ──► ui (widgets, nodes, layout, signals) ──► draw.List ──► backend/gio ──► GPU
                    ▲                                                     │
                    └──── PointerEvent, window size, TextMeasurer ◄───────┘
```

Packages and dependency direction (`ui` never imports Gio):

| Package | Role |
|---|---|
| `draw` | The boundary: command list format and the `TextMeasurer` interface. No dependencies. |
| `ui` | Public API: widgets, `Signal[T]`, `Theme`, `App`, `Post`, `Async`. Depends on `draw`. |
| `backend/gio` | Window, input, DPI, text shaping, `draw.List` → Gio ops. Depends on `draw`, `ui`, Gio v0.10.x. |
| `internal/fakebackend` | Fixed-width `TextMeasurer` for tests. |
| `examples/counter` | v1 finish line. |

`ui` and `draw` must build without cgo, so `GOOS=linux go vet` works on
Windows. Only `backend/gio` needs cgo, and only on Linux.

## 3. The drawing boundary (`draw`)

All coordinates are **absolute logical pixels** (`float32`). The backend
multiplies them by the DPI scale. The list is flat: plain structs with no
Go pointers, so it can later be passed to C/C++ unchanged.

```go
type Rect struct{ X, Y, W, H float32 }
type RGBA struct{ R, G, B, A uint8 } // straight (non-premultiplied) alpha

type Kind uint8
const (
    FillRect Kind = iota // Rect, Color, Radius
    StrokeRect           // Rect, Color, Radius, Width
    Text                 // Rect (top-left + max width), Color, Text, Font
    PushClip             // Rect, Radius
    PopClip
)

type FontStyle struct {
    Size   float32
    Weight uint16 // 400 regular, 600 semibold, ...
}

type Cmd struct {
    Kind   Kind
    Rect   Rect
    Color  RGBA
    Radius float32
    Width  float32
    Text   int32 // index into List.Texts, -1 if unused
    Font   FontStyle
}

type List struct {
    Cmds  []Cmd
    Texts []string
}
// Methods: FillRect, StrokeRect, Text, PushClip, PopClip append a command.
// Reset() truncates both slices but keeps their capacity (no per-frame allocs).

type TextMeasurer interface {
    // Measure returns the size of text laid out with the given style.
    // maxWidth <= 0 means unlimited (single line unless the text has '\n').
    Measure(text string, style FontStyle, maxWidth float32) Size
}
```

A `Text` command and a `Measure` call with the same arguments must agree.
The backend uses the same shaper for both.

## 4. Widgets and nodes (`ui`)

### Two trees
- **Widget**: an immutable value describing UI, created anew on every build.
  It is cheap because it is a small struct; old ones are collected by the GC.
- **node** (unexported): the persistent instance. It holds the current widget,
  parent and children, the layout result (size and offset relative to the
  parent), signal subscriptions, a dirty flag, depth and per-widget state
  (e.g. button hover).

```go
// Widget is sealed in v1: only flit's built-in widgets and Build implement it.
// User code composes widgets; custom render widgets come after v1.
type Widget interface{ isWidget() }
```

### Kinds of widgets
- **Composite**: `ui.Build(func(ctx *ui.Ctx) ui.Widget)` is how users write
  components. Build functions run with dependency tracking (§5).
- **Render widgets** (built-ins): implement an unexported interface with
  `children()`, `layout(...)`, `paint(...)` and an optional `pointer(...)`.

### Reconciliation
When a node rebuilds, its new child widgets are matched against the existing
child nodes **by index and dynamic type** (`reflect.TypeOf`). On a match the
node is kept and gets the new widget. On a mismatch the old subtree is
disposed and a new one is mounted. Disposing a node unsubscribes it from all
signals; otherwise a signal would keep it alive forever, and the GC cannot
help with that.

### API style
Short constructors for widgets; `NewX` only for stateful objects (`NewSignal`,
`NewApp`). Options are **chained value-receiver methods** that return a
modified copy:

```go
ui.Text("Clicks").Size(20).Color(theme.Primary)
ui.Column(a, b).Gap(8).CrossAlign(ui.Center)
ui.Container(child).Background(theme.Surface).Radius(12)
ui.Padding(ui.All(16), child)
ui.Button("+", func() { count.Set(count.Get() + 1) })
```

Go does not allow a function and a type with the same name in one package,
so the types are named `TextWidget`, `ColumnWidget`, and so on.

### v1 widgets
| Widget | Layout | Paint |
|---|---|---|
| `Text` | measured by `TextMeasurer`, max width from constraints | `Text` cmd |
| `Padding` | shrinks constraints by insets, adds them back to the child's size | children only |
| `Container` | optional fixed `Width`/`Height`, else child size (or max if no child) | `FillRect` (+ `StrokeRect` if a border is set) |
| `Row` / `Column` | children laid out loosely along the main axis; `MainAlign` (Start, Center, End, SpaceBetween), `CrossAlign` (Start, Center, End, Stretch), `Gap` | children only |
| `Button` | text + theme padding | rounded rect, color from state (normal/hover/pressed), text |

**Change from the planning discussion:** `Button` is a *render widget* in v1,
not a composite. Its hover/pressed state lives in its node. A composite button
would need component-local state, which is postponed until after v1. Once
local state exists, Button can become a composite of Container + Text +
a gesture widget without any API change.

## 5. State: `Signal[T]`

```go
count := ui.NewSignal(0) // Signal[int], T inferred
count.Get()              // read; inside a build, subscribes the building node
count.Set(5)             // UI goroutine only; marks subscribers dirty, wakes the app
count.Update(func(v int) int { return v + 1 })
```

- **Tracking:** a package-level variable holds "the node currently building".
  `Get()` reads it. In C++ terms this is like a `thread_local`. Go has no
  such thing, but it does not need one, because all building happens on the
  UI goroutine.
- Before a rebuild, a node drops all its subscriptions and records them again,
  so dependencies can change between builds (e.g. behind an `if`).
- `Set` always notifies, without an equality check, so `T` can be any type
  (`T any`, not `comparable`).
- Signals are created by the user (in `main` or in their own struct) and live
  outside the tree. A signal knows its subscriber nodes, and each node knows
  its `App`, which is how `Set` reaches the right app.

## 6. App, event loop and goroutines

```go
app := ui.NewApp(root)  // root is a Widget, usually ui.Build(...)
app.SetTheme(theme)     // optional
app.Post(fn)            // any goroutine: run fn on the UI goroutine soon
ui.Async(app, work, done) // run work() in a goroutine, then done(result) on the UI goroutine
```

`Async` is a generic function and not a method, because Go methods cannot
have type parameters.

The backend drives the app through a small exported surface:

```go
app.SetWakeup(func())                  // backend's "please schedule a frame", goroutine-safe
app.Pointer(ev PointerEvent)           // UI goroutine
app.Frame(viewport Size, m draw.TextMeasurer, out *draw.List) // UI goroutine
app.NeedsFrame() bool
```

`Post` appends to a mutex-protected queue and calls the wakeup function.
A mutex queue is used instead of a channel so that `Post` never blocks,
even with many pending calls. The queue is drained at the start of
`Frame` and `Pointer`.

**Frame pipeline** (runs only when something is dirty or the window changed):
1. Drain the `Post` queue.
2. Rebuild dirty nodes, shallowest first. A node whose ancestor was just
   rebuilt is skipped, because that rebuild already handled it.
3. Lay out the whole tree with tight constraints equal to the viewport.
4. Paint the whole tree into `out`, after `out.Reset()`.

Steps 3–4 cover the whole tree in v1. Relayout boundaries and paint caching
can come later, when profiling shows a need; they do not change the API.

**Input:** `PointerEvent{Kind: Press|Release|Move|Leave, Pos Point}`.
Hit-testing uses the layout rectangles: the deepest node containing the point
that handles pointer events gets the event. The app tracks the hovered node
to produce enter/leave. A click is a press and a release inside the same node.

## 7. Theme and look

```go
type Theme struct {
    Background, Surface, Primary, PrimaryHover, PrimaryPressed,
    OnPrimary, Text, TextMuted, Outline draw.RGBA
    Radius   float32   // 8
    Spacing  float32   // 4, the grid unit
    FontSize float32   // 14 body
    TitleSize float32  // 20
}
func DefaultTheme() Theme
```

Widgets read their defaults from the app's theme (via `Ctx.Theme()` in
builds, internally when painting), so an unstyled tree already looks
finished. Font: **Inter 4.1**, embedded with `go:embed` in `backend/gio`,
with `OFL.txt` next to the font files.

## 8. Gio backend (`backend/gio`)

```go
func Run(app *ui.App, opts Options) error // Options{Title string; Width, Height float32}
```

- Gio requires `app.Main()` on the main goroutine. `Run` starts the window
  loop in a goroutine, which becomes flit's UI goroutine, and then calls
  `app.Main()`.
- `SetWakeup` is wired to `window.Invalidate()`, which is goroutine-safe.
- On each `FrameEvent`: convert pointer events from a full-window input area
  into `ui.PointerEvent` (px → dp), call `app.Frame`, then translate the list:
  `FillRect` → `clip.RRect` + `paint.Fill`, `StrokeRect` → `clip.Stroke`,
  `Text` → shaped text with the same `text.Shaper` used by the measurer,
  `PushClip`/`PopClip` → a `clip` stack.
- The measurer wraps Gio's `text.Shaper` loaded with Inter.
- The Gio version is pinned in `go.mod` (v0.10.x).

## 9. Testing

- `draw`: list building and Reset.
- `ui` layout: table-driven tests per widget against `fakebackend`
  (every character is 8 × 16 px).
- `ui` paint: golden tests that build a small tree, run `Frame` and compare
  `draw.List` with the expected commands.
- Signals: build-counter tests prove that after `Set` **only the dependent
  node rebuilds**, and that a disposed node is unsubscribed. Run with
  `go test -race`.
- Input: synthetic `PointerEvent`s check hover, pressed and click on `Button`.
- Backend: manual check with `examples/counter` on Windows. Linux via GitHub
  Actions (ubuntu, `go vet` + `go test`) once the author creates the repo,
  until then `offload run -- go test ./...` on zap.

## 10. Implementation order (each step: tests green, commit)

1. `draw` package.
2. `ui` core: geometry, Constraints, Widget/node, mount and reconcile,
   `Text`, `Padding`, `Container`.
3. `Row` / `Column`.
4. Painting into `draw.List`, including clips.
5. `Signal[T]`, dirty rebuilds, `Post`, `Async`.
6. Pointer input, `Button`, `Theme`.
7. `backend/gio`, `Run`, `examples/counter`.
8. Linux CI.

After v1, the next step is a minimal C++ backend (clang via llvm-mingw,
rectangles only, one `extern "C"` call per frame), to prove the boundary
can be swapped.
