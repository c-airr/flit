package ui

import "github.com/c-airr/flit/draw"

// optional is a value plus whether it was set. It lets a zero value (a
// transparent color, no padding) be chosen on purpose instead of meaning
// "use the default".
//
// optional[T] is a generic struct: optional[float32] and optional[Insets]
// are two different types made from one definition, as with a C++ class
// template.
type optional[T any] struct {
	value T
	set   bool
}

func some[T any](v T) optional[T] {
	return optional[T]{value: v, set: true}
}

// or returns the value if it was set, else def.
func (o optional[T]) or(def T) T {
	if o.set {
		return o.value
	}
	return def
}

// ButtonWidget is a ready-made clickable button. Create it with Button.
type ButtonWidget struct {
	label     string
	onClick   func()
	bg        optional[draw.RGBA]
	hoverBg   optional[draw.RGBA]
	pressedBg optional[draw.RGBA]
	textColor optional[draw.RGBA]
	radius    optional[float32]
	padding   optional[Insets]
	fontSize  optional[float32]
	weight    optional[uint32]
}

var _ composite = ButtonWidget{}

// Button makes a button with a text label. Every part of its look can be
// changed with the option methods; whatever is not set comes from the
// theme. For a look the options cannot express, build your own with
// Pressable, which is all Button is made of. onClick may be nil.
func Button(label string, onClick func()) ButtonWidget {
	return ButtonWidget{label: label, onClick: onClick}
}

// Background sets the background color. If it is set and HoverBackground
// or PressedBackground are not, those states keep this color too.
func (w ButtonWidget) Background(c draw.RGBA) ButtonWidget {
	w.bg = some(c)
	return w
}

// HoverBackground sets the background color while the pointer is over the button.
func (w ButtonWidget) HoverBackground(c draw.RGBA) ButtonWidget {
	w.hoverBg = some(c)
	return w
}

// PressedBackground sets the background color while the button is held down.
func (w ButtonWidget) PressedBackground(c draw.RGBA) ButtonWidget {
	w.pressedBg = some(c)
	return w
}

// TextColor sets the label color.
func (w ButtonWidget) TextColor(c draw.RGBA) ButtonWidget {
	w.textColor = some(c)
	return w
}

// Radius sets the corner radius.
func (w ButtonWidget) Radius(r float32) ButtonWidget {
	w.radius = some(r)
	return w
}

// Padding sets the space between the edge and the label.
func (w ButtonWidget) Padding(in Insets) ButtonWidget {
	w.padding = some(in)
	return w
}

// FontSize sets the label size in dp.
func (w ButtonWidget) FontSize(v float32) ButtonWidget {
	w.fontSize = some(v)
	return w
}

// Weight sets the label weight: 400 regular, 600 semibold.
func (w ButtonWidget) Weight(v uint32) ButtonWidget {
	w.weight = some(v)
	return w
}

func (ButtonWidget) isWidget() {}

// build resolves the options against the theme and returns a Pressable
// that picks the background from its state.
func (w ButtonWidget) build(_ *node, ctx *Ctx) Widget {
	th := ctx.Theme()
	normal := w.bg.or(th.Primary)
	hover, pressed := th.PrimaryHover, th.PrimaryPressed
	if w.bg.set {
		hover, pressed = normal, normal
	}
	hover = w.hoverBg.or(hover)
	pressed = w.pressedBg.or(pressed)

	label := Text(w.label).
		Size(w.fontSize.or(th.FontSize)).
		Color(w.textColor.or(th.OnPrimary)).
		Weight(w.weight.or(600))
	pad := w.padding.or(Symmetric(2*th.Spacing, 4*th.Spacing))
	radius := w.radius.or(th.Radius)

	// The function below is a closure: it captures normal, hover, pressed,
	// label, pad and radius from this build, and Pressable calls it again
	// with a new state on every hover or press.
	return Pressable(func(_ *Ctx, s PressState) Widget {
		bg := normal
		switch {
		case s.Pressed:
			bg = pressed
		case s.Hovered:
			bg = hover
		}
		return Container(Padding(pad, label)).Background(bg).Radius(radius)
	}).OnClick(w.onClick)
}
