package domain

// Colour is one named choice for the day name (FR-040).
type Colour struct {
	Name    string
	R, G, B uint8
}

// DefaultColourName is the colour used when none has been saved (FR-041).
const DefaultColourName = "Red"

// palette holds every choice in menu order, one shade each for the dark
// Acrylic background (Amendment 2). Contrast and distinctness are enforced by
// palette_test.go (NFR-COL-001, NFR-COL-002).
var palette = [...]Colour{
	{Name: "Red", R: 0xFF, G: 0x66, B: 0x66},
	{Name: "Amber", R: 0xFF, G: 0xB0, B: 0x20},
	{Name: "Yellow", R: 0xF5, G: 0xF0, B: 0x4A},
	{Name: "Green", R: 0x4C, G: 0xD9, B: 0x64},
	{Name: "Blue", R: 0x5A, G: 0xA8, B: 0xFF},
	{Name: "Purple", R: 0xC5, G: 0x8C, B: 0xFF},
	{Name: "Neutral", R: 0xFF, G: 0xFF, B: 0xFF},
}

// Palette answers every colour in menu order. The slice is a copy, so no
// caller can alter the palette.
func Palette() []Colour {
	out := make([]Colour, len(palette))
	copy(out, palette[:])
	return out
}

// ColourNamed answers the colour called name; the default colour and false
// when there is none, so a stale or corrupt saved name still paints.
func ColourNamed(name string) (Colour, bool) {
	if c, ok := find(name); ok {
		return c, true
	}
	fallback, _ := find(DefaultColourName)
	return fallback, false
}

func find(name string) (Colour, bool) {
	for _, c := range palette {
		if c.Name == name {
			return c, true
		}
	}
	return Colour{}, false
}
