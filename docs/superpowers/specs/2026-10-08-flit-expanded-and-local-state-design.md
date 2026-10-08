# flit: Expanded and component-local state – design

Date: 2026-10-08 · Status: draft for the author's review

## 1. Purpose and scope

v1 can only show a counter. To build real apps, flit needs layouts that
fill the window and components that keep their own state. This is the
first of three cycles after v1:

- **A (this spec):** `Expanded`/`Spacer` and component-local state (hooks).
- **B:** keys (`ui.Key`) first, then `TextField` with keyboard and focus.
- **C:** scrolling.

Finish line for A: `examples/todo` runs, a to-do list driven by buttons
(no text input yet), laid out with `Expanded`/`Spacer`.

Non-goals for A: keys, effects/cleanup hooks, text input, scrolling.

## 2. Component-local state (hooks)

### API

```go
// UseSignal returns a signal that belongs to the building node and
// survives its rebuilds. initial is used only in the node's first build.
func UseSignal[T any](ctx *Ctx, initial T) *Signal[T]

// Remember returns a value that belongs to the building node. init runs
// only in the node's first build; later builds return the stored value.
func Remember[T any](ctx *Ctx, init func() T) T
```

`UseSignal(ctx, v)` is `Remember(ctx, func() *Signal[T] { return NewSignal(v) })`
with its own name in error messages.

```go
func Counter(label string) ui.Widget {
    return ui.Build(func(ctx *ui.Ctx) ui.Widget {
        count := ui.UseSignal(ctx, 0)
        return ui.Row(ui.Text(label), ui.Button("+", func() { count.Update(inc) }))
    })
}
```

**Documented on both functions:** the initial value and the init closure
are used only in the node's first build. If they depend on data from the
parent (e.g. `UseSignal(ctx, props.Start)`), later changes of that data do
not reach the stored value. Read parent data directly in the build instead.

### Where hooks work

In any user function that receives a `*Ctx` during a build: the `Build`
function and the `Pressable` function. The `Ctx` is valid only while that
build runs.

### How it works

- The composite node keeps a list of **slots**. Each build starts at slot 0
  and every hook call takes the next slot.
- The node's first build creates the slots. Every later build must make
  **the same number of hook calls, with the same hook and type in each
  slot.** Each slot records the hook name and the value's dynamic type.
- Violations panic with a message that names the slot and both sides:
  - type or hook changed:
    `ui: hook #2 was UseSignal[int] in the previous build and is UseSignal[string] now; call hooks in the same order every build, never inside if or for`
  - more calls than before (detected at the extra call):
    `ui: hook #3 is new in this build (the previous build made 3 hook calls); call hooks in the same order every build, never inside if or for`
  - fewer calls than before (detected when the build returns):
    `ui: this build made 2 hook calls, the previous build made 3; call hooks in the same order every build, never inside if or for`
  - a hook called outside a build (e.g. through a `Ctx` kept in a click handler):
    `ui: UseSignal called outside a build; hooks only work while a Build or Pressable function runs`
- If the build itself panics, the count check is skipped (the original
  panic is the one to see).
- State lives as long as the node. When reconciliation replaces the node
  (different widget type at that position), its slots are dropped.

### Known limitations

- Matching is by position and type: two `Build` components that swap
  places in a list swap their state. Keys (cycle B) fix this.
- Two slots of the same hook and type that swap order (e.g. two
  `UseSignal[int]` calls exchanged by an edit) are not detected; the check
  sees the same types.

## 3. `Expanded` and `Spacer`

### API

```go
func Expanded(child Widget) ExpandedWidget   // child may be nil
func (w ExpandedWidget) Flex(f int) ExpandedWidget // default 1; values < 1 count as 1
func Spacer() ExpandedWidget                // Expanded(nil)

ui.Row(ui.Text("Name"), ui.Spacer(), ui.Button("OK", ok))
ui.Column(header, ui.Expanded(content), footer)
ui.Row(ui.Expanded(a).Flex(2), ui.Expanded(b))
```

### Layout in Row/Column

A flex child is a child of a `Row`/`Column` that is an `Expanded`, looking
through composite nodes (`Build`, `Pressable`, `Button`): a component that
returns an `Expanded` works as a flex child.

When the main axis is bounded:
1. Lay out the non-flex children as today (main axis `0..Inf`).
2. `remaining = max(0, maxMain - sum of non-flex sizes - all gaps)`.
3. Each flex child gets `remaining * flex / totalFlex` as a tight main-axis
   size, in order. **The last flex child gets `remaining` minus what the
   others got**, so the parts always add up to exactly `remaining`.
4. Cross axis as today (including `Stretch`).
5. There is no free space left, so `MainAlign` has no effect; positions
   follow from the sizes and gaps.

When the main axis is unbounded (e.g. a `Column` inside a `Column`), flex
children are laid out like ordinary children.

Outside a `Row`/`Column`, an `Expanded` lays out exactly like its child
(an empty `Expanded` is the minimum size).

### Debug warnings

With the environment variable `FLIT_DEBUG=1`, flit logs one line per
node (not per frame) for misuse that is otherwise silent:
- `flit: Expanded is not a child of a Row or Column; it lays out like its child`
- `flit: Expanded in a Row/Column with an unbounded main axis; it lays out like an ordinary child`

The variable is read once at start-up. Without it nothing is logged.

## 4. `examples/todo`

- The task list is one signal in the app, `Signal[[]Task]` with
  `Task{Title string; Done bool}`. The list items are plain functions of a
  `Task`, **without hooks**, because without keys their state would
  follow positions, not tasks.
- Layout:
  - **header:** a title, then `Spacer`, then a counter like "2 / 5 done";
  - **middle:** an `Expanded` `Column` of rows. Each row has a toggle button, the title in an `Expanded`, and a remove button;
  - **footer:** an "Add task" button that appends "Task N".
- One legitimate hook: the toolbar component keeps a local
  `UseSignal(ctx, false)` "hide done" filter toggle.
- Cycle B replaces the "Add task" button with a `TextField`, after keys exist.

## 5. Testing

- **Hooks:**
  - State survives rebuilds of the node and of its ancestors.
  - State is dropped when the node is replaced by another type.
  - `Remember`'s init runs once.
  - Every panic message above is checked exactly: changed type, changed hook, more calls, fewer calls, outside a build.
  - A build that panics does not trigger a second panic from the count check.
  - Hooks work inside `Pressable`.
- **Expanded, table-driven:**
  - one `Expanded`, a 2:1 split, and splits that do not divide evenly (e.g. 100 into three), where the parts must sum exactly to `remaining`;
  - gaps;
  - overflow: the non-flex children already exceed the max, so flex children get 0;
  - unbounded main axis;
  - `Expanded` found through a composite;
  - `Expanded` outside a `Row`/`Column`;
  - `Spacer` pushing a child to the end;
  - `Flex(0)` treated as 1.
- **Debug warnings:** with debug enabled through a test hook (not the real env var), each warning is logged once per node; with it disabled, nothing is logged.
- `go test -race ./ui/...` as in v1.
