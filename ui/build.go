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

// Build makes a composite widget. fn runs when the widget is mounted and
// on every rebuild, and returns the widget tree to show; it may return nil.
//
//	ui.Build(func(ctx *ui.Ctx) ui.Widget {
//		return ui.Text("Hello").Color(ctx.Theme().Primary)
//	})
func Build(fn func(ctx *Ctx) Widget) BuildWidget {
	return BuildWidget{fn: fn}
}

func (BuildWidget) isWidget() {}
