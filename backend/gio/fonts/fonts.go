// Package fonts embeds the Inter typeface (regular and semibold) for the
// Gio backend, so text looks the same on every machine without relying on
// system fonts. Inter is under the SIL Open Font License; see OFL.txt.
package fonts

import (
	// The blank import "_" loads a package only for its side effects.
	// The embed package must be imported for //go:embed to work.
	_ "embed"
	"fmt"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

// The //go:embed directives make the compiler put each file into the
// binary and set the variable to its bytes, so the program needs no font
// files next to it at run time. The comment must sit right above the var.

//go:embed Inter-Regular.ttf
var interRegular []byte

//go:embed Inter-SemiBold.ttf
var interSemiBold []byte

// Typeface is the name that selects Inter in a font.Font.
const Typeface font.Typeface = "Inter"

// Collection parses the embedded fonts, for text.WithCollection.
// Each call parses them again, so create the shaper once and keep it.
func Collection() []font.FontFace {
	return []font.FontFace{
		mustParse(interRegular, "Inter-Regular.ttf"),
		mustParse(interSemiBold, "Inter-SemiBold.ttf"),
	}
}

// mustParse panics on error: the fonts are compiled into the binary, so a
// parse error means a broken build, not something a caller could handle.
func mustParse(src []byte, name string) font.FontFace {
	face, err := opentype.Parse(src)
	if err != nil {
		panic(fmt.Sprintf("fonts: embedded %s: %v", name, err))
	}
	return font.FontFace{Font: face.Font(), Face: face}
}
