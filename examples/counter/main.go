// Command counter is flit's v1 example: a number and a button that adds one.
package main

import (
	"strconv"

	"github.com/c-airr/flit/backend/gio"
	"github.com/c-airr/flit/ui"
)

func main() {
	count := ui.NewSignal(0)

	root := ui.Build(func(ctx *ui.Ctx) ui.Widget {
		th := ctx.Theme()
		return ui.Container(
			ui.Column(
				// Only this Build reads count, so a click rebuilds just the
				// number, not the whole window.
				ui.Build(func(*ui.Ctx) ui.Widget {
					return ui.Text(strconv.Itoa(count.Get())).Size(48).Weight(600)
				}),
				ui.Button("+", func() {
					count.Update(func(v int) int { return v + 1 })
				}),
			).Gap(16).MainAlign(ui.Center).CrossAlign(ui.Center),
		).Background(th.Background)
	})

	gio.Run(ui.NewApp(root), gio.Options{Title: "flit counter", Width: 320, Height: 240})
}
