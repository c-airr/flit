package ui

import (
	"fmt"
	"reflect"
)

// Hooks give a component state of its own. A composite node keeps a list
// of slots; every build walks it from the start, and each hook call takes
// the next slot. That is why hooks must be called in the same order on
// every build: the order is the only thing that ties a call to its slot.

const hookRule = "call hooks in the same order every build, never inside if or for"

// hookSlot is one stored hook value plus the hook's name with its type,
// e.g. "UseSignal[int]", which the next build's call must match.
type hookSlot struct {
	name  string
	value any
}

// UseSignal returns a signal that belongs to the building component and
// survives its rebuilds, so a component can have state of its own:
//
//	ui.Build(func(ctx *ui.Ctx) ui.Widget {
//		open := ui.UseSignal(ctx, false)
//		...
//	})
//
// initial is used only in the component's first build. If it comes from
// data the parent passes in, later changes of that data do not reach the
// signal; read such data directly in the build instead.
//
// Call it only directly in a Build or Pressable function, the same number
// of times and in the same order on every build; flit panics otherwise.
func UseSignal[T any](ctx *Ctx, initial T) *Signal[T] {
	// reflect.TypeFor[T]() is T's type as a value, so the slot can
	// remember it, like typeid(T) in C++.
	v := useSlot(ctx, "UseSignal", reflect.TypeFor[T](), func() any { return NewSignal(initial) })
	return v.(*Signal[T])
}

// Remember returns a value that belongs to the building component. init
// runs only in the component's first build; every later build gets the
// stored value back. Use it for things that should be created once per
// component, such as a cache. The same rules as for UseSignal apply.
func Remember[T any](ctx *Ctx, init func() T) T {
	v := useSlot(ctx, "Remember", reflect.TypeFor[T](), func() any { return init() })
	// The comma-ok form: if T is an interface type and init returned nil,
	// the slot holds nil, and this gives the zero T instead of panicking.
	t, _ := v.(T)
	return t
}

// useSlot returns the value in the next slot of the building node,
// creating it with init in the node's first build. It panics when the
// call does not match the slot the previous build left there.
func useSlot(ctx *Ctx, hook string, t reflect.Type, init func() any) any {
	if ctx == nil || ctx.node == nil || ctx.done {
		panic(fmt.Sprintf("ui: %s called outside a build; hooks only work while a Build or Pressable function runs", hook))
	}
	name := hook + "[" + t.String() + "]"
	n := ctx.node
	i := ctx.next
	ctx.next++

	if i < len(n.hooks) {
		if prev := n.hooks[i].name; prev != name {
			panic(fmt.Sprintf("ui: hook #%d was %s in the previous build and is %s now; %s", i, prev, name, hookRule))
		}
		return n.hooks[i].value
	}
	if n.hooksKnown {
		panic(fmt.Sprintf("ui: hook #%d is new in this build (the previous build made %d hook calls); %s", i, len(n.hooks), hookRule))
	}
	v := init()
	n.hooks = append(n.hooks, hookSlot{name: name, value: v})
	return v
}
