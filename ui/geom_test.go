package ui

import "testing"

// Table-driven tests are the usual Go shape: a slice of anonymous structs,
// one per case, and a single loop that checks them all.

func TestConstrain(t *testing.T) {
	tests := []struct {
		name string
		c    Constraints
		in   Size
		want Size
	}{
		{"tight forces the size", Tight(Size{W: 10, H: 20}), Size{W: 5, H: 50}, Size{W: 10, H: 20}},
		{"unbounded keeps the size", Constraints{0, Inf, 0, Inf}, Size{W: 30, H: 40}, Size{W: 30, H: 40}},
		{"each axis is clamped on its own", Constraints{10, 100, 10, 100}, Size{W: 5, H: 500}, Size{W: 10, H: 100}},
	}
	for _, tt := range tests {
		if got := tt.c.Constrain(tt.in); got != tt.want {
			t.Errorf("%s: %+v.Constrain(%+v) = %+v, want %+v", tt.name, tt.c, tt.in, got, tt.want)
		}
	}
}

func TestLoosen(t *testing.T) {
	got := Constraints{10, 100, 20, 200}.Loosen()
	want := Constraints{0, 100, 0, 200}
	if got != want {
		t.Errorf("Loosen() = %+v, want %+v", got, want)
	}
}

func TestDeflate(t *testing.T) {
	tests := []struct {
		name string
		c    Constraints
		in   Insets
		want Constraints
	}{
		{"mins clamp at zero", Constraints{10, 100, 10, 100}, All(8), Constraints{0, 84, 0, 84}},
		{"left+right from width, top+bottom from height", Constraints{0, 100, 0, 50},
			Insets{Top: 1, Right: 2, Bottom: 3, Left: 4}, Constraints{0, 94, 0, 46}},
		{"insets larger than max give zero, not negative", Constraints{0, 10, 0, 10}, All(8), Constraints{0, 0, 0, 0}},
		{"Inf stays Inf", Constraints{0, Inf, 0, Inf}, All(8), Constraints{0, Inf, 0, Inf}},
	}
	for _, tt := range tests {
		if got := tt.c.Deflate(tt.in); got != tt.want {
			t.Errorf("%s: %+v.Deflate(%+v) = %+v, want %+v", tt.name, tt.c, tt.in, got, tt.want)
		}
	}
}

func TestSymmetric(t *testing.T) {
	got := Symmetric(4, 8)
	want := Insets{Top: 4, Right: 8, Bottom: 4, Left: 8}
	if got != want {
		t.Errorf("Symmetric(4, 8) = %+v, want %+v", got, want)
	}
}
