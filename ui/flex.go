package ui

import "github.com/c-airr/flit/draw"

// Align says where children go along an axis.
type Align int

const (
	Start        Align = iota // top or left
	Center                    // centered
	End                       // bottom or right
	SpaceBetween              // main axis only: first at the start, last at the end, the rest evenly between
	Stretch                   // cross axis only: children take the full cross size
)

// FlexWidget lays its children out in a line: a Row (horizontal) or a
// Column (vertical). Create it with Row or Column.
//
// The main axis is the direction of the line, the cross axis is the other
// one: for a Column, main is vertical and cross is horizontal.
type FlexWidget struct {
	horizontal bool
	items      []Widget
	gap        float32
	main       Align
	cross      Align
}

var _ renderWidget = FlexWidget{}

// Row lays children out left to right.
func Row(children ...Widget) FlexWidget {
	return FlexWidget{horizontal: true, items: nonNil(children)}
}

// Column lays children out top to bottom.
func Column(children ...Widget) FlexWidget {
	return FlexWidget{items: nonNil(children)}
}

// nonNil copies the children without the nil ones, so code like
//
//	var extra ui.Widget // stays nil unless something sets it
//	ui.Column(title, extra)
//
// works. The copy also matters on its own: in Column(ws...) the variadic
// parameter is the caller's slice itself, not a copy, and the widget must
// not change if the caller later writes to ws.
func nonNil(ws []Widget) []Widget {
	out := make([]Widget, 0, len(ws))
	for _, w := range ws {
		if w != nil {
			out = append(out, w)
		}
	}
	return out
}

// Gap sets the space between neighbouring children.
func (w FlexWidget) Gap(v float32) FlexWidget {
	w.gap = v
	return w
}

// MainAlign sets the alignment along the line. Stretch acts as Start.
func (w FlexWidget) MainAlign(a Align) FlexWidget {
	w.main = a
	return w
}

// CrossAlign sets the alignment across the line. SpaceBetween acts as Start.
func (w FlexWidget) CrossAlign(a Align) FlexWidget {
	w.cross = a
	return w
}

func (FlexWidget) isWidget() {}

// children returns the stored slice itself. That is safe because nothing
// writes to it after the constructor: the option methods copy the struct,
// and the copy shares the same backing array, read-only.
func (w FlexWidget) children() []Widget { return w.items }

// layout works in main/cross terms so Row and Column share one code path.
// Children are unbounded on the main axis and get at most the flex's cross
// size. The flex takes the full main size when it is bounded, otherwise
// just what the children need.
func (w FlexWidget) layout(n *node, e *env, c Constraints) Size {
	minMain, maxMain := c.MinH, c.MaxH
	minCross, maxCross := c.MinW, c.MaxW
	if w.horizontal {
		minMain, maxMain = c.MinW, c.MaxW
		minCross, maxCross = c.MinH, c.MaxH
	}

	stretch := w.cross == Stretch && !isInf(maxCross)
	childMinCross := float32(0)
	if stretch {
		childMinCross = maxCross
	}
	childC := w.constraints(0, Inf, childMinCross, maxCross)

	var total, widest float32
	for i, child := range n.children {
		m, x := w.axes(layoutNode(child, e, childC))
		total += m
		if i > 0 {
			total += w.gap
		}
		widest = max(widest, x)
	}

	mainSize := maxMain
	if isInf(maxMain) {
		mainSize = max(total, minMain)
	}
	crossSize := clamp(widest, minCross, maxCross)
	if stretch {
		crossSize = maxCross
	}

	// When the children overflow, free is 0 rather than negative, so
	// nothing moves backwards or overlaps; the extra just sticks out.
	free := max(0, mainSize-total)
	pos, gap := float32(0), w.gap
	switch w.main {
	case Center:
		pos = free / 2
	case End:
		pos = free
	case SpaceBetween:
		if len(n.children) > 1 {
			gap += free / float32(len(n.children)-1)
		}
	}

	for _, child := range n.children {
		m, x := w.axes(child.size)
		var crossPos float32
		switch w.cross {
		case Center:
			crossPos = (crossSize - x) / 2
		case End:
			crossPos = crossSize - x
		}
		child.offset = w.point(pos, crossPos)
		pos += m + gap
	}
	return w.size(mainSize, crossSize)
}

func (FlexWidget) paint(n *node, e *env, origin Point, out *draw.List) {
	paintChildren(n, e, origin, out)
}

// axes splits a size into its main and cross parts. Go functions can
// return several values; the caller unpacks them with m, x := ...,
// like a structured binding over a std::pair in C++.
func (w FlexWidget) axes(s Size) (main, cross float32) {
	if w.horizontal {
		return s.W, s.H
	}
	return s.H, s.W
}

func (w FlexWidget) size(main, cross float32) Size {
	if w.horizontal {
		return Size{W: main, H: cross}
	}
	return Size{W: cross, H: main}
}

func (w FlexWidget) point(main, cross float32) Point {
	if w.horizontal {
		return Point{X: main, Y: cross}
	}
	return Point{X: cross, Y: main}
}

func (w FlexWidget) constraints(minMain, maxMain, minCross, maxCross float32) Constraints {
	if w.horizontal {
		return Constraints{MinW: minMain, MaxW: maxMain, MinH: minCross, MaxH: maxCross}
	}
	return Constraints{MinW: minCross, MaxW: maxCross, MinH: minMain, MaxH: maxMain}
}
