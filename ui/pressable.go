package ui

// PressState is what a Pressable knows about the pointer.
type PressState struct {
	Hovered bool // the pointer is over it
	Pressed bool // a press started on it and has not been released yet
}

// PressableWidget makes its child react to the pointer. Create it with Pressable.
type PressableWidget struct {
	fn      func(ctx *Ctx, s PressState) Widget
	onClick func()
	cursor  Cursor
}

var _ composite = PressableWidget{}

// Pressable is the building block for anything clickable. fn returns the
// child for the current state; it runs again whenever the state changes,
// so the child can look different when hovered or pressed:
//
//	ui.Pressable(func(ctx *ui.Ctx, s ui.PressState) ui.Widget {
//		bg := ctx.Theme().Primary
//		if s.Hovered {
//			bg = ctx.Theme().PrimaryHover
//		}
//		return ui.Container(ui.Text("+")).Background(bg).Radius(12)
//	}).OnClick(increment)
//
// The state lives in the Pressable's node, so it survives rebuilds of the
// widgets around it.
func Pressable(fn func(ctx *Ctx, s PressState) Widget) PressableWidget {
	return PressableWidget{fn: fn}
}

// OnClick sets the function to run on a click: a press and a release on
// this Pressable. It runs on the UI goroutine and may Set signals.
func (w PressableWidget) OnClick(fn func()) PressableWidget {
	w.onClick = fn
	return w
}

// Cursor sets the cursor shown while the pointer is over this Pressable.
// The default is CursorPointer.
func (w PressableWidget) Cursor(c Cursor) PressableWidget {
	w.cursor = c
	return w
}

func (PressableWidget) isWidget() {}

func (w PressableWidget) build(n *node, ctx *Ctx) Widget {
	// The comma-ok form gives the zero PressState before the first event.
	s, _ := n.state.(PressState)
	return w.fn(ctx, s)
}

func (w PressableWidget) cursorShape() Cursor {
	if w.cursor == 0 {
		return CursorPointer
	}
	return w.cursor
}

// setPress changes n's PressState through change and schedules a rebuild
// if anything actually changed.
func setPress(n *node, change func(s *PressState)) {
	s, _ := n.state.(PressState)
	old := s
	change(&s) // &s is a pointer to the local copy, like passing by reference
	if s != old {
		n.state = s
		n.markDirty()
	}
}
