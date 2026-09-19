package ui

import (
	"unsafe"

	"github.com/oernster/WhatDay/internal/domain"
	"golang.org/x/sys/windows"
)

// standardDPI is Windows' 100% scale, used when a monitor's DPI cannot be read.
const standardDPI = 96

// screen is one attached monitor as the window needs it.
type screen struct {
	domain.Monitor
	handle  uintptr
	dpi     int
	primary bool
}

// enumerated collects monitors during EnumDisplayMonitors; the callback is
// made once, since Windows allows a process only so many.
var (
	enumerated []screen
	enumerate  = windows.NewCallback(func(handle, _, _, _ uintptr) uintptr {
		enumerated = append(enumerated, describe(handle))
		return 1 // carry on
	})
)

// monitors answers every attached monitor and the primary one. With none
// reported (a transient state during a display change) it answers the
// nearest monitor to the origin, so the window always has somewhere to be.
func monitors() ([]screen, screen) {
	enumerated = nil
	_, _, _ = pEnumMonitors.Call(0, 0, enumerate, 0)
	found := enumerated
	if len(found) == 0 {
		origin, _, _ := pMonitorFromPoint.Call(packPoint(0, 0), monitorNearest)
		found = []screen{describe(origin)}
	}
	primary := found[0]
	for _, s := range found {
		if s.primary {
			primary = s
		}
	}
	return found, primary
}

// describe reads one monitor's rectangles, flags and DPI.
func describe(handle uintptr) screen {
	info := monitorInfo{}
	info.cbSize = uint32(unsafe.Sizeof(info))
	_, _, _ = pGetMonitorInfo.Call(handle, uintptr(unsafe.Pointer(&info)))
	var dpiX, dpiY uint32
	dpi := standardDPI
	if hr, _, _ := pGetDpiForMonitor.Call(handle, mdtEffectiveDPI, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); hr == 0 && dpiX > 0 {
		dpi = int(dpiX)
	}
	return screen{
		Monitor: domain.Monitor{Bounds: toDomain(info.monitor), Work: toDomain(info.work)},
		handle:  handle,
		dpi:     dpi,
		primary: info.flags&monitorPrimary != 0,
	}
}

func toDomain(r rect) domain.Rect {
	return domain.Rect{Left: int(r.left), Top: int(r.top), Right: int(r.right), Bottom: int(r.bottom)}
}

// domains answers the monitors as the domain sees them.
func domains(screens []screen) []domain.Monitor {
	out := make([]domain.Monitor, len(screens))
	for i, s := range screens {
		out[i] = s.Monitor
	}
	return out
}

// screenAt answers the monitor holding p, else the one Windows judges
// nearest, else the primary.
func screenAt(screens []screen, primary screen, p domain.Point) screen {
	for _, s := range screens {
		if s.Bounds.Contains(p) {
			return s
		}
	}
	nearest, _, _ := pMonitorFromPoint.Call(packPoint(int32(p.X), int32(p.Y)), monitorNearest)
	for _, s := range screens {
		if s.handle == nearest {
			return s
		}
	}
	return primary
}
