package ui

import (
	"strconv"
	"testing"
)

func TestSignalGetSetUpdate(t *testing.T) {
	s := NewSignal(1) // Signal[int]: the type parameter is inferred from 1
	s.Set(2)
	if got := s.Get(); got != 2 {
		t.Errorf("after Set(2), Get() = %d", got)
	}
	s.Update(func(v int) int { return v * 10 })
	if got := s.Get(); got != 20 {
		t.Errorf("after Update(*10), Get() = %d, want 20", got)
	}

	var z Signal[string] // the zero value must be usable
	z.Set("x")
	if got := z.Get(); got != "x" {
		t.Errorf("zero Signal: Get() = %q, want %q", got, "x")
	}
}

func TestOnlyDependentBuildReruns(t *testing.T) {
	count := NewSignal(0)
	var aRuns, bRuns int
	tr := newTestTree(Column(
		Build(func(*Ctx) Widget {
			aRuns++
			return Text(strconv.Itoa(count.Get()))
		}),
		Build(func(*Ctx) Widget {
			bRuns++
			return Text("static")
		}),
	))
	aText := tr.root.children[0].children[0]

	count.Set(5)
	tr.rebuildDirty()

	if aRuns != 2 || bRuns != 1 {
		t.Errorf("runs: A=%d B=%d, want A=2 B=1", aRuns, bRuns)
	}
	if tr.root.children[0].children[0] != aText {
		t.Error("A's Text node was replaced, want the same node")
	}
	if got := textOf(aText); got != "5" {
		t.Errorf("A's text = %q, want %q", got, "5")
	}
}

func TestGetOutsideBuildDoesNotSubscribe(t *testing.T) {
	s := NewSignal(0)
	tr := newTestTree(Text("x"))
	s.Get()
	s.Set(1)
	if len(tr.dirty) != 0 {
		t.Errorf("%d dirty nodes, want 0", len(tr.dirty))
	}
}

func TestDisposedNodeIsUnsubscribed(t *testing.T) {
	show := NewSignal(true)
	sig := NewSignal(0)
	inner := Build(func(*Ctx) Widget {
		sig.Get()
		return nil
	})
	tr := newTestTree(Build(func(*Ctx) Widget {
		if show.Get() {
			return Column(inner)
		}
		return Column()
	}))
	innerNode := tr.root.children[0].children[0]

	show.Set(false)
	tr.rebuildDirty()

	if !innerNode.disposed {
		t.Fatal("inner Build node is not disposed")
	}
	if len(sig.subs) != 0 {
		t.Errorf("signal still has %d subscribers, want 0", len(sig.subs))
	}
	sig.Set(1)
	if len(tr.dirty) != 0 {
		t.Errorf("%d dirty nodes after Set on the dead node's signal, want 0", len(tr.dirty))
	}
}

func TestDependenciesFollowTheLastBuild(t *testing.T) {
	flag := NewSignal(true)
	a := NewSignal(0)
	tr := newTestTree(Build(func(*Ctx) Widget {
		if flag.Get() {
			a.Get()
		}
		return nil
	}))

	flag.Set(false)
	tr.rebuildDirty()
	a.Set(1)

	if len(tr.dirty) != 0 {
		t.Errorf("%d dirty nodes, want 0: the last build did not read a", len(tr.dirty))
	}
}

func TestParentAndChildRebuildOnce(t *testing.T) {
	s := NewSignal(0)
	var parentRuns, childRuns int
	tr := newTestTree(Build(func(*Ctx) Widget {
		parentRuns++
		s.Get()
		return Build(func(*Ctx) Widget {
			childRuns++
			s.Get()
			return nil
		})
	}))

	s.Set(1)
	tr.rebuildDirty()

	if parentRuns != 2 || childRuns != 2 {
		t.Errorf("runs: parent=%d child=%d, want 2 and 2", parentRuns, childRuns)
	}
}

func TestSetDuringBuildWaitsForNextRebuild(t *testing.T) {
	s := NewSignal(0)
	runs := 0
	tr := newTestTree(Build(func(*Ctx) Widget {
		runs++
		if v := s.Get(); v < 3 {
			s.Set(v + 1)
		}
		return nil
	}))
	if len(tr.dirty) != 1 {
		t.Fatalf("after mount: %d dirty nodes, want 1", len(tr.dirty))
	}

	tr.rebuildDirty()

	if runs != 2 || len(tr.dirty) != 1 {
		t.Errorf("after one rebuild: runs=%d dirty=%d, want 2 and 1", runs, len(tr.dirty))
	}
}
