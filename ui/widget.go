package ui

import "github.com/c-airr/flit/draw"

// Widget is an immutable description of a piece of UI. Widgets are small
// structs, created anew on every build and thrown away; the persistent
// state lives in nodes.
//
// Widget is sealed in v1: isWidget is unexported, so only types in this
// package can have that method and therefore implement Widget. User code
// composes widgets with Build; custom render widgets come after v1.
type Widget interface{ isWidget() }

// renderWidget is implemented by the built-in widgets that lay themselves
// out. Listing Widget inside it (interface embedding) means every
// renderWidget is also a Widget, like inheriting from an abstract base
// class in C++, but without any vtable layout to think about.
type renderWidget interface {
	Widget
	// children returns the child widgets to mount under this one.
	children() []Widget
	// layout sizes the widget under c. It lays out n.children, sets their
	// offsets and returns its own size; layoutNode stores it in n.size.
	layout(n *node, e *env, c Constraints) Size
	// paint appends the widget's draw commands to out. origin is n's
	// top-left corner in absolute coordinates; children are painted with
	// paintChildren.
	paint(n *node, e *env, origin Point, out *draw.List)
}

// env is what layout and paint need from outside the tree.
type env struct {
	measurer draw.TextMeasurer
	theme    Theme
}
