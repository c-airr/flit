package ui

import "github.com/c-airr/flit/draw"

// Theme holds the colors and sizes that widgets fall back to when an option
// is not set, so an unstyled tree already looks finished.
type Theme struct {
	Background, Surface, Primary, PrimaryHover, PrimaryPressed,
	OnPrimary, Text, TextMuted, Outline draw.RGBA

	Radius    float32 // corner radius
	Spacing   float32 // grid unit; gaps and paddings are multiples of it
	FontSize  float32 // body text
	TitleSize float32 // headings
}

// DefaultTheme is a light theme with an indigo accent.
func DefaultTheme() Theme {
	return Theme{
		Background:     draw.Hex(0xF7F7F8),
		Surface:        draw.Hex(0xFFFFFF),
		Primary:        draw.Hex(0x4F46E5),
		PrimaryHover:   draw.Hex(0x4338CA),
		PrimaryPressed: draw.Hex(0x3730A3),
		OnPrimary:      draw.Hex(0xFFFFFF),
		Text:           draw.Hex(0x18181B),
		TextMuted:      draw.Hex(0x71717A),
		Outline:        draw.Hex(0xE4E4E7),
		Radius:         8,
		Spacing:        4,
		FontSize:       14,
		TitleSize:      20,
	}
}
