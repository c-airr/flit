package ui

import "github.com/c-airr/flit/draw"

// PaddingWidget puts empty space around its child. Create it with Padding.
type PaddingWidget struct {
	insets Insets
	child  Widget
}

var _ renderWidget = PaddingWidget{}

// Padding surrounds child with the given insets. child may be nil, which
// leaves just the empty space.
func Padding(in Insets, child Widget) PaddingWidget {
	return PaddingWidget{insets: in, child: child}
}

func (PaddingWidget) isWidget() {}

func (w PaddingWidget) children() []Widget {
	if w.child == nil {
		return nil
	}
	return []Widget{w.child}
}

// layout gives the child what is left after the insets, places it at the
// top-left inset, and adds the insets back to the child's size.
func (w PaddingWidget) layout(n *node, e *env, c Constraints) Size {
	in := w.insets
	var inner Size
	if len(n.children) == 1 {
		child := n.children[0]
		inner = layoutNode(child, e, c.Deflate(in))
		child.offset = Point{X: in.Left, Y: in.Top}
	}
	return c.Constrain(Size{W: inner.W + in.Left + in.Right, H: inner.H + in.Top + in.Bottom})
}

func (PaddingWidget) paint(n *node, e *env, origin Point, out *draw.List) {
	paintChildren(n, e, origin, out)
}
