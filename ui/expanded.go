package ui

import "github.com/c-airr/flit/draw"

// ExpandedWidget makes a child of a Row or Column take the space left over
// by its siblings. Create it with Expanded or Spacer.
type ExpandedWidget struct {
	child Widget
	flex  int
}

var _ renderWidget = ExpandedWidget{}

// Expanded makes child fill the space its Row or Column has left after the
// other children. Several Expanded children share that space in proportion
// to their Flex. child may be nil, which leaves empty space.
//
//	ui.Column(header, ui.Expanded(content), footer) // content takes the rest
//
// It also works when a component returns it: a Build or Pressable between
// the Row/Column and the Expanded is looked through. Anywhere else it lays
// out like its child.
func Expanded(child Widget) ExpandedWidget {
	return ExpandedWidget{child: child, flex: 1}
}

// Flex sets this child's share relative to the other Expanded children:
// Flex(2) gets twice as much as Flex(1). Values below 1 count as 1.
func (w ExpandedWidget) Flex(f int) ExpandedWidget {
	w.flex = f
	return w
}

// Spacer is empty space that takes what is left, e.g. to push the next
// children to the end of a Row.
func Spacer() ExpandedWidget {
	return Expanded(nil)
}

func (w ExpandedWidget) factor() int {
	return max(1, w.flex)
}

func (ExpandedWidget) isWidget() {}

func (w ExpandedWidget) children() []Widget {
	if w.child == nil {
		return nil
	}
	return []Widget{w.child}
}

// layout passes the constraints to the child. The sharing out happens in
// the Row or Column, which gives an Expanded tight constraints of its share.
func (w ExpandedWidget) layout(n *node, e *env, c Constraints) Size {
	if !inFlex(n) {
		warnOnce(n, warnNotInFlex)
	}
	if len(n.children) == 1 {
		child := n.children[0]
		s := layoutNode(child, e, c)
		child.offset = Point{}
		return c.Constrain(s)
	}
	return c.Constrain(Size{})
}

func (ExpandedWidget) paint(n *node, e *env, origin Point, out *draw.List) {
	paintChildren(n, e, origin, out)
}

// flexOf returns the flex factor of a Row/Column child: its Expanded's,
// looking down through composites, or 0 if it is not an Expanded.
func flexOf(n *node) int {
	for {
		switch w := n.widget.(type) {
		case ExpandedWidget:
			return w.factor()
		case composite:
			if len(n.children) != 1 {
				return 0
			}
			n = n.children[0]
		default:
			return 0
		}
	}
}

// inFlex reports whether n's nearest ancestor that is not a composite is
// a Row or Column, i.e. whether that Row/Column treats n as a flex child.
func inFlex(n *node) bool {
	for p := n.parent; p != nil; p = p.parent {
		switch p.widget.(type) {
		case composite:
			continue // a Build or Pressable in between is fine; keep going up
		case FlexWidget:
			return true
		default:
			return false
		}
	}
	return false
}
