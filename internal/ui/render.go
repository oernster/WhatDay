package ui

import (
	"time"
	"unsafe"

	"github.com/oernster/WhatDay/internal/domain"
)

const (
	opaque    = 255
	whiteText = 0x00FFFFFF
	bitsPerPx = 32
)

// frame is a premultiplied ARGB bitmap selected into its own memory DC.
type frame struct{ dc, dib, old uintptr }

func (f frame) release() {
	_, _, _ = pSelectObject.Call(f.dc, f.old)
	_, _, _ = pDeleteObject.Call(f.dib)
	_, _, _ = pDeleteDC.Call(f.dc)
}

// newFont makes the day name's font at a character height in pixels.
func newFont(height int) uintptr {
	f, _, _ := pCreateFont.Call(uintptr(-height), 0, 0, 0, fwSemibold, 0, 0, 0, 0, 0, 0, antialiased, 0, uintptr(unsafe.Pointer(wide(fontFace))))
	return f
}

// widestDayWidth measures every day name at the font for an indicator of
// height, so the width never changes from day to day (FR-014).
func widestDayWidth(height int) int {
	dc, _, _ := pGetDC.Call(0)
	defer func() { _, _, _ = pReleaseDC.Call(0, dc) }()
	font := newFont(domain.FontHeight(height))
	old, _, _ := pSelectObject.Call(dc, font)
	defer func() {
		_, _, _ = pSelectObject.Call(dc, old)
		_, _, _ = pDeleteObject.Call(font)
	}()
	widest := 0
	for d := time.Sunday; d <= time.Saturday; d++ {
		name := d.String()
		var sz size
		_, _, _ = pGetTextExtent.Call(dc, uintptr(unsafe.Pointer(wide(name))), uintptr(len(name)), uintptr(unsafe.Pointer(&sz)))
		widest = max(widest, int(sz.cx))
	}
	return widest
}

// render draws text centred in colour over a fully transparent background,
// so the Acrylic backdrop shows through everywhere but the letters. The
// letters' alpha is their antialiased coverage, drawn white on black first
// (probe round 3, Appendix A M-4 and M-5).
func render(text string, colour domain.Colour, s domain.Size) frame {
	w, h := s.W, s.H
	header := bitmapInfoHeader{width: int32(w), height: -int32(h), planes: 1, bitCount: bitsPerPx}
	header.size = uint32(unsafe.Sizeof(header))
	var bits unsafe.Pointer
	dc, _, _ := pCreateCompatDC.Call(0)
	dib, _, _ := pCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&header)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	old, _, _ := pSelectObject.Call(dc, dib)

	font := newFont(domain.FontHeight(h))
	previous, _, _ := pSelectObject.Call(dc, font)
	_, _, _ = pSetBkMode.Call(dc, bkTransparent)
	_, _, _ = pSetTextColor.Call(dc, whiteText)
	area := rect{0, 0, int32(w), int32(h)}
	_, _, _ = pDrawText.Call(dc, uintptr(unsafe.Pointer(wide(text))), ^uintptr(0), uintptr(unsafe.Pointer(&area)), dtCentre)
	_, _, _ = pSelectObject.Call(dc, previous)
	_, _, _ = pDeleteObject.Call(font)

	if bits != nil {
		pixels := unsafe.Slice((*uint32)(bits), w*h)
		for i, p := range pixels {
			pixels[i] = premultiplied(p&opaque, colour)
		}
	}
	return frame{dc: dc, dib: dib, old: old}
}

// premultiplied answers one ARGB pixel of colour at coverage 0 to 255.
func premultiplied(coverage uint32, c domain.Colour) uint32 {
	r := uint32(c.R) * coverage / opaque
	g := uint32(c.G) * coverage / opaque
	b := uint32(c.B) * coverage / opaque
	return coverage<<24 | r<<16 | g<<8 | b
}
