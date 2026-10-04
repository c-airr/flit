package draw

// TextMeasurer is the part of the boundary that goes from the renderer
// back to flit: layout needs to know how big a piece of text is, and only
// the renderer knows, because it owns the fonts and the text shaper.
//
// A renderer must measure and draw with the same shaper, so that a Text
// command drawn with the same text, style and width fits the measured size.
//
// In C++ terms this is an abstract class with one pure virtual method.
// Any Go type with this method satisfies the interface automatically.
type TextMeasurer interface {
	// Measure returns the size of text laid out with style.
	// maxWidth <= 0 means unlimited: one line unless the text contains '\n'.
	Measure(text string, style FontStyle, maxWidth float32) Size
}
