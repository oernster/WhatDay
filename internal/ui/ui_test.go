package ui

import (
	"testing"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
)

func TestMenuEntriesFollowTheModel(t *testing.T) {
	t.Parallel()
	items := []application.MenuItem{
		{Kind: application.MenuAbout, Label: "About WhatDay"},
		{Kind: application.MenuSeparator},
		{Kind: application.MenuColour, Label: "Red"},
		{Kind: application.MenuColour, Label: "Amber", Checked: true},
		{Kind: application.MenuSeparator},
		{Kind: application.MenuQuit, Label: "Quit WhatDay"},
	}
	want := []menuEntry{
		{id: cmdAbout, flags: mfString, label: "About WhatDay"},
		{flags: mfSeparator},
		{id: cmdColourBase, flags: mfString, label: "Red"},
		{id: cmdColourBase + 1, flags: mfString | mfChecked, label: "Amber"},
		{flags: mfSeparator},
		{id: cmdQuit, flags: mfString, label: "Quit WhatDay"},
	}
	got := menuEntries(items)
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestChosenResolvesCommands(t *testing.T) {
	t.Parallel()
	entries := menuEntries([]application.MenuItem{
		{Kind: application.MenuAbout, Label: "About WhatDay"},
		{Kind: application.MenuSeparator},
		{Kind: application.MenuColour, Label: "Blue"},
	})
	if e, ok := chosen(entries, cmdColourBase); !ok || e.label != "Blue" {
		t.Errorf("colour command: got %+v, %v", e, ok)
	}
	if e, ok := chosen(entries, cmdAbout); !ok || e.id != cmdAbout {
		t.Errorf("about command: got %+v, %v", e, ok)
	}
	quit := menuEntries([]application.MenuItem{{Kind: application.MenuQuit, Label: "Quit WhatDay"}})
	if e, ok := chosen(quit, cmdQuit); !ok || e.label != "Quit WhatDay" {
		t.Errorf("quit command: got %+v, %v", e, ok)
	}
	// A dismissed menu answers 0, which must not match the separator's id.
	if _, ok := chosen(entries, 0); ok {
		t.Error("a dismissed menu chose something")
	}
	if _, ok := chosen(entries, cmdColourBase+9); ok {
		t.Error("an unknown command chose something")
	}
}

func TestPremultiplied(t *testing.T) {
	t.Parallel()
	red := domain.Colour{Name: "Red", R: 0xFF, G: 0x66, B: 0x66}
	cases := []struct {
		coverage, want uint32
	}{
		{0, 0},
		{opaque, 0xFFFF6666},
		{0x80, 0x80803333},
	}
	for _, c := range cases {
		if got := premultiplied(c.coverage, red); got != c.want {
			t.Errorf("coverage %d: got %08X, want %08X", c.coverage, got, c.want)
		}
	}
}

// TestRenderPaintsTheWordOnTransparency draws for real with GDI and reads the
// pixels back. The corners must be fully transparent; somewhere the letters
// must be the chosen colour at full coverage.
func TestRenderPaintsTheWordOnTransparency(t *testing.T) {
	const height = 48
	s := domain.IndicatorSize(height, widestDayWidth(height))
	blue := domain.Colour{Name: "Blue", R: 0x5A, G: 0xA8, B: 0xFF}
	f := render("Wednesday", blue, s)
	defer f.release()

	bits := dibBits(t, f, s)
	for _, corner := range []int{0, s.W - 1, (s.H - 1) * s.W, s.H*s.W - 1} {
		if bits[corner] != 0 {
			t.Errorf("corner pixel %d is %08X, want transparent", corner, bits[corner])
		}
	}
	solid := premultiplied(opaque, blue)
	found := 0
	for _, p := range bits {
		if p == solid {
			found++
		}
	}
	if found == 0 {
		t.Fatal("no pixel of the word is drawn in full colour")
	}
}

func TestWidestDayFitsWednesday(t *testing.T) {
	t.Parallel()
	const height = 48
	// Measured by the probe: "Wednesday" was 143 px at this height.
	const measured = 143
	if got := widestDayWidth(height); got != measured {
		t.Fatalf("got %d, want the probe's measured %d", got, measured)
	}
}

func TestOwnIconAlwaysAnswersAnIcon(t *testing.T) {
	t.Parallel()
	// A test binary carries no icon resource, so this is the fallback path.
	if ownIcon() == 0 {
		t.Fatal("no icon at all")
	}
}

func TestMonitorsFindsAPrimary(t *testing.T) {
	t.Parallel()
	screens, primary := monitors()
	if len(screens) == 0 || primary.Bounds.Right <= primary.Bounds.Left {
		t.Fatalf("got %d screens, primary %+v", len(screens), primary)
	}
	if primary.dpi <= 0 {
		t.Fatalf("primary DPI %d", primary.dpi)
	}
	if got := screenAt(screens, primary, centre(domain.Point{X: primary.Work.Left, Y: primary.Work.Top}, domain.Size{W: 2, H: 2})); got.handle != primary.handle {
		t.Errorf("a point on the primary monitor found %+v", got)
	}
}

// dibBits reads the pixels of a rendered frame through GetDIBits.
func dibBits(t *testing.T, f frame, s domain.Size) []uint32 {
	t.Helper()
	header := bitmapInfoHeader{width: int32(s.W), height: -int32(s.H), planes: 1, bitCount: bitsPerPx}
	header.size = uint32(unsafe.Sizeof(header))
	out := make([]uint32, s.W*s.H)
	getDIBits := gdi32.NewProc("GetDIBits")
	// The bitmap must not be selected into a DC while GetDIBits reads it.
	_, _, _ = pSelectObject.Call(f.dc, f.old)
	lines, _, _ := getDIBits.Call(f.dc, f.dib, 0, uintptr(s.H), uintptr(unsafe.Pointer(&out[0])), uintptr(unsafe.Pointer(&header)), 0)
	_, _, _ = pSelectObject.Call(f.dc, f.dib)
	if int(lines) != s.H {
		t.Fatalf("GetDIBits read %d lines, want %d", lines, s.H)
	}
	return out
}
