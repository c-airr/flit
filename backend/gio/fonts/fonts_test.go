package fonts

import (
	"testing"

	"gioui.org/font"
)

func TestCollection(t *testing.T) {
	faces := Collection()
	if len(faces) != 2 {
		t.Fatalf("got %d faces, want 2", len(faces))
	}
	wantWeights := []font.Weight{font.Normal, font.SemiBold}
	for i, f := range faces {
		if f.Face == nil {
			t.Errorf("face %d: Face is nil", i)
		}
		if f.Font.Typeface != Typeface || f.Font.Weight != wantWeights[i] || f.Font.Style != font.Regular {
			t.Errorf("face %d: %+v, want typeface %q, weight %v, regular style", i, f.Font, Typeface, wantWeights[i])
		}
	}
}
