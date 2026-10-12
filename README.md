# flit

A Flutter-style GUI library for Go. You describe the UI as a tree of widgets, keep state in signals, and flit rebuilds only the parts that changed. Unstyled widgets already look finished, and every widget takes style options when you want your own look.

It draws with [Gio](https://gioui.org) on Windows and Linux.

<!-- screenshots: docs/screenshots/counter.png, docs/screenshots/todo.png -->

> **Status: early.** The core works (layout, signals, hooks, buttons, custom interactive widgets), but there are no text fields, scrolling, keyboard input, images or animations yet, and there is no tagged release. The API can still change.

## Example

```go
package main

import (
	"strconv"

	"github.com/c-airr/flit/backend/gio"
	"github.com/c-airr/flit/ui"
)

func main() {
	count := ui.NewSignal(0)

	root := ui.Build(func(ctx *ui.Ctx) ui.Widget {
		th := ctx.Theme()
		return ui.Container(
			ui.Column(
				// Only this Build reads count, so a click rebuilds just the
				// number, not the whole window.
				ui.Build(func(*ui.Ctx) ui.Widget {
					return ui.Text(strconv.Itoa(count.Get())).Size(48).Weight(600)
				}),
				ui.Button("+", func() {
					count.Update(func(v int) int { return v + 1 })
				}),
			).Gap(16).MainAlign(ui.Center).CrossAlign(ui.Center),
		).Background(th.Background)
	})

	gio.Run(ui.NewApp(root), gio.Options{Title: "flit counter", Width: 320, Height: 240})
}
```

More in [`examples/`](examples): `counter` (above) and `todo` (a list with `Expanded`, `Spacer` and a local hook).

## Install and run

```sh
go get github.com/c-airr/flit
```

To run the examples from a clone:

```sh
git clone https://github.com/c-airr/flit
cd flit
go run ./examples/counter
go run ./examples/todo
```

The module needs Go 1.26.2 or newer (see `go.mod`). On Windows nothing else is required. On Linux, Gio needs a C compiler and a few development packages, for example on Debian/Ubuntu:

```sh
sudo apt-get install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev
```

## How it works

- **Widgets are cheap values.** `ui.Build(func(ctx *ui.Ctx) ui.Widget)` is how you write a component. flit keeps a persistent tree of nodes underneath and only rebuilds dirty parts.
- **Signals** hold state: `ui.NewSignal(v)`, then `Get`, `Set`, `Update`. A `Build` that calls `Get` re-runs when the value changes, and no other part of the tree does.
- **Hooks** give a component its own state: `ui.UseSignal(ctx, initial)` and `ui.Remember(ctx, init)`. They must be called in the same order on every build; flit panics if the order or types change.
- **Threads:** `app.Post(fn)` runs `fn` on the UI goroutine from any goroutine, and `ui.Async(app, work, done)` runs `work` in a goroutine and delivers the result to `done` on the UI goroutine.
- **Immediate-mode backend, retained UI.** Only `backend/gio` knows about Gio. The `ui` package produces a flat list of draw commands (`draw.List`), so `ui` and `draw` build without cgo and the backend can be replaced.

## Widgets

| Widget | What it does |
|---|---|
| `ui.Text(s)` | Text. Options: `Size`, `Color`, `Weight`. |
| `ui.Button(label, onClick)` | Button. Options: `Background`, `HoverBackground`, `PressedBackground`, `TextColor`, `Radius`, `Padding`, `FontSize`, `Weight`. |
| `ui.Container(child)` | A box. Options: `Background`, `Radius`, `Border`, `Width`, `Height`, `Clip`. |
| `ui.Padding(insets, child)` | Space around a child. Insets: `ui.All(v)`, `ui.Symmetric(vertical, horizontal)`. |
| `ui.Row(...)`, `ui.Column(...)` | Lay children out in a line. Options: `Gap`, `MainAlign`, `CrossAlign` with `Start`, `Center`, `End`, `SpaceBetween` and `Stretch` (cross axis only). |
| `ui.Expanded(child)`, `ui.Spacer()` | Take the free space in a Row or Column. `Expanded(...).Flex(n)` sets the share. |
| `ui.Pressable(fn)` | Draw any interactive element yourself: `fn` gets the press state (`Hovered`, `Pressed`) and returns a widget. `OnClick` and `Cursor` set the behavior. |
| `ui.Build(fn)` | A component with its own dependency tracking. |

Colors come from `draw.RGBA` (`draw.Hex(0x4F46E5)`). `ui.DefaultTheme()` is a light theme with an indigo accent; replace it with `app.SetTheme(th)`. Components read the theme through `ctx.Theme()`.

Set `FLIT_DEBUG=1` to get warnings about layouts that are allowed but probably not intended, for example `Expanded` outside a Row or Column.

## Project layout

| Package | Role |
|---|---|
| `ui` | Public API: widgets, layout, `Signal[T]`, hooks, `Theme`, `App`, `Post`, `Async`. |
| `draw` | The boundary between `ui` and a backend: the draw-command list and the `TextMeasurer` interface. |
| `backend/gio` | Window, input, DPI, text shaping, and drawing on Gio. Bundles the Inter font (SIL OFL). |
| `internal/fakebackend` | Fixed-width text measurer for tests. |
| `examples/` | `counter` and `todo`. |

The design notes live in [`docs/superpowers/specs`](docs/superpowers/specs).

## Tests

```sh
go vet ./...
go test ./...
```

CI runs these on Linux (with `-race`) and Windows.

## Not yet

Text fields and keyboard input, scrolling, keys for list rows, images, animations, dark theme, hot reload.

## License

[Apache-2.0](LICENSE). The bundled Inter font is under the SIL Open Font License (`backend/gio/fonts/OFL.txt`).
