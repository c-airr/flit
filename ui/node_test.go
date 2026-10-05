package ui

import (
	"testing"

	"github.com/c-airr/flit/internal/fakebackend"
)

func newTestTree(root Widget) *tree {
	return newTree(root, fakebackend.Measurer{}, DefaultTheme())
}

// textOf reads the string out of a node holding a TextWidget.
// n.widget.(TextWidget) is a type assertion, Go's dynamic_cast: it panics
// if the widget is some other type, which in a test is a clear failure.
func textOf(n *node) string {
	return n.widget.(TextWidget).text
}

func TestReconcileKeepsNodesOfSameType(t *testing.T) {
	tr := newTestTree(Container(Text("a")))
	root, child := tr.root, tr.root.children[0]

	tr.setRoot(Container(Text("b")))

	if tr.root != root {
		t.Error("root node was replaced, want the same node")
	}
	if len(tr.root.children) != 1 || tr.root.children[0] != child {
		t.Fatal("child node was replaced, want the same node")
	}
	if got := textOf(child); got != "b" {
		t.Errorf("child text = %q, want %q", got, "b")
	}
	if child.disposed {
		t.Error("kept child is marked disposed")
	}
}

func TestReconcileReplacesNodeOfOtherType(t *testing.T) {
	tr := newTestTree(Container(Text("a")))
	old := tr.root.children[0]

	tr.setRoot(Container(Padding(All(1), nil)))

	if !old.disposed || old.parent != nil {
		t.Errorf("old Text node: disposed=%v parent=%p, want disposed and no parent", old.disposed, old.parent)
	}
	if len(tr.root.children) != 1 {
		t.Fatalf("root has %d children, want 1", len(tr.root.children))
	}
	nw := tr.root.children[0]
	if _, ok := nw.widget.(PaddingWidget); !ok || nw == old {
		t.Errorf("child is %T (same node: %v), want a new PaddingWidget node", nw.widget, nw == old)
	}
	if nw.parent != tr.root {
		t.Error("new child's parent is not the root")
	}
	if tr.root.depth != 0 || nw.depth != 1 {
		t.Errorf("depths root=%d child=%d, want 0 and 1", tr.root.depth, nw.depth)
	}
}

func TestDisposeIsRecursive(t *testing.T) {
	tr := newTestTree(Container(Padding(All(1), Text("a"))))
	pad := tr.root.children[0]
	text := pad.children[0]

	tr.setRoot(Container(Text("b")))

	if !pad.disposed || !text.disposed {
		t.Errorf("disposed: padding=%v text=%v, want both", pad.disposed, text.disposed)
	}
}

func TestRemovedChildIsDisposed(t *testing.T) {
	tr := newTestTree(Container(Text("a")))
	old := tr.root.children[0]

	tr.setRoot(Container(nil))

	if len(tr.root.children) != 0 {
		t.Errorf("root has %d children, want 0", len(tr.root.children))
	}
	if !old.disposed {
		t.Error("removed child is not disposed")
	}
}

func TestSetRootOfOtherTypeReplacesRoot(t *testing.T) {
	tr := newTestTree(Container(nil))
	old := tr.root

	tr.setRoot(Padding(All(1), nil))

	if !old.disposed || tr.root == old {
		t.Error("old root not replaced and disposed")
	}
	if _, ok := tr.root.widget.(PaddingWidget); !ok {
		t.Errorf("root is %T, want PaddingWidget", tr.root.widget)
	}
}

func TestBuildMountsItsResult(t *testing.T) {
	tr := newTestTree(Build(func(*Ctx) Widget { return Text("a") }))

	if _, ok := tr.root.widget.(BuildWidget); !ok {
		t.Fatalf("root is %T, want BuildWidget", tr.root.widget)
	}
	if len(tr.root.children) != 1 {
		t.Fatalf("Build node has %d children, want 1", len(tr.root.children))
	}
	child := tr.root.children[0]
	if got := textOf(child); got != "a" || child.depth != 1 {
		t.Errorf("child text=%q depth=%d, want \"a\" and 1", got, child.depth)
	}
}

func TestBuildRunsAgainOnUpdate(t *testing.T) {
	tr := newTestTree(Build(func(*Ctx) Widget { return Text("a") }))
	child := tr.root.children[0]

	tr.setRoot(Build(func(*Ctx) Widget { return Text("b") }))

	if tr.root.children[0] != child {
		t.Fatal("Build child node was replaced, want the same node")
	}
	if got := textOf(child); got != "b" {
		t.Errorf("child text = %q, want %q", got, "b")
	}
}

func TestBuildReturningNilHasNoChild(t *testing.T) {
	tr := newTestTree(Build(func(*Ctx) Widget { return nil }))
	tr.layout(Size{W: 200, H: 100})

	if len(tr.root.children) != 0 {
		t.Errorf("Build node has %d children, want 0", len(tr.root.children))
	}
}

func TestCtxThemeIsTheTreeTheme(t *testing.T) {
	th := DefaultTheme()
	th.Radius = 3
	var got Theme
	newTree(Build(func(ctx *Ctx) Widget {
		got = ctx.Theme()
		return nil
	}), fakebackend.Measurer{}, th)

	if got != th {
		t.Errorf("ctx.Theme() = %+v, want %+v", got, th)
	}
}
