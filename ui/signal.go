package ui

// Signal holds a value that the UI depends on. A composite build that calls
// Get is subscribed automatically; Set marks every subscriber dirty, and
// the next frame rebuilds just those nodes.
//
// Signal is a generic type: T is a type parameter, like a template
// parameter in C++. Unlike a C++ template, Go compiles generic code once
// per "shape" of types, and T can only be used in the ways its constraint
// allows; "any" allows everything except arithmetic and comparison.
//
// Get, Set and Update must be called on the UI goroutine. Hand results from
// other goroutines over with App.Post or Async.
type Signal[T any] struct {
	value T
	// subs is a set of nodes: a map with empty struct values. struct{}
	// takes no memory, so map[K]struct{} is Go's std::unordered_set<K>.
	subs map[*node]struct{}
}

// dependency is what a node remembers about each signal it read, so it can
// unsubscribe later. Signal[int] and Signal[string] are different types,
// so the node keeps them behind this interface.
type dependency interface {
	unsubscribe(n *node)
}

// NewSignal makes a signal holding v. The type parameter is inferred:
// NewSignal(0) is a *Signal[int].
func NewSignal[T any](v T) *Signal[T] {
	return &Signal[T]{value: v}
}

// Get returns the value. Inside a build it also subscribes the building node.
func (s *Signal[T]) Get() T {
	if n := currentBuild; n != nil {
		if _, ok := s.subs[n]; !ok {
			if s.subs == nil {
				s.subs = make(map[*node]struct{})
			}
			s.subs[n] = struct{}{}
			n.deps = append(n.deps, s)
		}
	}
	return s.value
}

// Set stores v and marks every subscriber dirty. It notifies even when v
// equals the old value, which is why T does not have to be comparable.
func (s *Signal[T]) Set(v T) {
	s.value = v
	for n := range s.subs {
		n.markDirty()
	}
}

// Update sets the value to fn(old value). It does not subscribe, even
// inside a build.
func (s *Signal[T]) Update(fn func(T) T) {
	s.Set(fn(s.value))
}

func (s *Signal[T]) unsubscribe(n *node) {
	delete(s.subs, n)
}
