package ui

import (
	"time"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
)

// Controller is what the window asks of the application; *application.Indicator
// is one.
type Controller interface {
	Start()
	Refresh()
	Menu() []application.MenuItem
	ChooseColour(name string)
	MovedTo(position domain.Point)
	Placement(size domain.Size, monitors []domain.Monitor, primaryWork domain.Rect) domain.Point
}

// maxWakeGap bounds each wait on the way to midnight. A timer that drifts
// from the wall clock is put right within a minute; so is a missed resume or
// clock-change message (FR-003 to FR-005, domain.WakeDelay).
const maxWakeGap = time.Minute

// wakeTimerID names the one Win32 timer the window keeps.
const wakeTimerID = 1

// Window is the indicator: the application's View and Scheduler.
type Window struct {
	hwnd       uintptr
	controller Controller
	log        application.Log
	now        func() time.Time

	day    string
	colour domain.Colour
	size   domain.Size
	wake   time.Time
	hidden bool

	drag      dragState
	trayIcon  uintptr
	trayAdded bool
}

type dragState struct {
	pressed, moving bool
	pressAt, winAt  point
	screens         []screen
	primary         screen
}

// ShowDay paints name, repainting only when it changes (View).
func (w *Window) ShowDay(name string) {
	if name == w.day {
		return
	}
	w.day = name
	w.log.Printf("showing %s", name)
	w.invalidate()
}

// SetColour paints the day name in colour from now on (View).
func (w *Window) SetColour(colour domain.Colour) {
	w.colour = colour
	w.invalidate()
}

// WakeAt arms the wake-up for instant (Scheduler). Each wait is at most
// maxWakeGap; every wake calls Refresh, which re-arms.
func (w *Window) WakeAt(instant time.Time) {
	w.wake = instant
	wait, _ := domain.WakeDelay(w.now(), instant, maxWakeGap)
	ms := (wait + time.Millisecond - 1) / time.Millisecond // round up: never early
	_, _, _ = pSetTimer.Call(w.hwnd, wakeTimerID, uintptr(max(ms, 1)), 0)
}

func (w *Window) invalidate() {
	if w.hwnd != 0 {
		_, _, _ = pInvalidateRect.Call(w.hwnd, 0, 0)
	}
}

func (w *Window) paint() {
	var ps paintStruct
	dc, _, _ := pBeginPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
	f := render(w.day, w.colour, w.size)
	_, _, _ = pBitBlt.Call(dc, 0, 0, uintptr(w.size.W), uintptr(w.size.H), f.dc, 0, 0, srcCopy)
	f.release()
	_, _, _ = pEndPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
}

// applyDwm gives the window the dark Acrylic backdrop with rounded corners
// (FR-011, FR-015). Each call was measured to answer S_OK on a borderless
// popup (M-5); a refusal is logged, not fatal.
func (w *Window) applyDwm() {
	set := func(what string, attribute uintptr, value uint32) {
		if hr, _, _ := pDwmSetAttr.Call(w.hwnd, attribute, uintptr(unsafe.Pointer(&value)), dwordSize); hr != 0 {
			w.log.Printf("DWM refused %s: 0x%X", what, hr)
		}
	}
	set("dark mode", dwmaDarkMode, 1)
	set("rounded corners", dwmaCorners, dwmcpRound)
	whole := margins{-1, -1, -1, -1}
	if hr, _, _ := pDwmExtendFrame.Call(w.hwnd, uintptr(unsafe.Pointer(&whole))); hr != 0 {
		w.log.Printf("DWM refused the frame extension: 0x%X", hr)
	}
	set("the Acrylic backdrop", dwmaBackdrop, dwmsbtAcrylic)
}

// layout sizes the indicator for the monitor it sits on and places it where
// the application says (FR-014, FR-021 to FR-023).
func (w *Window) layout() {
	screens, primary := monitors()
	w.size = sizeOn(primary, primary)
	pos := w.controller.Placement(w.size, domains(screens), primary.Work)
	here := screenAt(screens, primary, centre(pos, w.size))
	w.size = sizeOn(here, primary)
	pos = w.controller.Placement(w.size, domains(screens), primary.Work)
	w.moveTo(pos, w.size)
}

func (w *Window) moveTo(pos domain.Point, s domain.Size) {
	flags := uintptr(swpNoActivate)
	if w.hidden {
		flags |= swpNoZOrder
	}
	_, _, _ = pSetWindowPos.Call(w.hwnd, hwndTopmost, uintptr(pos.X), uintptr(pos.Y), uintptr(s.W), uintptr(s.H), flags)
	w.invalidate()
}

// sizeOn answers the indicator's size on s.
func sizeOn(s, primary screen) domain.Size {
	height := domain.IndicatorHeight(s.Monitor, s.dpi, primary.Monitor, primary.dpi)
	return domain.IndicatorSize(height, widestDayWidth(height))
}

func centre(p domain.Point, s domain.Size) domain.Point {
	return domain.Point{X: p.X + s.W/2, Y: p.Y + s.H/2}
}

func (w *Window) pressed() {
	w.drag = dragState{pressed: true}
	_, _, _ = pGetCursorPos.Call(uintptr(unsafe.Pointer(&w.drag.pressAt)))
	var r rect
	_, _, _ = pGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	w.drag.winAt = point{r.left, r.top}
	w.drag.screens, w.drag.primary = monitors()
	_, _, _ = pSetCapture.Call(w.hwnd)
}

// moved follows the pointer once it passes the drag threshold, keeping the
// strip inside the work area of the monitor it is over (FR-017, FR-018).
func (w *Window) moved() {
	if !w.drag.pressed {
		return
	}
	var p point
	_, _, _ = pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	dx, dy := int(p.x-w.drag.pressAt.x), int(p.y-w.drag.pressAt.y)
	tx, _, _ := pGetSystemMetrics.Call(smCxDrag)
	ty, _, _ := pGetSystemMetrics.Call(smCyDrag)
	if !w.drag.moving && !domain.PassedDragThreshold(dx, dy, int(tx), int(ty)) {
		return
	}
	w.drag.moving = true
	wanted := domain.Point{X: int(w.drag.winAt.x) + dx, Y: int(w.drag.winAt.y) + dy}
	here := screenAt(w.drag.screens, w.drag.primary, centre(wanted, w.size))
	w.size = sizeOn(here, w.drag.primary)
	w.moveTo(domain.ClampToWork(wanted, w.size, here.Work), w.size)
}

// released ends a press: a drag saves where it ended; a click does nothing
// (FR-019).
func (w *Window) released() {
	_, _, _ = pReleaseCapture.Call()
	moved := w.drag.moving
	w.drag = dragState{}
	if !moved {
		return
	}
	var r rect
	_, _, _ = pGetWindowRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	w.controller.MovedTo(domain.Point{X: int(r.left), Y: int(r.top)})
}

// fullscreen hides the strip while the shell reports a fullscreen app, then
// brings it back on top (FR-020, M-3).
func (w *Window) fullscreen(opening bool) {
	w.hidden = opening
	if opening {
		_, _, _ = pShowWindow.Call(w.hwnd, swHide)
		return
	}
	_, _, _ = pSetWindowPos.Call(w.hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
}
