package domain_test

import (
	"testing"

	"github.com/oernster/WhatDay/internal/domain"
)

// Thresholds from REQUIREMENTS.md NFR-COL-001 and NFR-COL-002.
const (
	minContrast = 3.0 // WCAG 2 large text
	minDeltaE   = 20.0
)

// Backgrounds every shade must read against. The first is measured (M-4);
// the second is a provisional lighter worst case standing in for Acrylic over
// a white window until Q-5 measures it.
var backgrounds = []struct {
	name   string
	colour domain.Colour
}{
	{"measured taskbar mean #1C222F", domain.Colour{R: 0x1C, G: 0x22, B: 0x2F}},
	{"provisional worst case #3A3A3A", domain.Colour{R: 0x3A, G: 0x3A, B: 0x3A}},
}

func TestPaletteNamesAndOrder(t *testing.T) {
	t.Parallel()
	want := []string{"Red", "Amber", "Yellow", "Green", "Blue", "Purple", "Neutral"}
	got := domain.Palette()
	if len(got) != len(want) {
		t.Fatalf("got %d colours, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("position %d: got %q, want %q", i, got[i].Name, name)
		}
	}
}

func TestPaletteIsACopy(t *testing.T) {
	t.Parallel()
	first := domain.Palette()
	first[0].R = 0
	if domain.Palette()[0].R == 0 {
		t.Fatal("changing a returned palette changed the palette")
	}
}

func TestPaletteContrast(t *testing.T) {
	t.Parallel()
	for _, c := range domain.Palette() {
		for _, bg := range backgrounds {
			if got := contrastRatio(c, bg.colour); got < minContrast {
				t.Errorf("%s on %s: %.2f, want at least %.1f", c.Name, bg.name, got, minContrast)
			}
		}
	}
}

func TestPaletteDistinct(t *testing.T) {
	t.Parallel()
	colours := domain.Palette()
	for i := range colours {
		for j := i + 1; j < len(colours); j++ {
			if got := deltaE2000(toLab(colours[i]), toLab(colours[j])); got < minDeltaE {
				t.Errorf("%s / %s: dE2000 %.1f, want at least %.1f", colours[i].Name, colours[j].Name, got, minDeltaE)
			}
		}
	}
}

func TestColourNamed(t *testing.T) {
	t.Parallel()
	got, ok := domain.ColourNamed("Blue")
	if !ok || got.Name != "Blue" {
		t.Fatalf("got %+v, %v; want Blue, true", got, ok)
	}
}

func TestUnknownColourFallsBackToDefault(t *testing.T) {
	t.Parallel()
	got, ok := domain.ColourNamed("Magenta")
	if ok || got.Name != domain.DefaultColourName {
		t.Fatalf("got %+v, %v; want %s, false", got, ok, domain.DefaultColourName)
	}
}
