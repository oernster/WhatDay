package domain_test

import (
	"testing"

	"github.com/oernster/WhatDay/internal/domain"
)

// Measured on the reference machine (REQUIREMENTS.md Appendix A, M-5).
var (
	measuredWork    = domain.Rect{Left: 0, Top: 0, Right: 3440, Bottom: 1392}
	measuredBounds  = domain.Rect{Left: 0, Top: 0, Right: 3440, Bottom: 1440}
	measuredSize    = domain.Size{W: 175, H: 48}
	measuredMonitor = domain.Monitor{Bounds: measuredBounds, Work: measuredWork}
)

func TestSizeFitsWidestDay(t *testing.T) {
	t.Parallel()
	// The probe measured "Wednesday" at 143 px on a 48 px taskbar (its 175 px
	// window less 16 px truncated padding a side). The domain rounds the
	// padding instead: 48 * 0.35 = 16.8, so 17 a side and 143 + 34 = 177.
	const taskbar, widest = 48, 143
	want := domain.Size{W: 177, H: 48}
	if got := domain.IndicatorSize(taskbar, widest); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestTaskbarHeightFromWorkArea(t *testing.T) {
	t.Parallel()
	// Measured: bounds bottom 1440, work bottom 1392 (M-5), a 48 px taskbar.
	if got := domain.TaskbarHeight(measuredMonitor); got != 48 {
		t.Fatalf("got %d, want 48", got)
	}
}

func TestIndicatorHeightUsesOwnTaskbar(t *testing.T) {
	t.Parallel()
	const standardDPI, highDPI = 96, 240
	// Illustrative 250% monitor with its own 120 px taskbar.
	high := domain.Monitor{
		Bounds: domain.Rect{Left: -3840, Top: 0, Right: 0, Bottom: 2400},
		Work:   domain.Rect{Left: -3840, Top: 0, Right: 0, Bottom: 2280},
	}
	if got := domain.IndicatorHeight(high, highDPI, measuredMonitor, standardDPI); got != 120 {
		t.Fatalf("got %d, want 120", got)
	}
}

func TestIndicatorHeightWithoutTaskbarScalesPrimary(t *testing.T) {
	t.Parallel()
	const standardDPI, highDPI = 96, 240
	bare := domain.Monitor{
		Bounds: domain.Rect{Left: -3840, Top: 0, Right: 0, Bottom: 2400},
		Work:   domain.Rect{Left: -3840, Top: 0, Right: 0, Bottom: 2400},
	}
	// 48 px at 96 DPI is 120 px at 240 DPI.
	if got := domain.IndicatorHeight(bare, highDPI, measuredMonitor, standardDPI); got != 120 {
		t.Fatalf("got %d, want 120", got)
	}
}

func TestFontHeight(t *testing.T) {
	t.Parallel()
	if got := domain.FontHeight(48); got != 26 {
		t.Fatalf("got %d, want 26", got)
	}
}

func TestClampToWorkArea(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		pos  domain.Point
		want domain.Point
	}{
		{"onto the taskbar", domain.Point{X: 3400, Y: 1420}, domain.Point{X: 3265, Y: 1344}},
		{"off the top left", domain.Point{X: -50, Y: -50}, domain.Point{X: 0, Y: 0}},
		{"already inside", domain.Point{X: 100, Y: 200}, domain.Point{X: 100, Y: 200}},
	}
	for _, c := range cases {
		if got := domain.ClampToWork(c.pos, measuredSize, measuredWork); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestDefaultPosition(t *testing.T) {
	t.Parallel()
	want := domain.Point{X: 3265, Y: 1344}
	if got := domain.DefaultPosition(measuredSize, measuredWork); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPlaceWithoutSavedPosition(t *testing.T) {
	t.Parallel()
	got := domain.Place(domain.Point{}, false, measuredSize, []domain.Monitor{measuredMonitor}, measuredWork)
	if want := (domain.Point{X: 3265, Y: 1344}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPlaceKeepsSavedPositionOnSecondMonitor(t *testing.T) {
	t.Parallel()
	// The saved point is where the probe was dragged (M-6); this monitor's
	// rectangle is illustrative, not measured.
	second := domain.Monitor{Bounds: domain.Rect{Left: 3440, Top: 1440, Right: 7280, Bottom: 3840}, Work: domain.Rect{Left: 3440, Top: 1440, Right: 7280, Bottom: 3792}}
	saved := domain.Point{X: 3442, Y: 3672}
	got := domain.Place(saved, true, measuredSize, []domain.Monitor{measuredMonitor, second}, measuredWork)
	if got != saved {
		t.Fatalf("got %+v, want %+v", got, saved)
	}
}

func TestPlaceClampsSavedPositionOverlappingTaskbar(t *testing.T) {
	t.Parallel()
	saved := domain.Point{X: 3000, Y: 1370}
	got := domain.Place(saved, true, measuredSize, []domain.Monitor{measuredMonitor}, measuredWork)
	if want := (domain.Point{X: 3000, Y: 1344}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLostMonitorFallsBack(t *testing.T) {
	t.Parallel()
	saved := domain.Point{X: 5000, Y: 2000} // on a monitor that is no longer attached
	got := domain.Place(saved, true, measuredSize, []domain.Monitor{measuredMonitor}, measuredWork)
	if want := (domain.Point{X: 3265, Y: 1344}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestThresholdSeparatesClickFromDrag(t *testing.T) {
	t.Parallel()
	const tx, ty = 4, 4
	cases := []struct {
		dx, dy int
		want   bool
	}{
		{0, 0, false},
		{4, -4, false},
		{5, 0, true},
		{0, -5, true},
		{-5, 0, true},
	}
	for _, c := range cases {
		if got := domain.PassedDragThreshold(c.dx, c.dy, tx, ty); got != c.want {
			t.Errorf("(%d,%d): got %v, want %v", c.dx, c.dy, got, c.want)
		}
	}
}
