package ui

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
	"golang.org/x/sys/windows"
)

// className names the indicator's window class.
const className = application.ProductName + "Indicator"

// NewWindow prepares the indicator; Run opens it.
func NewWindow(log application.Log, now func() time.Time) *Window {
	return &Window{log: log, now: now}
}

// Run opens the indicator and the tray icon, starts controller and pumps
// messages until the window is destroyed. It answers an error only when the
// window cannot be made at all.
func (w *Window) Run(controller Controller) error {
	// Win32 windows belong to the thread that made them.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w.controller = controller

	if ok, _, _ := pSetDpiAwareness.Call(dpiPerMonitor2); ok == 0 {
		w.log.Printf("per-monitor DPI awareness refused; sizes may be scaled by Windows")
	}
	taskbarCreated, _, _ := pRegisterWinMsg.Call(uintptr(unsafe.Pointer(wide("TaskbarCreated"))))
	if err := w.create(taskbarCreated); err != nil {
		return err
	}
	w.applyDwm()
	controller.Start()
	w.layout()
	_, _, _ = pShowWindow.Call(w.hwnd, swShowNoAct)
	w.addTray()
	w.appBar(abmNew)

	var m msg
	for {
		got, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(got) <= 0 {
			return nil
		}
		_, _, _ = pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// create registers the class and makes the borderless, topmost tool window
// (FR-010, FR-012, FR-013), hidden until it has been laid out.
func (w *Window) create(taskbarCreated uintptr) error {
	instance, _, _ := pGetModuleHandle.Call(0)
	cursor, _, _ := pLoadCursor.Call(0, idcArrow)
	class := wndClassEx{
		wndProc: windows.NewCallback(func(hwnd, message, wParam, lParam uintptr) uintptr {
			return w.handle(hwnd, message, wParam, lParam, taskbarCreated)
		}),
		instance:  instance,
		cursor:    cursor,
		className: wide(className),
	}
	class.size = uint32(unsafe.Sizeof(class))
	if atom, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return fmt.Errorf("registering the window class: %w", err)
	}
	hwnd, _, err := pCreateWindowEx.Call(wsExTopmost|wsExToolWindow, uintptr(unsafe.Pointer(class.className)), uintptr(unsafe.Pointer(wide(application.ProductName))), wsPopup, 0, 0, 1, 1, 0, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("creating the indicator window: %w", err)
	}
	w.hwnd = hwnd
	return nil
}

// handle is the window procedure.
func (w *Window) handle(hwnd, message, wParam, lParam, taskbarCreated uintptr) uintptr {
	switch message {
	case wmPaint:
		w.paint()
		return 0
	case wmLButtonDown:
		w.pressed()
		return 0
	case wmMouseMove:
		w.moved()
		return 0
	case wmLButtonUp:
		w.released()
		return 0
	case wmTray:
		if lParam == wmLButtonUp || lParam == wmRButtonUp {
			w.showMenu()
		}
		return 0
	case wmAppBar:
		if wParam == abnFullscreenApp {
			w.fullscreen(lParam != 0)
		}
		return 0
	case wmTimer:
		// Every wake refreshes; Refresh re-arms (FR-003).
		w.controller.Refresh()
		return 0
	case wmTimeChange:
		w.log.Printf("the clock or timezone changed")
		w.controller.Refresh()
		return 0
	case wmPowerBroadcast:
		if wParam == pbtResumeAutomatic || wParam == pbtResumeSuspend {
			w.log.Printf("resumed from sleep")
			w.controller.Refresh()
		}
		return 1 // TRUE: the broadcast is acknowledged
	case wmDpiChanged:
		dpi := wParam & lowWordMask
		if w.drag.pressed {
			// A drag onto a monitor of another scale: the drag sizes the
			// strip itself. Laying out here would put it back at the saved
			// position, which is the old one until the drag ends.
			w.log.Printf("DPI changed to %d during a drag; the drag sizes the strip", dpi)
			return 0
		}
		w.log.Printf("DPI changed to %d", dpi)
		w.layout()
		return 0
	case wmDisplayChange:
		w.log.Printf("displays changed")
		w.layout()
		return 0
	case wmSettingChange:
		if wParam == spiSetWorkArea {
			w.layout()
		}
	case wmDestroy:
		w.removeTray()
		w.appBar(abmRemove)
		_, _, _ = pPostQuitMessage.Call(0)
		return 0
	}
	if message == taskbarCreated && taskbarCreated != 0 {
		// Explorer restarted: the tray and the appbar list were lost (FR-024).
		w.log.Printf("Explorer restarted; restoring the tray icon")
		w.trayAdded = false
		w.addTray()
		w.appBar(abmNew)
		w.layout()
		return 0
	}
	result, _, _ := pDefWindowProc.Call(hwnd, message, wParam, lParam)
	return result
}
