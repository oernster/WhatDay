package domain_test

import (
	"math"
	"testing"

	"github.com/oernster/WhatDay/internal/domain"
)

// Colour science used only to test the palette: WCAG 2 contrast and the
// CIEDE2000 colour difference. Its one home is here.

type lab struct{ l, a, b float64 }

func linear(channel uint8) float64 {
	c := float64(channel) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func luminance(c domain.Colour) float64 {
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

// contrastRatio is the WCAG 2 contrast ratio between two colours.
func contrastRatio(x, y domain.Colour) float64 {
	lx, ly := luminance(x), luminance(y)
	return (math.Max(lx, ly) + 0.05) / (math.Min(lx, ly) + 0.05)
}

// toLab converts sRGB to CIELAB under D65.
func toLab(c domain.Colour) lab {
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	x := (0.4124564*r + 0.3575761*g + 0.1804375*b) / 0.95047
	y := 0.2126729*r + 0.7151522*g + 0.0721750*b
	z := (0.0193339*r + 0.1191920*g + 0.9503041*b) / 1.08883
	f := func(t float64) float64 {
		if t > 216.0/24389 {
			return math.Cbrt(t)
		}
		return (24389.0/27*t + 16) / 116
	}
	return lab{116*f(y) - 16, 500 * (f(x) - f(y)), 200 * (f(y) - f(z))}
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }

func hueDeg(b, a float64) float64 {
	h := math.Atan2(b, a) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return h
}

// deltaE2000 is the CIEDE2000 colour difference (Sharma, Wu and Dalal 2005).
func deltaE2000(p, q lab) float64 {
	pow7 := func(v float64) float64 { return math.Pow(v, 7) }
	const twentyFive7 = 6103515625 // 25^7
	c1, c2 := math.Hypot(p.a, p.b), math.Hypot(q.a, q.b)
	cBar := (c1 + c2) / 2
	g := 0.5 * (1 - math.Sqrt(pow7(cBar)/(pow7(cBar)+twentyFive7)))
	a1, a2 := (1+g)*p.a, (1+g)*q.a
	c1p, c2p := math.Hypot(a1, p.b), math.Hypot(a2, q.b)
	h1, h2 := hueDeg(p.b, a1), hueDeg(q.b, a2)

	dl, dc := q.l-p.l, c2p-c1p
	var dh float64
	switch {
	case c1p*c2p == 0:
	case math.Abs(h2-h1) <= 180:
		dh = h2 - h1
	case h2-h1 > 180:
		dh = h2 - h1 - 360
	default:
		dh = h2 - h1 + 360
	}
	dH := 2 * math.Sqrt(c1p*c2p) * math.Sin(rad(dh/2))

	lBar, cBarP := (p.l+q.l)/2, (c1p+c2p)/2
	var hBar float64
	switch {
	case c1p*c2p == 0:
		hBar = h1 + h2
	case math.Abs(h1-h2) <= 180:
		hBar = (h1 + h2) / 2
	case h1+h2 < 360:
		hBar = (h1 + h2 + 360) / 2
	default:
		hBar = (h1 + h2 - 360) / 2
	}
	t := 1 - 0.17*math.Cos(rad(hBar-30)) + 0.24*math.Cos(rad(2*hBar)) +
		0.32*math.Cos(rad(3*hBar+6)) - 0.20*math.Cos(rad(4*hBar-63))
	dTheta := 30 * math.Exp(-math.Pow((hBar-275)/25, 2))
	rc := 2 * math.Sqrt(pow7(cBarP)/(pow7(cBarP)+twentyFive7))
	sl := 1 + 0.015*math.Pow(lBar-50, 2)/math.Sqrt(20+math.Pow(lBar-50, 2))
	sc := 1 + 0.045*cBarP
	sh := 1 + 0.015*cBarP*t
	rt := -math.Sin(rad(2*dTheta)) * rc
	return math.Sqrt(math.Pow(dl/sl, 2) + math.Pow(dc/sc, 2) + math.Pow(dH/sh, 2) + rt*(dc/sc)*(dH/sh))
}

// TestDeltaE2000MatchesReference proves the formula against published pairs
// before any palette test relies on it.
func TestDeltaE2000MatchesReference(t *testing.T) {
	t.Parallel()
	const tolerance = 1e-4
	cases := []struct {
		p, q lab
		want float64
	}{
		{lab{50, 2.6772, -79.7751}, lab{50, 0, -82.7485}, 2.0425},
		{lab{50, 2.5, 0}, lab{73, 25, -18}, 27.1492},
		{lab{60.2574, -34.0099, 36.2677}, lab{60.4626, -34.1751, 39.4387}, 1.2644},
		{lab{2.0776, 0.0795, -1.1350}, lab{0.9033, -0.0636, -0.5514}, 0.9082},
	}
	for _, c := range cases {
		if got := deltaE2000(c.p, c.q); math.Abs(got-c.want) > tolerance {
			t.Errorf("%v vs %v: got %.4f, want %.4f", c.p, c.q, got, c.want)
		}
	}
}

func TestContrastMatchesReference(t *testing.T) {
	t.Parallel()
	black := domain.Colour{}
	white := domain.Colour{R: 0xFF, G: 0xFF, B: 0xFF}
	const wcagMaximum = 21
	if got := contrastRatio(black, white); math.Abs(got-wcagMaximum) > 1e-9 {
		t.Fatalf("black on white: got %v, want %v", got, wcagMaximum)
	}
}
