# flit cycle A (Expanded + local state) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hook-based component-local state (`UseSignal`, `Remember`),
`Expanded`/`Spacer` in `Row`/`Column`, debug warnings, and `examples/todo`.

**Architecture:** Composite nodes get a slot list; each build walks it in
order through the `Ctx`, checking hook name and type per slot. Flex layout
gains a second pass that splits the remaining main-axis space among
`Expanded` children (found through composites), the last one taking the
remainder.

**Tech Stack:** Go 1.26.2, Gio v0.10.3 (example only), llvm-mingw clang for `-race`.

**Spec:** `docs/superpowers/specs/2026-10-08-flit-expanded-and-local-state-design.md`

## Global Constraints

- Code, comments, commits in English; comment non-obvious Go for a C++ reader.
- `ui` and `draw` must not import Gio and must not use cgo.
- Widget API style: short constructors, chained value-receiver options, `XxxWidget` types.
- Panic and warning texts are exactly as in the spec (hook numbers are 0-based).
- Every task ends green on:
  - `go test ./...`
  - `go vet ./...`
  - `GOOS=linux go vet ./draw/... ./ui/...`
  - `CC=clang CGO_ENABLED=1 go test -race ./ui/...` with llvm-mingw's `bin` on PATH

  Then commit, ending the message with the session's attribution lines.
- No push without the author's OK.

## Review Focus

1. A hook-order panic raised during a rebuild that a click caused: it must
   surface with the spec's message, not a nil-pointer panic (Task 1).
2. Nested `Build`s: parent and child keep separate slots (Task 1).
3. A child wider than its `Expanded` share (long text in a `Row`): it is
   squeezed to the share, and siblings keep their positions (Task 2).
4. No room left (non-flex children already overflow): flex children get 0,
   never negative or NaN (Task 2).
5. `Column(header, Expanded(x))` inside a `Row`: the Column's main axis is
   bounded by the Row's cross size, so `Expanded` works there (Task 2).

---

### Task 1: hooks

**Files:**
- Create: `ui/hooks.go`
- Modify: `ui/build.go` (`Ctx`), `ui/node.go` (`runBuild`, `node`)
- Test: `ui/hooks_test.go`

**Interfaces:**
- Consumes: `runBuild`, `composite`, `NewSignal`, `newTestTree`, `NewApp`, `Pointer` (existing).
- Produces:
  - `func UseSignal[T any](ctx *Ctx, initial T) *Signal[T]`
  - `func Remember[T any](ctx *Ctx, init func() T) T`
  - `node` gains `hooks []hookSlot` (`hookSlot{name string; value any}`, name like
    `UseSignal[int]` or `Remember[int]`, from `reflect.TypeFor[T]().String()`)
    and `hooksKnown bool` (true after the first completed build).
  - `Ctx` gains `node *node`, `next int` (next slot), `done bool`.

- [ ] **Step 1: Write failing tests** in `ui/hooks_test.go`. Helper
  `panicMessage(fn func()) string` recovers and returns `fmt.Sprint(r)` ("" if no panic).
  - `TestUseSignalSurvivesRebuild`: a Build with `count := UseSignal(ctx, 0)` that shows `count.Get()` and keeps `count` in an outer variable. `count.Set(3); rebuildDirty()` → the text is "3", and a parent rebuild (`setRoot` of the same tree) keeps 3.
  - `TestRememberInitRunsOnce`: init counts calls; after 3 rebuilds, 1 call.
  - `TestStateDroppedWhenNodeReplaced`: root Build switches its child between `Build(component with UseSignal(ctx, 0))` and `Text("x")` by a signal; set the component's signal to 5, switch away and back → value 0.
  - `TestNestedBuildsHaveOwnSlots`: parent `UseSignal(ctx, "p")`, child `UseSignal(ctx, 1)` → no panic over 2 rebuilds, values independent.
  - `TestHookInsidePressable`: `UseSignal` in a `Pressable` function survives hover rebuilds (driven through `App.Pointer`).
  - `TestHookPanics` (table; each case = two build bodies switched by a signal, driven with `newTestTree` + `rebuildDirty`), exact messages:
    - type: `UseSignal(ctx, 0)` → `UseSignal(ctx, "x")`: `ui: hook #0 was UseSignal[int] in the previous build and is UseSignal[string] now; call hooks in the same order every build, never inside if or for`
    - hook: `UseSignal(ctx, 0)` → `Remember(ctx, func() int { return 0 })`: `ui: hook #0 was UseSignal[int] in the previous build and is Remember[int] now; call hooks in the same order every build, never inside if or for`
    - more: 2 calls → 3 calls: `ui: hook #2 is new in this build (the previous build made 2 hook calls); call hooks in the same order every build, never inside if or for`
    - fewer: 3 calls → 2 calls: `ui: this build made 2 hook calls, the previous build made 3; call hooks in the same order every build, never inside if or for`
  - `TestHookOutsideBuild`: keep the `ctx` from a build, call `UseSignal(saved, 0)` after it → `ui: UseSignal called outside a build; hooks only work while a Build or Pressable function runs`; same for `Remember` with its own name.
  - `TestBuildPanicSkipsCountCheck`: a build that makes 1 hook call, then on rebuild panics with "boom" before any hook → recovered message is "boom".
  - `TestHookPanicFromClick` (Review Focus 1): a `Pressable`'s `OnClick` flips a signal that makes the Build call one more hook; `App.Pointer` click then `Frame` → the recovered message starts with `ui: hook #1 is new`.
- [ ] **Step 2: Run** `go test ./ui/ -run 'Hook|UseSignal|Remember|Nested|StateDropped'`. Expected: FAIL (undefined: UseSignal).
- [ ] **Step 3: Implement.** `runBuild` creates the `Ctx` with the node,
  marks it `done` in a `defer` (also on panic), and after a build that
  returned normally checks the count (only if `hooksKnown`), then sets
  `hooksKnown`. Both hooks share one unexported helper that takes the
  slot name and an init function returning `any`. Doc comments carry the
  spec's "initial value / init used only in the first build" note.
- [ ] **Step 4: Run** the Global Constraints checks. Expected: PASS.
- [ ] **Step 5: Commit** `ui: UseSignal and Remember hooks with order and type checks`.

### Task 2: `Expanded`, `Spacer`, debug warnings

**Files:**
- Create: `ui/expanded.go`, `ui/debug.go`
- Modify: `ui/flex.go`, `ui/node.go` (`node.warned bool`)
- Test: `ui/expanded_test.go`

**Interfaces:**
- Consumes: `FlexWidget.layout`, `layoutNode`, `composite`, `layoutUnder`, `childOffsets` (existing).
- Produces:
  - `func Expanded(child Widget) ExpandedWidget`, `func (w ExpandedWidget) Flex(f int) ExpandedWidget`, `func Spacer() ExpandedWidget`. `ExpandedWidget` is a `renderWidget` (child or none; paints its child).
  - `func flexOf(n *node) int`: follows single-child composite nodes down; returns the `Expanded` flex (values < 1 → 1), or 0 if not an `Expanded`.
  - `ui/debug.go`: `var debugEnabled = os.Getenv("FLIT_DEBUG") == "1"`, `var debugLogf = log.Printf` (tests replace both), `func warnOnce(n *node, msg string)` (logs `msg` once per node when enabled).

- [ ] **Step 1: Write failing tests** in `ui/expanded_test.go` (fake measurer; Row under `Constraints{0, 200, 0, 100}` unless stated; `Text("ab")` = 16×16):
  - `TestExpandedSplits` (table of child widths and x offsets):
    - `Row(Text("ab"), Expanded(nil))` → widths 16, 184; x 0, 16
    - `Row(Expanded(nil).Flex(2), Expanded(nil))` → 133.33334, 66.66666 (the second = 200 − first)
    - `Row(Expanded(nil), Expanded(nil), Expanded(nil))` with the max set to 100 → the three widths sum exactly to 100 (assert `w0+w1+w2 == 100` in float32)
    - `Row(Text("ab"), Expanded(nil)).Gap(10)` → widths 16, 174; x 0, 26
    - `Row(Text("ab"), Spacer(), Text("ab"))` → last x = 184
    - `Row(Expanded(nil).Flex(0), Expanded(nil))` → 100, 100
    - Row size stays 200 in every case.
  - `TestExpandedOverflow` (Review Focus 4): `Row(Text(25 × "a"), Expanded(nil))` (200 wide text) plus a third `Text("ab")` → the Expanded's width 0, no NaN.
  - `TestExpandedSqueezesWideChild` (Review Focus 3): `Row(Expanded(Text("abcdefghijklmnopqrstuvwxyz")), Text("ab"))` → Expanded child width 184, the last Text at x 184.
  - `TestExpandedThroughComposite`: `Row(Build(func(*Ctx) Widget { return Expanded(nil) }), Text("ab"))` → first child width 184.
  - `TestExpandedUnboundedMain`: `Row(Text("ab"), Expanded(Text("abcd")))` under `Constraints{0, Inf, 0, 100}` → widths 16, 32.
  - `TestExpandedInColumnInRow` (Review Focus 5): `Row(Column(Text("ab"), Expanded(nil)))` → the Expanded's height 84.
  - `TestExpandedOutsideFlex`: `Expanded(Text("ab"))` alone under `Constraints{0, 200, 0, 100}` → 16×16; `Expanded(nil)` → 0×0.
  - `TestDebugWarnings`: with `debugEnabled = true` and `debugLogf` capturing (restored with `t.Cleanup`):
    - laying out `Expanded(nil)` outside a flex twice logs exactly one `flit: Expanded is not a child of a Row or Column; it lays out like its child`;
    - the unbounded case logs exactly one `flit: Expanded in a Row/Column with an unbounded main axis; it lays out like an ordinary child`;
    - with `debugEnabled = false`, nothing is logged.
- [ ] **Step 2: Run** `go test ./ui/ -run 'Expanded|Debug'`. Expected: FAIL (undefined: Expanded).
- [ ] **Step 3: Implement** the spec §3 algorithm in `FlexWidget.layout`:
  - pass 1: the non-flex children;
  - then `remaining`;
  - pass 2: the flex children in order, the last one taking the remainder;
  - `free` is 0 when any flex child exists.

  With an unbounded main axis, flex children are laid out in pass 1 and `warnOnce` the unbounded warning. `ExpandedWidget.layout` warns when its nearest non-composite ancestor is not a `FlexWidget`.
- [ ] **Step 4: Run** the Global Constraints checks. Expected: PASS (all v1 flex tests unchanged).
- [ ] **Step 5: Commit** `ui: Expanded and Spacer in Row/Column; FLIT_DEBUG warnings`.

### Task 3: `examples/todo`

**Files:**
- Create: `examples/todo/main.go`

**Interfaces:**
- Consumes: Tasks 1–2, `gio.Run`.
- Produces: nothing used by other code.

- [ ] **Step 1: Write the example per spec §4.**
  - Data: `type Task struct{ Title string; Done bool }`. One `ui.NewSignal([]Task{...})`, seeded with 3 tasks, one of them done. Changes replace the slice: `Update` with a copied slice, never mutate in place.
  - Row and button functions take a `Task` and its index, and use no hooks.
  - The toolbar is a `Build` with `hide := ui.UseSignal(ctx, false)` and a button that toggles it. Pass `hide` down by keeping the list inside the same `Build`.
  - Window: `Options{Title: "flit todo", Width: 420, Height: 360}`.
- [ ] **Step 2: Verify.**
  - Run `go vet ./... && go build -o <scratch>/todo.exe ./examples/todo`.
  - Start it and take a screenshot. Check that the header counter sits at the right edge, the list fills the middle, the footer sits at the bottom, and each row's remove button is at the right edge.
  - Show the screenshot to the author. Let them check the clicks with a real mouse; no synthetic clicks.
- [ ] **Step 3: Commit** `examples: todo list with Expanded, Spacer and a local hook`.
