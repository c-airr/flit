package ui

import "github.com/c-airr/flit/draw"

// paintNode paints n and its subtree. Offsets are relative to the parent,
// draw commands are absolute, so the origin is accumulated on the way down.
func paintNode(n *node, e *env, parentOrigin Point, out *draw.List) {
	origin := Point{X: parentOrigin.X + n.offset.X, Y: parentOrigin.Y + n.offset.Y}
	switch w := n.widget.(type) {
	case composite:
		paintChildren(n, e, origin, out)
	case renderWidget:
		w.paint(n, e, origin, out)
	}
}

func paintChildren(n *node, e *env, origin Point, out *draw.List) {
	for _, child := range n.children {
		paintNode(child, e, origin, out)
	}
}

// paint replaces the contents of out with the commands for the whole tree.
// Call layout first.
func (t *tree) paint(out *draw.List) {
	out.Reset()
	if t.root == nil {
		return
	}
	paintNode(t.root, &t.env, Point{}, out)
}

// bounds is n's rectangle in absolute coordinates.
func bounds(n *node, origin Point) draw.Rect {
	return draw.Rect{X: origin.X, Y: origin.Y, W: n.size.W, H: n.size.H}
}
