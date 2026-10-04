# flit core (spec steps 1–4 + Gio spike) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The `draw` boundary, the `ui` widget tree with layout and painting
for Text/Padding/Container/Row/Column/Build, and a throwaway Gio window that
renders a hand-built `draw.List`.

**Architecture:** `draw` is a dependency-free flat command list. `ui` keeps a
retained tree of `node`s built from immutable widget values. Layout uses
constraints down and sizes up, and paint emits absolute-coordinate commands
into a `draw.List`. Gio appears only in `backend/gio/fonts` and the spike.

**Tech Stack:** Go 1.26.2, Gio v0.10.3 (spike only), Inter 4.1 (OFL).

**Spec:** `docs/superpowers/specs/2026-10-04-flit-v1-design.md`
(plan tasks 1–5 = spec steps 1–4; task 6 = the spike after step 4).

## Global Constraints

- Module `github.com/c-airr/flit`. Code, comments, commit messages in English.
- `ui` and `draw` must not import Gio and must not use cgo:
  `GOOS=linux go vet ./draw/... ./ui/...` passes.
- Coordinates: absolute logical pixels, `float32`.
- `draw.FormatVersion = 1`; `draw.Cmd` is exactly 48 bytes with the offsets in spec §3.
- Widget constructors are short (`ui.Text`, `ui.Column`); options are
  value-receiver chained methods that return a modified copy; types are named
  `XxxWidget` (Row and Column share `FlexWidget`).
- Comment non-obvious Go for a C++ reader (generics, interfaces, value
  semantics, `unsafe`, embedding).
- Every task ends with `go test ./...` and `go vet ./...` green, then a commit
  ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- `go test -race` needs cgo and gcc on Windows, and neither is installed. Tasks 1–6
  have no concurrency, so the race detector is not needed until spec step 5.

## Review Focus

1. Empty text `""`: Text gets width 0 and one line of height, with no crash or NaN (Task 3).
2. Non-ASCII text (`"Zażółć"`): `TextBuf` offsets are bytes, `TextOf` returns
   the exact string, and the fake measurer counts runes, not bytes (Tasks 1, 2).
3. Unbounded constraints (`Inf` max): a childless Container, or a Column as
   a Row child, must never report an `Inf` size; it falls back to min (Tasks 2–4).
4. Overflow: Column children taller than the max get no negative gaps with
   SpaceBetween, and the Column's own size stays clamped to max (Task 4).
5. Empty parents: a Container with a nil child, or a Row/Column with zero
   children, has a size by the rules and paints without panicking (Tasks 3–5).

---

### Task 1: `draw` package

**Files:**
- Create: `draw/types.go`: `Point`, `Size`, `Rect`, `RGBA`, `Hex`, `Kind`, `FontStyle`, `Cmd`, `FormatVersion`, compile-time size assertion
- Create: `draw/list.go`: `List` and its methods
- Create: `draw/measure.go`: `TextMeasurer`
- Test: `draw/types_test.go`, `draw/list_test.go`

**Interfaces:**
- Produces:
  - `type Point struct{ X, Y float32 }`, `type Size struct{ W, H float32 }`, `type Rect struct{ X, Y, W, H float32 }`
  - `type RGBA struct{ R, G, B, A uint8 }`; `func Hex(rgb uint32) RGBA` (0xRRGGBB, A=255)
  - `type Kind uint32` with `FillRect=1, StrokeRect, Text, PushClip, PopClip`; `func (k Kind) String() string` returns the constant's name
  - `type FontStyle struct{ Size float32; Weight uint32 }`
  - `type Cmd struct{ Kind; Rect; Color RGBA; Radius, Width float32; TextOff, TextLen uint32; Font FontStyle }`
  - `type List struct{ Version uint32; Cmds []Cmd; TextBuf []byte }` with
    `Reset()`, `FillRect(r Rect, c RGBA, radius float32)`,
    `StrokeRect(r Rect, c RGBA, radius, width float32)`,
    `Text(r Rect, s string, c RGBA, f FontStyle)`,
    `PushClip(r Rect, radius float32)`, `PopClip()`,
    `TextOf(c Cmd) string`, `String() string`
  - `type TextMeasurer interface{ Measure(text string, style FontStyle, maxWidth float32) Size }`

- [ ] **Step 1: Write failing tests**

`TestCmdLayout`: `unsafe.Sizeof(Cmd{}) == 48`, and `unsafe.Offsetof` gives
Kind 0, Rect 4, Color 20, Radius 24, Width 28, TextOff 32, TextLen 36, Font 40.

`TestHex`: `Hex(0x4F46E5) == RGBA{0x4F, 0x46, 0xE5, 0xFF}`.

`TestListText`: after `Reset`, `Text(...)` with `"ab"` and then `"Zażółć"` →
`Version == 1`, the second cmd has `TextOff == 2`, `TextLen == 10` (bytes),
and `TextOf` returns both strings exactly.

`TestListResetKeepsCapacity`: after appending, `Reset` gives `len(Cmds) == 0`,
`len(TextBuf) == 0`, and `cap` is unchanged.

`TestListString`: a list with one of each kind prints exactly:
```
FillRect x=0 y=0 w=100 h=40 color=#4f46e5ff radius=8
StrokeRect x=0 y=0 w=100 h=40 color=#e4e4e7ff radius=8 width=1
Text x=16 y=12 w=0 color=#ffffffff size=14 weight=400 "Clicks"
PushClip x=0 y=0 w=100 h=40 radius=8
PopClip
```
(each line ends with `\n`; floats use `strconv.FormatFloat(v, 'g', -1, 32)`; text uses `%q`).

- [ ] **Step 2: Run** `go test ./draw/`. Expected: FAIL (undefined identifiers).
- [ ] **Step 3: Implement** the files above. Non-text commands get
  `TextOff=0, TextLen=0`. Add a compile-time check in `types.go`:
  `var _ [48 - unsafe.Sizeof(Cmd{})]byte` and `var _ [unsafe.Sizeof(Cmd{}) - 48]byte`
  (a negative array length fails compilation, which is Go's `static_assert`).
- [ ] **Step 4: Run** `go test ./draw/ && go vet ./draw/ && GOOS=linux go vet ./draw/`. Expected: PASS.
- [ ] **Step 5: Commit** `draw: flat command list with fixed Cmd layout and byte text buffer`.

### Task 2: geometry and fake measurer

**Files:**
- Create: `ui/geom.go`: aliases, `Inf`, `Constraints`, `Insets`
- Create: `internal/fakebackend/measurer.go`
- Test: `ui/geom_test.go`, `internal/fakebackend/measurer_test.go`

**Interfaces:**
- Consumes: `draw.Point`, `draw.Size`, `draw.FontStyle`, `draw.TextMeasurer`
- Produces:
  - `type Point = draw.Point`, `type Size = draw.Size` (aliases, not new types)
  - `var Inf = float32(math.Inf(1))`
  - `type Constraints struct{ MinW, MaxW, MinH, MaxH float32 }`;
    `func Tight(s Size) Constraints`; `func (c Constraints) Loosen() Constraints` (mins → 0);
    `func (c Constraints) Constrain(s Size) Size` (clamp each dimension into [min, max]);
    `func (c Constraints) Deflate(in Insets) Constraints` (subtract insets from min and max, clamp at 0; Inf stays Inf)
  - `type Insets struct{ Top, Right, Bottom, Left float32 }`; `func All(v float32) Insets`; `func Symmetric(vertical, horizontal float32) Insets`
  - `fakebackend.Measurer{}` implementing `draw.TextMeasurer`, with constants `CharW = 8`, `LineH = 16`

- [ ] **Step 1: Write failing tests** (table-driven)

Constraints: `Tight(Size{10,20}).Constrain(Size{5,50}) == Size{10,20}`;
`Constraints{0,Inf,0,Inf}.Constrain(Size{30,40}) == Size{30,40}`;
`Constraints{10,100,10,100}.Deflate(All(8)) == Constraints{0,84,0,84}`;
`Constraints{0,Inf,0,Inf}.Deflate(All(8))` keeps `MaxW == Inf`.

Measurer (style ignored): `""` → `{0,16}`; `"Clicks"` → `{48,16}`;
`"Zażółć"` → `{48,16}` (6 runes); `"ab\ncde"` → `{24,32}`;
`"abcdef"` with maxWidth 20 → 2 chars per line → `{16,48}`; maxWidth 0 → no wrapping.

- [ ] **Step 2: Run** `go test ./ui/ ./internal/...`. Expected: FAIL.
- [ ] **Step 3: Implement.** Measurer: split on `'\n'`; per line, rune count
  `utf8.RuneCountInString`; if `maxWidth > 0`, `perLine = max(1, floor(maxWidth/8))`,
  visual lines = `max(1, ceil(runes/perLine))`, width = `min(runes, perLine)*8`;
  result W = max line width, H = visual lines × 16.
- [ ] **Step 4: Run** `go test ./... && go vet ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `ui: constraints and insets; fakebackend: fixed-width measurer`.

### Task 3: widget tree, reconciliation, Text / Padding / Container / Build layout

**Files:**
- Create: `ui/theme.go`: `Theme`, `DefaultTheme`
- Create: `ui/widget.go`: `Widget`, `renderWidget`, `env`
- Create: `ui/node.go`: `node`, mount/update/dispose, `layoutNode`, `tree`
- Create: `ui/build.go`, `ui/text.go`, `ui/padding.go`, `ui/container.go`
- Test: `ui/node_test.go`, `ui/layout_test.go`

**Interfaces:**
- Consumes: Task 1 `draw` types; Task 2 `Constraints`, `Insets`, `Inf`, `fakebackend.Measurer`
- Produces:
  - `type Theme struct{ Background, Surface, Primary, PrimaryHover, PrimaryPressed, OnPrimary, Text, TextMuted, Outline draw.RGBA; Radius, Spacing, FontSize, TitleSize float32 }`;
    `func DefaultTheme() Theme` with Background `Hex(0xF7F7F8)`, Surface `Hex(0xFFFFFF)`,
    Primary `Hex(0x4F46E5)`, PrimaryHover `Hex(0x4338CA)`, PrimaryPressed `Hex(0x3730A3)`,
    OnPrimary `Hex(0xFFFFFF)`, Text `Hex(0x18181B)`, TextMuted `Hex(0x71717A)`, Outline `Hex(0xE4E4E7)`,
    Radius 8, Spacing 4, FontSize 14, TitleSize 20
  - `type Widget interface{ isWidget() }` (sealed)
  - unexported `renderWidget interface{ Widget; children() []Widget; layout(n *node, e *env, c Constraints) Size }`
    (Task 5 adds `paint`)
  - `type env struct{ measurer draw.TextMeasurer; theme Theme }`
  - `type node struct{ widget Widget; parent *node; children []*node; depth int; size Size; offset Point; state any; disposed bool }`
  - `func layoutNode(n *node, e *env, c Constraints) Size`: dispatches and stores `n.size`
  - `type tree struct{ root *node; env env }`; `func newTree(root Widget, m draw.TextMeasurer, th Theme) *tree`;
    `func (t *tree) setRoot(w Widget)`; `func (t *tree) layout(viewport Size)` (root gets `Tight(viewport)`)
  - `type Ctx struct{...}`; `func (c *Ctx) Theme() Theme`; `func Build(fn func(ctx *Ctx) Widget) BuildWidget`
  - `func Text(s string) TextWidget` + `Size(float32)`, `Color(draw.RGBA)`, `Weight(uint32)`
  - `func Padding(in Insets, child Widget) PaddingWidget`
  - `func Container(child Widget) ContainerWidget` (child may be nil) + `Background(draw.RGBA)`,
    `Radius(float32)`, `Border(c draw.RGBA, width float32)`, `Width(float32)`, `Height(float32)`, `Clip()`

- [ ] **Step 1: Write failing tests**

Reconciliation (`node_test.go`): mount `Container(Text("a"))`, then
`setRoot(Container(Text("b")))` → the root node and the child node are the
same pointers, and the child's widget now holds `"b"`.
`setRoot(Container(Padding(All(1), nil)))` → the old Text node has
`disposed == true` and a new child node exists. `depth` is root 0, child 1.
`Build(func(*Ctx) Widget { return Text("a") })` mounts a Build node with one Text child.

Layout (`layout_test.go`, fake measurer, default theme). Remember that tight
constraints force a size: a Text under a tight 200×100 is 200×100.
- `Text("")` via `layoutNode` with `Constraints{0,200,0,100}` → `{0,16}`.
- `Text("Clicks")` with `Constraints{0,200,0,100}` → `{48,16}`, and `n.state == float32(200)`.
- `Padding(All(16), Text("Clicks"))` as root in viewport 200×100: padding `{200,100}`, text offset `{16,16}`, text size `{168,68}` (tight).
- `Container(nil)` as root → `{200,100}`; `Container(nil)` under `Constraints{0,Inf,0,Inf}` → `{0,0}`, never Inf.
- `Container(Text("x")).Width(50).Height(30)` under `Constraints{0,200,0,100}` → `{50,30}`, and the child is `{50,30}` (tight).
(Theme defaults for Text are checked by Task 5's golden test.)

- [ ] **Step 2: Run** `go test ./ui/`. Expected: FAIL.
- [ ] **Step 3: Implement.**
  - Reconcile children by index and `reflect.TypeOf` equality. On a match:
    keep the node, set the widget, recurse. On a mismatch: `dispose()` the
    old subtree (recursive, sets `disposed`, clears `parent`) and mount a
    new one. Extra old nodes are disposed.
  - `BuildWidget`: children = `[]Widget{fn(ctx)}`, computed on mount and update.
    Layout passes the constraints through to that child.
  - Text: `maxW = c.MaxW` if finite, else 0; store it in `n.state` (Task 5
    needs it); size = `c.Constrain(measurer.Measure(...))`. Zero size,
    color or weight means the theme default.
  - Padding: the child gets `c.Deflate(in)`, its offset is `{Left, Top}`, and
    its size is the child's size plus the insets, run through `c.Constrain`.
  - Container: `Width`/`Height` > 0 tightens that axis to the value clamped
    into c. A child gets those constraints at offset 0, and the container
    size = `cc.Constrain(childSize)`. With no child, it is `cc.MaxW` if finite,
    else `cc.MinW` (same for H).
- [ ] **Step 4: Run** `go test ./... && go vet ./... && GOOS=linux go vet ./draw/... ./ui/...`. Expected: PASS.
- [ ] **Step 5: Commit** `ui: retained node tree with reconciliation; Text, Padding, Container, Build layout`.

### Task 4: Row and Column

**Files:**
- Create: `ui/flex.go`
- Test: `ui/flex_test.go`

**Interfaces:**
- Consumes: Task 3 `renderWidget`, `layoutNode`, `node`
- Produces: `type Align int` with `Start, Center, End, SpaceBetween, Stretch`;
  `func Row(children ...Widget) FlexWidget`, `func Column(children ...Widget) FlexWidget`;
  methods `Gap(float32)`, `MainAlign(Align)`, `CrossAlign(Align)`. Stretch on the
  main axis and SpaceBetween on the cross axis act as Start.

- [ ] **Step 1: Write failing tests** (Column under `Constraints{0,200,0,100}`, fake measurer; Row mirrors one case)
- `Column(Text("ab"), Text("abcd")).Gap(8)`: child offsets `{0,0}`, `{0,24}`; column size `{32,100}` (main = MaxH because it is finite; cross = widest child).
- `MainAlign(Center)`: total 40, free 60 → offsets y=30 and y=54.
- `MainAlign(End)` → y=60 and y=84; `MainAlign(SpaceBetween)` → y=0 and y=84.
- `CrossAlign(Center)` with widths 16 and 32 → x=8 and x=0; `CrossAlign(Stretch)` → the children get tight width 200 and the column width is 200.
- Overflow: 8 × `Text("x")` with Gap 0 = 128 > 100 under SpaceBetween → offsets 0, 16, …, 112 (no negative gaps), column size `{8,100}`.
- Shrinking: `setRoot(Column(Text("a"), Text("b"), Text("c")))` then `setRoot(Column(Text("a")))` → the first child node is the same pointer, and the other two are `disposed`.
- `Column()` with no children → `{0,100}`.
- Under `Constraints{0,Inf,0,Inf}`, `Column(Text("ab"))` → `{16,16}` (main falls back to the total, never Inf).
- `Row(Text("ab"), Text("abcd")).Gap(4)` → x offsets 0 and 20.

- [ ] **Step 2: Run** `go test ./ui/ -run Flex`. Expected: FAIL.
- [ ] **Step 3: Implement.** Children get the main axis as `0..Inf` and the cross
  axis as `0..crossMax` (tight `crossMax` when Stretch and finite). The main
  size is `MaxMain` if finite, else `max(total, MinMain)`. The cross size is
  the widest child, put through the constraints (`MaxCross` for Stretch).
  `free = max(0, mainSize - total)`. SpaceBetween with n > 1 uses gap + free/(n-1).
- [ ] **Step 4: Run** `go test ./... && go vet ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `ui: Row and Column with gap and alignment`.

### Task 5: painting into `draw.List`

**Files:**
- Create: `ui/paint.go`: `paintNode`, `tree.paint`
- Modify: `ui/widget.go` (add `paint` to `renderWidget`), plus `text.go`, `padding.go`, `container.go`, `flex.go`
- Test: `ui/paint_test.go`

**Interfaces:**
- Consumes: Tasks 1, 3, 4
- Produces: `renderWidget` gains `paint(n *node, e *env, origin Point, out *draw.List)`;
  `func paintNode(n *node, e *env, parentOrigin Point, out *draw.List)` (origin = parent + `n.offset`;
  Build paints its child); `func (t *tree) paint(out *draw.List)` (calls `out.Reset()` first)

- [ ] **Step 1: Write failing golden tests** (compare `out.String()`)

Viewport 200×100, root
`Container(Padding(All(16), Column(Text("Clicks"), Text("5").Size(20)).Gap(8))).Background(DefaultTheme().Surface).Radius(8)`:
```
FillRect x=0 y=0 w=200 h=100 color=#ffffffff radius=8
Text x=16 y=16 w=168 color=#18181bff size=14 weight=400 "Clicks"
Text x=16 y=40 w=168 color=#18181bff size=20 weight=400 "5"
```
Clip and border: root `Container(Text("x")).Radius(4).Clip().Border(DefaultTheme().Outline, 1)` →
```
PushClip x=0 y=0 w=200 h=100 radius=4
Text x=0 y=0 w=200 color=#18181bff size=14 weight=400 "x"
PopClip
StrokeRect x=0 y=0 w=200 h=100 color=#e4e4e7ff radius=4 width=1
```
Empty: `Row()` as root → empty list with `Version == 1`. A transparent background (zero RGBA) emits no FillRect.

- [ ] **Step 2: Run** `go test ./ui/ -run Paint`. Expected: FAIL.
- [ ] **Step 3: Implement.** Container order: FillRect (if `bg.A > 0`), then
  PushClip (if clip), children, PopClip, then StrokeRect (if border width > 0,
  so the border draws on top). Text: Rect = origin, `W = n.state` max width,
  `H = n.size.H`. Padding and Flex paint only their children.
- [ ] **Step 4: Run** `go test ./... && go vet ./... && GOOS=linux go vet ./draw/... ./ui/...`. Expected: PASS.
- [ ] **Step 5: Commit** `ui: paint the tree into draw.List`.

### Task 6: Inter font package + Gio spike (throwaway window)

**Files:**
- Create: `backend/gio/fonts/fonts.go`, `Inter-Regular.ttf`, `Inter-SemiBold.ttf`, `OFL.txt` (from the Inter 4.1 release zip, `extras/ttf/`) — **kept** for step 7
- Create: `examples/spike-gio/main.go` — **throwaway**, replaced in spec step 7
- Modify: `go.mod`, `go.sum` (`gioui.org v0.10.3`)

**Interfaces:**
- Consumes: Task 1 `draw.List`
- Produces: `func fonts.Collection() []font.FontFace` (gioui.org/font), typeface `"Inter"`

- [ ] **Step 1:** Download `https://github.com/rsms/inter/releases/download/v4.1/Inter-4.1.zip`. Copy the two TTFs and the license (as `OFL.txt`) into `backend/gio/fonts/`. Run `go get gioui.org@v0.10.3`.
- [ ] **Step 2:** Implement `fonts.Collection()` with `go:embed` and `opentype.Parse`. Test `TestCollection`: 2 faces, both with typeface `"Inter"`, weights Normal and SemiBold.
- [ ] **Step 3:** Write the spike. Use the standard Gio main (window loop in a goroutine,
  `app.Main()` on main), build one `draw.List` by hand, and translate it each frame:
  FillRect → `clip.UniformRRect` + `paint.FillShape`, StrokeRect → `clip.Stroke`,
  Text → `widget.Label` with a `text.Shaper` built from `fonts.Collection()` (no system fonts),
  PushClip/PopClip → a stack of `clip` pushes. Positions are dp × `gtx.Metric.PxPerDp`.
  Contents: a light background, a white rounded card with an outline,
  `"Hello, flit"` semibold 20, `"Zażółć gęślą jaźń 123"` regular 14, an indigo
  rounded "button" with white text, and a clipped rounded card whose inner rect overflows.
- [ ] **Step 4: Verify.** Run `go vet ./... && go build ./examples/spike-gio`. Then
  `go run ./examples/spike-gio` in the background, take a screenshot of the window,
  and check: corners are rounded, the text is sharp and has Polish diacritics, and the clip cuts off the overflow.
  Show the screenshot to the author.
- [ ] **Step 5: Commit** `spike: Gio window rendering a hand-built draw.List; embed Inter`.
