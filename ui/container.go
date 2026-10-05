package ui

import "github.com/c-airr/flit/draw"

// ContainerWidget is a box: an optional fixed size, background, rounded
// corners, border and clipping around an optional child.
// Create it with Container.
type ContainerWidget struct {
	child       Widget
	background  draw.RGBA // zero (fully transparent) = no background
	radius      float32
	borderColor draw.RGBA
	borderWidth float32 // 0 = no border
	width       float32 // 0 = not fixed
	height      float32 // 0 = not fixed
	clip        bool
}

var _ renderWidget = ContainerWidget{}

// Container makes a box around child. child may be nil.
// With no child and no fixed size, the box takes all the space it is allowed.
func Container(child Widget) ContainerWidget {
	return ContainerWidget{child: child}
}

// Background fills the box with c.
func (w ContainerWidget) Background(c draw.RGBA) ContainerWidget {
	w.background = c
	return w
}

// Radius rounds the corners of the background, border and clip.
func (w ContainerWidget) Radius(r float32) ContainerWidget {
	w.radius = r
	return w
}

// Border draws an outline of the given color and width on top of the content.
func (w ContainerWidget) Border(c draw.RGBA, width float32) ContainerWidget {
	w.borderColor = c
	w.borderWidth = width
	return w
}

// Width fixes the box width, within what the parent allows.
func (w ContainerWidget) Width(v float32) ContainerWidget {
	w.width = v
	return w
}

// Height fixes the box height, within what the parent allows.
func (w ContainerWidget) Height(v float32) ContainerWidget {
	w.height = v
	return w
}

// Clip cuts off any part of the child that sticks out of the box.
func (w ContainerWidget) Clip() ContainerWidget {
	w.clip = true
	return w
}

func (ContainerWidget) isWidget() {}

func (w ContainerWidget) children() []Widget {
	if w.child == nil {
		return nil
	}
	return []Widget{w.child}
}

// layout tightens the constraints to the fixed width/height, if set, and
// sizes the box to its child. With no child it takes the max, or the min
// when the max is unbounded, so the size is never Inf.
func (w ContainerWidget) layout(n *node, e *env, c Constraints) Size {
	cc := c
	if w.width > 0 {
		v := clamp(w.width, c.MinW, c.MaxW)
		cc.MinW, cc.MaxW = v, v
	}
	if w.height > 0 {
		v := clamp(w.height, c.MinH, c.MaxH)
		cc.MinH, cc.MaxH = v, v
	}
	if len(n.children) == 1 {
		child := n.children[0]
		s := layoutNode(child, e, cc)
		child.offset = Point{}
		return cc.Constrain(s)
	}
	return Size{W: finiteOr(cc.MaxW, cc.MinW), H: finiteOr(cc.MaxH, cc.MinH)}
}
