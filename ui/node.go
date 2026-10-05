package ui

import (
	"cmp"
	"reflect"
	"slices"

	"github.com/c-airr/flit/draw"
)

// node is the persistent instance behind a widget. Widgets are rebuilt all
// the time; a node lives as long as reconciliation keeps matching it, and
// holds what must survive between builds: layout results, per-widget state
// and signal subscriptions.
type node struct {
	widget   Widget
	tree     *tree
	parent   *node
	children []*node
	depth    int // root is 0; rebuilds go shallowest first

	size   Size  // set by layout
	offset Point // top-left corner relative to the parent, set by the parent's layout

	// state is per-widget data. any is the empty interface (interface{}):
	// it holds a value of any type, a bit like std::any, and a type
	// assertion such as n.state.(float32) gets the value back out.
	state any

	dirty    bool         // a signal it read has changed; rebuild in the next frame
	deps     []dependency // signals read by the last build
	disposed bool
}

// composite is implemented by widgets that produce one child by running
// code (Build, and later Pressable and Button). They have no layout or
// paint of their own: both pass straight through to the child.
type composite interface {
	Widget
	build(n *node, ctx *Ctx) Widget
}

// currentBuild is the composite node whose build is running, so that
// Signal.Get knows whom to subscribe. It is a plain package variable
// because all building happens on the single UI goroutine (spec §5).
var currentBuild *node

// mount creates the node for w and, recursively, its whole subtree.
func mount(w Widget, parent *node, t *tree) *node {
	if w == nil {
		return nil
	}
	n := &node{widget: w, tree: t, parent: parent}
	if parent != nil {
		n.depth = parent.depth + 1
	}
	for _, cw := range childWidgets(n) {
		n.children = append(n.children, mount(cw, n, t))
	}
	return n
}

// update gives n a new widget of the same type and reconciles its children
// against the new child widgets. For a composite this reruns its build.
func (n *node) update(w Widget) {
	n.widget = w
	ws := childWidgets(n)
	old := n.children
	kids := make([]*node, len(ws))
	for i, cw := range ws {
		if i < len(old) && sameType(old[i].widget, cw) {
			old[i].update(cw)
			kids[i] = old[i]
			continue
		}
		if i < len(old) {
			old[i].dispose()
		}
		kids[i] = mount(cw, n, n.tree)
	}
	for i := len(ws); i < len(old); i++ {
		old[i].dispose()
	}
	n.children = kids
}

// dispose marks n and its whole subtree as dead, detaches it from the tree
// and unsubscribes it from its signals. Without the unsubscribe a signal
// would keep the dead node reachable, so the GC could never free it, and
// every Set would keep marking it dirty.
func (n *node) dispose() {
	n.disposed = true
	n.parent = nil
	n.dropDeps()
	for _, c := range n.children {
		c.dispose()
	}
}

func (n *node) dropDeps() {
	for _, d := range n.deps {
		d.unsubscribe(n)
	}
	n.deps = nil
}

// markDirty schedules n for a rebuild in the next frame.
func (n *node) markDirty() {
	if n.dirty || n.disposed {
		return
	}
	n.dirty = true
	n.tree.dirty = append(n.tree.dirty, n)
}

// sameType compares the dynamic types of two widgets, i.e. the concrete
// struct types behind the interfaces, like comparing typeid(*a) and
// typeid(*b) in C++.
func sameType(a, b Widget) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

// childWidgets gets n's child widgets: by running the build of a composite,
// from children() for render widgets.
//
// The switch below is a type switch: it branches on the dynamic type
// inside the interface, and in each case w already has that type.
func childWidgets(n *node) []Widget {
	switch w := n.widget.(type) {
	case composite:
		if child := runBuild(n, w); child != nil {
			return []Widget{child}
		}
		return nil
	case renderWidget:
		return w.children()
	}
	return nil
}

// runBuild runs a composite's build with dependency tracking: the old
// subscriptions are dropped first, so after the build n depends on exactly
// the signals this build read.
func runBuild(n *node, w composite) Widget {
	n.dropDeps()
	n.dirty = false
	prev := currentBuild
	currentBuild = n
	// defer runs when runBuild returns, even if build panics, so a panic
	// in user code cannot leave currentBuild pointing at this node.
	defer func() { currentBuild = prev }()
	return w.build(n, &Ctx{env: &n.tree.env})
}

// layoutNode lays out n under c, stores the result in n.size and returns it.
// A composite has no layout of its own: it gives its constraints to its child.
func layoutNode(n *node, e *env, c Constraints) Size {
	var s Size
	switch w := n.widget.(type) {
	case composite:
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

// tree owns the root node, the environment the tree is laid out in, and
// the list of nodes waiting for a rebuild.
type tree struct {
	root  *node
	env   env
	dirty []*node
}

func newTree(root Widget, m draw.TextMeasurer, th Theme) *tree {
	t := &tree{env: env{measurer: m, theme: th}}
	t.root = mount(root, nil, t)
	return t
}

// setRoot reconciles the tree against a new root widget.
func (t *tree) setRoot(w Widget) {
	if t.root != nil && sameType(t.root.widget, w) {
		t.root.update(w)
		return
	}
	if t.root != nil {
		t.root.dispose()
	}
	t.root = mount(w, nil, t)
}

// rebuildDirty reruns the builds of the dirty nodes, shallowest first.
// Rebuilding a node also rebuilds the composites below it, which clears
// their dirty flag, so they are skipped when their turn comes. Nodes marked
// dirty while this runs (a build that calls Set) wait for the next call,
// so this can never loop forever.
func (t *tree) rebuildDirty() {
	todo := t.dirty
	t.dirty = nil
	// SortFunc takes a comparison function, like std::sort with a lambda;
	// cmp.Compare returns -1, 0 or +1.
	slices.SortFunc(todo, func(a, b *node) int { return cmp.Compare(a.depth, b.depth) })
	for _, n := range todo {
		if n.dirty && !n.disposed {
			n.update(n.widget)
		}
	}
}

// layout lays out the whole tree; the root fills the viewport exactly.
func (t *tree) layout(viewport Size) {
	if t.root == nil {
		return
	}
	layoutNode(t.root, &t.env, Tight(viewport))
	t.root.offset = Point{}
}
