package domain

import "math"

// Layout proportions of the indicator, as shares of its height.
const (
	// FontToHeight is the day name's character height.
	FontToHeight = 0.55
	// PadToHeight is the space left and right of the widest day name.
	PadToHeight = 0.35
)

// Point is a position in virtual-screen pixels.
type Point struct{ X, Y int }

// Size is a width and height in pixels.
type Size struct{ W, H int }

// Rect is a rectangle in virtual-screen pixels; Right and Bottom are
// exclusive, as Windows reports them.
type Rect struct{ Left, Top, Right, Bottom int }

// Contains reports whether p lies inside r.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.Left && p.X < r.Right && p.Y >= r.Top && p.Y < r.Bottom
}

// IndicatorSize answers the indicator's size: as high as the taskbar and wide
// enough for the widest day name plus padding either side, so the width never
// changes from day to day (FR-014).
func IndicatorSize(taskbarHeight, widestText int) Size {
	pad := int(math.Round(float64(taskbarHeight) * PadToHeight))
	return Size{W: widestText + 2*pad, H: taskbarHeight}
}

// FontHeight answers the day name's character height for an indicator of the
// given height.
func FontHeight(indicatorHeight int) int {
	return int(math.Round(float64(indicatorHeight) * FontToHeight))
}

// ClampToWork answers the position nearest to pos at which an indicator of
// size lies wholly inside work, so it can never overlap the taskbar (FR-018).
func ClampToWork(pos Point, size Size, work Rect) Point {
	return Point{
		X: min(max(pos.X, work.Left), work.Right-size.W),
		Y: min(max(pos.Y, work.Top), work.Bottom-size.H),
	}
}

// DefaultPosition answers the bottom-right corner of the primary monitor's
// work area (FR-021).
func DefaultPosition(size Size, primaryWork Rect) Point {
	return Point{X: primaryWork.Right - size.W, Y: primaryWork.Bottom - size.H}
}

// Monitor is one attached display: its whole rectangle and its work area
// (the rectangle less the taskbar).
type Monitor struct{ Bounds, Work Rect }

// Place answers where the indicator goes at startup or after a display change
// (FR-021, FR-022). It takes the default position when there is no saved
// position or when the saved centre lies on no attached monitor (its monitor
// has gone); otherwise it keeps the saved position, clamped into the work area
// of the monitor holding its centre. Bounds rather than work areas decide
// "lost", so a strip whose centre sits over a taskbar is nudged, not moved
// to another monitor (Amendment 1).
func Place(saved Point, hasSaved bool, size Size, monitors []Monitor, primaryWork Rect) Point {
	fallback := DefaultPosition(size, primaryWork)
	if !hasSaved {
		return fallback
	}
	centre := Point{X: saved.X + size.W/2, Y: saved.Y + size.H/2}
	for _, m := range monitors {
		if m.Bounds.Contains(centre) {
			return ClampToWork(saved, size, m.Work)
		}
	}
	return fallback
}

// PassedDragThreshold reports whether a pointer that has moved by (dx, dy)
// since the button went down has left the system drag rectangle, whose half
// extents are (thresholdX, thresholdY). A press and release that never passes
// it is a click, which does nothing (FR-017, FR-019).
func PassedDragThreshold(dx, dy, thresholdX, thresholdY int) bool {
	return abs(dx) > thresholdX || abs(dy) > thresholdY
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
