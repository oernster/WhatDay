// Package ui is WhatDay's Windows surface: the indicator window, the tray icon
// with its menu, plus the message loop that carries midnight, resume, clock
// and display changes to the application. Every Win32 call lives here or in
// infrastructure; the domain and application never see one.
//
// Built from three feasibility probes measured on the reference machine
// (REQUIREMENTS.md Appendix A).
package ui

import "golang.org/x/sys/windows"

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")
	shell  = windows.NewLazySystemDLL("shell32.dll")
	shcore = windows.NewLazySystemDLL("shcore.dll")
	dwm    = windows.NewLazySystemDLL("dwmapi.dll")
	kernel = windows.NewLazySystemDLL("kernel32.dll")

	pRegisterClassEx  = user32.NewProc("RegisterClassExW")
	pCreateWindowEx   = user32.NewProc("CreateWindowExW")
	pDefWindowProc    = user32.NewProc("DefWindowProcW")
	pGetMessage       = user32.NewProc("GetMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessage  = user32.NewProc("DispatchMessageW")
	pPostQuitMessage  = user32.NewProc("PostQuitMessage")
	pPostMessage      = user32.NewProc("PostMessageW")
	pShowWindow       = user32.NewProc("ShowWindow")
	pSetTimer         = user32.NewProc("SetTimer")
	pCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	pAppendMenu       = user32.NewProc("AppendMenuW")
	pTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	pDestroyMenu      = user32.NewProc("DestroyMenu")
	pGetCursorPos     = user32.NewProc("GetCursorPos")
	pSetForeground    = user32.NewProc("SetForegroundWindow")
	pSetDpiAwareness  = user32.NewProc("SetProcessDpiAwarenessContext")
	pDrawText         = user32.NewProc("DrawTextW")
	pBeginPaint       = user32.NewProc("BeginPaint")
	pEndPaint         = user32.NewProc("EndPaint")
	pGetWindowRect    = user32.NewProc("GetWindowRect")
	pInvalidateRect   = user32.NewProc("InvalidateRect")
	pSetWindowPos     = user32.NewProc("SetWindowPos")
	pSetCapture       = user32.NewProc("SetCapture")
	pReleaseCapture   = user32.NewProc("ReleaseCapture")
	pGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	pGetDC            = user32.NewProc("GetDC")
	pReleaseDC        = user32.NewProc("ReleaseDC")
	pLoadCursor       = user32.NewProc("LoadCursorW")
	pLoadIcon         = user32.NewProc("LoadIconW")
	pMessageBox       = user32.NewProc("MessageBoxW")
	pRegisterWinMsg   = user32.NewProc("RegisterWindowMessageW")
	pMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfo   = user32.NewProc("GetMonitorInfoW")
	pEnumMonitors     = user32.NewProc("EnumDisplayMonitors")
	pCreateCompatDC   = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection = gdi32.NewProc("CreateDIBSection")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pDeleteDC         = gdi32.NewProc("DeleteDC")
	pCreateFont       = gdi32.NewProc("CreateFontW")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pGetTextExtent    = gdi32.NewProc("GetTextExtentPoint32W")
	pBitBlt           = gdi32.NewProc("BitBlt")
	pShellNotifyIcon  = shell.NewProc("Shell_NotifyIconW")
	pSHAppBarMessage  = shell.NewProc("SHAppBarMessage")
	pExtractIconEx    = shell.NewProc("ExtractIconExW")
	pGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")
	pDwmSetAttr       = dwm.NewProc("DwmSetWindowAttribute")
	pDwmExtendFrame   = dwm.NewProc("DwmExtendFrameIntoClientArea")
	pGetModuleHandle  = kernel.NewProc("GetModuleHandleW")
)

// Window messages and their arguments.
const (
	wmDestroy        = 0x0002
	wmPaint          = 0x000F
	wmSettingChange  = 0x001A
	wmTimeChange     = 0x001E
	wmDisplayChange  = 0x007E
	wmTimer          = 0x0113
	wmMouseMove      = 0x0200
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmRButtonUp      = 0x0205
	wmPowerBroadcast = 0x0218
	wmDpiChanged     = 0x02E0
	wmNull           = 0x0000
	wmTray           = 0x8001 // WM_APP + 1
	wmAppBar         = 0x8002 // WM_APP + 2

	pbtResumeSuspend   = 0x0007
	pbtResumeAutomatic = 0x0012
	spiSetWorkArea     = 0x002F
)

// Window styles, placement flags and system metrics.
const (
	wsPopup         = 0x80000000
	wsExTopmost     = 0x00000008
	wsExToolWindow  = 0x00000080
	swShowNoAct     = 4
	swHide          = 0
	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpShowWindow   = 0x0040
	hwndTopmost     = ^uintptr(0) // HWND_TOPMOST (-1)
	smCxDrag        = 68
	smCyDrag        = 69
	monitorNearest  = 2
	monitorPrimary  = 0x1 // MONITORINFOF_PRIMARY
	mdtEffectiveDPI = 0
	idcArrow        = 32512
	idiApplication  = 32512
	dpiPerMonitor2  = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
)

// Menus, tray, appbar and message box.
const (
	mfString         = 0x0000
	mfChecked        = 0x0008
	mfSeparator      = 0x0800
	tpmReturnCmd     = 0x0100
	tpmBottomAlign   = 0x0020
	nimAdd           = 0
	nimDelete        = 2
	nifMessage       = 1
	nifIcon          = 2
	nifTip           = 4
	abmNew           = 0
	abmRemove        = 1
	abnFullscreenApp = 2
	mbIconInfo       = 0x40
)

// Drawing and DWM.
const (
	bkTransparent = 1
	antialiased   = 4
	fwSemibold    = 600
	dtCentre      = 0x0001 | 0x0004 | 0x0020 // DT_CENTER | DT_VCENTER | DT_SINGLELINE
	srcCopy       = 0x00CC0020
	dwmaDarkMode  = 20
	dwmaCorners   = 33
	dwmaBackdrop  = 38
	dwmcpRound    = 2
	dwmsbtAcrylic = 3
	dwordSize     = 4
	fontFace      = "Segoe UI"
)

type point struct{ x, y int32 }
type size struct{ cx, cy int32 }
type rect struct{ left, top, right, bottom int32 }
type margins struct{ l, r, t, b int32 }

type wndClassEx struct {
	size, style                uint32
	wndProc                    uintptr
	clsExtra, wndExtra         int32
	instance, icon, cursor, bg uintptr
	menuName, className        *uint16
	iconSm                     uintptr
}

type msg struct {
	hwnd, message, wParam, lParam uintptr
	time                          uint32
	x, y                          int32
	private                       uint32
}

type paintStruct struct {
	hdc                uintptr
	erase              int32
	rc                 rect
	restore, incUpdate int32
	reserved           [32]byte
}

type notifyIconData struct {
	cbSize           uint32
	hwnd             uintptr
	id, flags, cbMsg uint32
	icon             uintptr
	tip              [128]uint16
	state, stateMask uint32
	info             [256]uint16
	version          uint32
	infoTitle        [64]uint16
	infoFlags        uint32
	guid             [16]byte
	balloonIcon      uintptr
}

type appBarData struct {
	cbSize      uint32
	hwnd        uintptr
	cbMsg, edge uint32
	rc          rect
	lParam      uintptr
}

type monitorInfo struct {
	cbSize        uint32
	monitor, work rect
	flags         uint32
}

type bitmapInfoHeader struct {
	size                   uint32
	width, height          int32
	planes, bitCount       uint16
	compression, sizeImage uint32
	xPels, yPels           int32
	clrUsed, clrImportant  uint32
}

// packPoint passes a POINT by value, as MonitorFromPoint takes it on x64.
func packPoint(x, y int32) uintptr {
	return uintptr(uint32(x)) | uintptr(uint32(y))<<32
}

// windowsString answers s as UTF-16 with its terminating NUL; empty for a
// string holding NUL.
func windowsString(s string) []uint16 {
	u, err := windows.UTF16FromString(s)
	if err != nil {
		return nil
	}
	return u
}

// wide converts s for a W call; a string holding NUL is replaced by an empty
// one rather than failing, since every string passed here is our own.
func wide(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		empty := uint16(0)
		return &empty
	}
	return p
}
