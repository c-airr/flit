package ui

// PointerKind says what happened to the pointer.
type PointerKind uint8

const (
	Press   PointerKind = iota + 1 // the primary button went down
	Release                        // the primary button went up
	Move                           // the pointer moved
	Leave                          // the pointer left the window
)

// PointerEvent is one pointer event in dp, relative to the window's top-left
// corner. The backend converts its own events into these.
type PointerEvent struct {
	Kind PointerKind
	Pos  Point
}

// Pointer handles one pointer event: it updates which Pressable is hovered
// and pressed, and runs OnClick on a click. Call it on the UI goroutine,
// before the Frame that should show the result. Events that arrive before
// the first Frame are ignored, because nothing has been laid out yet.
func (a *App) Pointer(ev PointerEvent) {
	a.drainPosted()
	if a.tree == nil {
		return
	}
	// A rebuild may have removed the hovered or pressed node since the
	// last event. A removed node gets no more state changes and no click.
	if a.hover != nil && a.hover.disposed {
		a.hover = nil
	}
	if a.pressed != nil && a.pressed.disposed {
		a.pressed = nil
	}

	if ev.Kind == Leave {
		a.setHover(nil)
		return
	}
	target := a.tree.hitTest(ev.Pos)
	a.setHover(target)

	switch ev.Kind {
	case Press:
		if target != nil {
			a.pressed = target
			setPress(target, func(s *PressState) { s.Pressed = true })
		}
	case Release:
		p := a.pressed
		a.pressed = nil
		if p == nil {
			return
		}
		setPress(p, func(s *PressState) { s.Pressed = false })
		if p == target {
			// Read OnClick from the node's current widget: a rebuild
			// since the press may have given it a new handler.
			if fn := p.widget.(PressableWidget).onClick; fn != nil {
				fn()
			}
		}
	}
}

func (a *App) setHover(n *node) {
	if n == a.hover {
		return
	}
	if a.hover != nil {
		setPress(a.hover, func(s *PressState) { s.Hovered = false })
	}
	a.hover = n
	if n != nil {
		setPress(n, func(s *PressState) { s.Hovered = true })
	}
}

// Cursor is the cursor the backend should show: the hovered Pressable's,
// or CursorDefault.
func (a *App) Cursor() Cursor {
	if h := a.hover; h != nil && !h.disposed {
		return h.widget.(PressableWidget).cursorShape()
	}
	return CursorDefault
}

// hitTest returns the deepest Pressable whose rectangle from the last
// layout contains p, or nil.
func (t *tree) hitTest(p Point) *node {
	if t.root == nil {
		return nil
	}
	return hitNode(t.root, Point{}, p)
}

func hitNode(n *node, parentOrigin Point, p Point) *node {
	origin := Point{X: parentOrigin.X + n.offset.X, Y: parentOrigin.Y + n.offset.Y}
	inside := p.X >= origin.X && p.X < origin.X+n.size.W &&
		p.Y >= origin.Y && p.Y < origin.Y+n.size.H

	// Whatever a clipping Container cuts off is not on screen, so it
	// cannot be hit either.
	if c, ok := n.widget.(ContainerWidget); ok && c.clip && !inside {
		return nil
	}
	// Later children are painted on top, so they get the first chance.
	for i := len(n.children) - 1; i >= 0; i-- {
		if hit := hitNode(n.children[i], origin, p); hit != nil {
			return hit
		}
	}
	if _, ok := n.widget.(PressableWidget); ok && inside {
		return n
	}
	return nil
}
