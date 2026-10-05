package ui

import (
	"reflect"

	"github.com/c-airr/flit/draw"
)

// node is the persistent instance behind a widget. Widgets are rebuilt all
// the time; a node lives as long as reconciliation keeps matching it, and
// holds what must survive between builds: layout results and per-widget
// state.
type node struct {
	widget   Widget
	parent   *node
	children []*node
	depth    int // root is 0; rebuilds go shallowest first

	size   Size  // set by layout
	offset Point // top-left corner relative to the parent, set by the parent's layout

	// state is per-widget data. any is the empty interface (interface{}):
	// it holds a value of any type, a bit like std::any, and a type
	// assertion such as n.state.(float32) gets the value back out.
	state any

	disposed bool
}

// mount creates the node for w and, recursively, its whole subtree.
func mount(w Widget, parent *node, e *env) *node {
	if w == nil {
		return nil
	}
	n := &node{widget: w, parent: parent}
	if parent != nil {
		n.depth = parent.depth + 1
	}
	for _, cw := range childWidgets(n, e) {
		n.children = append(n.children, mount(cw, n, e))
	}
	return n
}

// update gives n a new widget of the same type and reconciles its children
// against the new child widgets.
func (n *node) update(w Widget, e *env) {
	n.widget = w
	ws := childWidgets(n, e)
	old := n.children
	kids := make([]*node, len(ws))
	for i, cw := range ws {
		if i < len(old) && sameType(old[i].widget, cw) {
			old[i].update(cw, e)
			kids[i] = old[i]
			continue
		}
		if i < len(old) {
			old[i].dispose()
		}
		kids[i] = mount(cw, n, e)
	}
	for i := len(ws); i < len(old); i++ {
		old[i].dispose()
	}
	n.children = kids
}

// dispose marks n and its whole subtree as dead and detaches it from the
// tree. From step 5 on it also unsubscribes the nodes from signals:
// a signal would otherwise keep a dead node reachable, and the GC would
// never free it.
func (n *node) dispose() {
	n.disposed = true
	n.parent = nil
	for _, c := range n.children {
		c.dispose()
	}
}

// sameType compares the dynamic types of two widgets, i.e. the concrete
// struct types behind the interfaces, like comparing typeid(*a) and
// typeid(*b) in C++.
func sameType(a, b Widget) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

// childWidgets gets n's child widgets: from the build function for Build,
// from children() for render widgets.
//
// The switch below is a type switch: it branches on the dynamic type
// inside the interface, and in each case w already has that type.
func childWidgets(n *node, e *env) []Widget {
	switch w := n.widget.(type) {
	case BuildWidget:
		if child := w.fn(&Ctx{env: e}); child != nil {
			return []Widget{child}
		}
		return nil
	case renderWidget:
		return w.children()
	}
	return nil
}

// layoutNode lays out n under c, stores the result in n.size and returns it.
// Build has no layout of its own: it gives its constraints to its child.
func layoutNode(n *node, e *env, c Constraints) Size {
	var s Size
	switch w := n.widget.(type) {
	case BuildWidget:
		if len(n.children) == 1 {
			child := n.children[0]
			s = layoutNode(child, e, c)
			child.offset = Point{}
		} else {
			s = c.Constrain(Size{})
		}
	case renderWidget:
		s = w.layout(n, e, c)
	}
	n.size = s
	return s
}

// tree owns the root node and the environment the whole tree is laid out in.
type tree struct {
	root *node
	env  env
}

func newTree(root Widget, m draw.TextMeasurer, th Theme) *tree {
	t := &tree{env: env{measurer: m, theme: th}}
	t.root = mount(root, nil, &t.env)
	return t
}

// setRoot reconciles the tree against a new root widget.
func (t *tree) setRoot(w Widget) {
	if t.root != nil && sameType(t.root.widget, w) {
		t.root.update(w, &t.env)
		return
	}
	if t.root != nil {
		t.root.dispose()
	}
	t.root = mount(w, nil, &t.env)
}

// layout lays out the whole tree; the root fills the viewport exactly.
func (t *tree) layout(viewport Size) {
	if t.root == nil {
		return
	}
	layoutNode(t.root, &t.env, Tight(viewport))
	t.root.offset = Point{}
}
