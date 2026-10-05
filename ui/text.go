package ui

import "github.com/c-airr/flit/draw"

// TextWidget shows a string. Create it with Text.
type TextWidget struct {
	text     string
	fontSize float32   // 0 = theme FontSize
	color    draw.RGBA // zero = theme Text color
	weight   uint32    // 0 = 400 (regular)
}

// Compile-time check that TextWidget is a renderWidget. Without it a typo
// in a method signature would make the type switch in layoutNode silently
// skip this widget.
var _ renderWidget = TextWidget{}

// Text makes a text widget with the theme's default size, color and weight.
func Text(s string) TextWidget {
	return TextWidget{text: s}
}

// Size sets the font size in logical pixels.
//
// The option methods have value receivers: w is a copy, so Size changes the
// copy and returns it, and the original stays untouched. That is what keeps
// widgets immutable and lets calls chain: Text("a").Size(20).Weight(600).
func (w TextWidget) Size(v float32) TextWidget {
	w.fontSize = v
	return w
}

// Color sets the text color.
func (w TextWidget) Color(c draw.RGBA) TextWidget {
	w.color = c
	return w
}

// Weight sets the font weight: 400 regular, 600 semibold.
func (w TextWidget) Weight(v uint32) TextWidget {
	w.weight = v
	return w
}

func (TextWidget) isWidget() {}

func (TextWidget) children() []Widget { return nil }

// layout measures the text, wrapping at the max width if there is one.
// The max width is kept in n.state for painting: the renderer must wrap
// the text at the same width that was measured.
func (w TextWidget) layout(n *node, e *env, c Constraints) Size {
	maxW := finiteOr(c.MaxW, 0) // 0 tells the measurer not to wrap
	n.state = maxW
	return c.Constrain(e.measurer.Measure(w.text, w.style(e.theme), maxW))
}

// paint emits one Text command. Its W is the max width used for measuring,
// not the measured width, so the renderer wraps at the same place.
func (w TextWidget) paint(n *node, e *env, origin Point, out *draw.List) {
	// The two-value form of a type assertion does not panic: ok is false
	// and maxW is 0 if state holds no float32 (layout has not run).
	maxW, _ := n.state.(float32)
	r := draw.Rect{X: origin.X, Y: origin.Y, W: maxW, H: n.size.H}
	c := w.color
	if c == (draw.RGBA{}) {
		c = e.theme.Text
	}
	out.Text(r, w.text, c, w.style(e.theme))
}

func (w TextWidget) style(th Theme) draw.FontStyle {
	s := draw.FontStyle{Size: w.fontSize, Weight: w.weight}
	if s.Size == 0 {
		s.Size = th.FontSize
	}
	if s.Weight == 0 {
		s.Weight = 400
	}
	return s
}
