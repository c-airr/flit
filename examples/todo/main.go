// Command todo is a small to-do list: one signal holds the tasks, Expanded
// and Spacer lay the window out, and a hook keeps the "hide done" toggle.
// Adding tasks uses a button for now; a text field comes with keys.
package main

import (
	"fmt"
	"slices"

	"github.com/c-airr/flit/backend/gio"
	"github.com/c-airr/flit/ui"
)

type Task struct {
	Title string
	Done  bool
}

func main() {
	// All tasks live in one signal. Changes replace the slice with a new
	// one instead of editing it in place: Set needs a new value to store,
	// and widgets built from the old slice must not see it change.
	tasks := ui.NewSignal([]Task{
		{Title: "Write the spec", Done: true},
		{Title: "Build the to-do example"},
		{Title: "Add a text field"},
	})

	root := ui.Build(func(ctx *ui.Ctx) ui.Widget {
		th := ctx.Theme()
		return ui.Container(
			ui.Padding(ui.All(16), ui.Column(
				header(tasks),
				// The list takes all the height the header and footer leave.
				// Clip hides rows that do not fit; scrolling comes later.
				ui.Expanded(ui.Container(taskList(tasks)).Clip()),
				ui.Row(
					ui.Button("Add task", func() { addTask(tasks) }),
					ui.Spacer(),
				),
			).Gap(12).CrossAlign(ui.Stretch)),
		).Background(th.Background)
	})

	gio.Run(ui.NewApp(root), gio.Options{Title: "flit todo", Width: 420, Height: 360})
}

// header shows the title and, pushed to the right edge by a Spacer, how
// many tasks are done. It is its own Build, so only it and the list
// rebuild when the tasks change.
func header(tasks *ui.Signal[[]Task]) ui.Widget {
	return ui.Build(func(ctx *ui.Ctx) ui.Widget {
		th := ctx.Theme()
		list := tasks.Get()
		done := 0
		for _, t := range list {
			if t.Done {
				done++
			}
		}
		return ui.Row(
			ui.Text("Tasks").Size(th.TitleSize).Weight(600),
			ui.Spacer(),
			ui.Text(fmt.Sprintf("%d / %d done", done, len(list))).Color(th.TextMuted),
		).CrossAlign(ui.Center)
	})
}

// taskList is a component with state of its own: whether done tasks are
// hidden. The rows below get no hooks: until flit has keys, a row's state
// would stay with its position, not with its task.
func taskList(tasks *ui.Signal[[]Task]) ui.Widget {
	return ui.Build(func(ctx *ui.Ctx) ui.Widget {
		th := ctx.Theme()
		hide := ui.UseSignal(ctx, false)

		label := "Hide done"
		if hide.Get() {
			label = "Show done"
		}
		rows := []ui.Widget{
			ui.Row(
				ui.Spacer(),
				ui.Button(label, func() { hide.Update(func(v bool) bool { return !v }) }).
					Background(th.Surface).
					HoverBackground(th.Outline).
					TextColor(th.Text).
					Padding(ui.Symmetric(4, 10)),
			),
		}
		for i, t := range tasks.Get() {
			if hide.Get() && t.Done {
				continue
			}
			// Since Go 1.22 every loop iteration has its own i and t, so
			// each row's closures see their own task.
			rows = append(rows, taskRow(t,
				func() { toggleTask(tasks, i) },
				func() { removeTask(tasks, i) }))
		}
		return ui.Column(rows...).Gap(8).CrossAlign(ui.Stretch)
	})
}

// taskRow is a plain function of its task: a checkbox, the title taking
// the middle, and a remove button at the right edge.
func taskRow(t Task, toggle, remove func()) ui.Widget {
	return ui.Build(func(ctx *ui.Ctx) ui.Widget {
		th := ctx.Theme()
		title := ui.Text(t.Title)
		if t.Done {
			title = title.Color(th.TextMuted)
		}
		return ui.Container(ui.Padding(ui.Symmetric(8, 12), ui.Row(
			checkbox(t.Done, toggle),
			ui.Expanded(title),
			ui.Button("×", remove).
				Background(th.Surface).
				HoverBackground(th.Background).
				PressedBackground(th.Outline).
				TextColor(th.TextMuted).
				Padding(ui.Symmetric(0, 8)).
				FontSize(18),
		).Gap(12).CrossAlign(ui.Center))).
			Background(th.Surface).
			Radius(th.Radius).
			Border(th.Outline, 1)
	})
}

// checkbox is drawn by hand with Pressable: the app decides the look of
// every state.
func checkbox(checked bool, onClick func()) ui.Widget {
	return ui.Pressable(func(ctx *ui.Ctx, s ui.PressState) ui.Widget {
		th := ctx.Theme()
		bg, border := th.Surface, th.Outline
		if s.Hovered {
			border = th.Primary
		}
		var mark ui.Widget
		if checked {
			bg, border = th.Primary, th.Primary
			mark = ui.Column(ui.Text("✓").Color(th.OnPrimary).Weight(600)).
				MainAlign(ui.Center).CrossAlign(ui.Center)
		}
		return ui.Container(mark).Width(20).Height(20).Radius(6).Background(bg).Border(border, 1.5)
	}).OnClick(onClick)
}

func addTask(tasks *ui.Signal[[]Task]) {
	tasks.Update(func(old []Task) []Task {
		// slices.Clone makes a new backing array, so append cannot write
		// into the slice the current widgets were built from.
		return append(slices.Clone(old), Task{Title: fmt.Sprintf("Task %d", len(old)+1)})
	})
}

func toggleTask(tasks *ui.Signal[[]Task], i int) {
	tasks.Update(func(old []Task) []Task {
		list := slices.Clone(old)
		list[i].Done = !list[i].Done
		return list
	})
}

func removeTask(tasks *ui.Signal[[]Task], i int) {
	tasks.Update(func(old []Task) []Task {
		return slices.Delete(slices.Clone(old), i, i+1)
	})
}
