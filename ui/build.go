package ui

// Ctx is passed to build functions. It gives access to the app's theme;
// later it will carry more (the app, component-local state).
type Ctx struct {
	env *env
}

// Theme returns the theme the tree is built with.
func (c *Ctx) Theme() Theme {
	return c.env.theme
}

// BuildWidget is a composite widget: it describes its UI by calling a
// function. It is how user code writes components.
type BuildWidget struct {
	fn func(ctx *Ctx) Widget
}

var _ composite = BuildWidget{}

// Build makes a composite widget. fn runs when the widget is mounted, on
// every rebuild of an ancestor, and whenever a signal it read changes. It
// returns the widget tree to show; it may return nil.
//
//	ui.Build(func(ctx *ui.Ctx) ui.Widget {
//		return ui.Text(strconv.Itoa(count.Get())).Color(ctx.Theme().Primary)
//	})
func Build(fn func(ctx *Ctx) Widget) BuildWidget {
	return BuildWidget{fn: fn}
}

func (BuildWidget) isWidget() {}

func (w BuildWidget) build(_ *node, ctx *Ctx) Widget {
	return w.fn(ctx)
}
