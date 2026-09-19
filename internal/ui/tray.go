package ui

import (
	"os"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
)

// Menu command identifiers. Zero is what TrackPopupMenu answers when the menu
// is dismissed, so no entry uses it.
const (
	cmdAbout      = 1
	cmdColourBase = 100
)

// menuEntry is one Win32 menu item built from the application's menu model.
type menuEntry struct {
	id, flags uintptr
	label     string
}

// menuEntries turns the menu model into Win32 items (FR-032).
func menuEntries(items []application.MenuItem) []menuEntry {
	entries := make([]menuEntry, 0, len(items))
	colours := 0
	for _, item := range items {
		switch item.Kind {
		case application.MenuAbout:
			entries = append(entries, menuEntry{id: cmdAbout, flags: mfString, label: item.Label})
		case application.MenuSeparator:
			entries = append(entries, menuEntry{flags: mfSeparator})
		case application.MenuColour:
			flags := uintptr(mfString)
			if item.Checked {
				flags |= mfChecked
			}
			entries = append(entries, menuEntry{id: uintptr(cmdColourBase + colours), flags: flags, label: item.Label})
			colours++
		}
	}
	return entries
}

// chosen answers the entry a command identifies; false when the menu was
// dismissed or the command is not one of these entries.
func chosen(entries []menuEntry, command uintptr) (menuEntry, bool) {
	for _, e := range entries {
		if command != 0 && e.id == command {
			return e, true
		}
	}
	return menuEntry{}, false
}

// showMenu opens the tray menu at the pointer and acts on the choice
// (FR-031 to FR-034).
func (w *Window) showMenu() {
	entries := menuEntries(w.controller.Menu())
	menu, _, _ := pCreatePopupMenu.Call()
	defer func() { _, _, _ = pDestroyMenu.Call(menu) }()
	for _, e := range entries {
		var label uintptr
		if e.label != "" {
			label = uintptr(unsafe.Pointer(wide(e.label)))
		}
		_, _, _ = pAppendMenu.Call(menu, e.flags, e.id, label)
	}
	var at point
	_, _, _ = pGetCursorPos.Call(uintptr(unsafe.Pointer(&at)))
	// The menu closes on a click elsewhere only if its owner is in front.
	_, _, _ = pSetForeground.Call(w.hwnd)
	command, _, _ := pTrackPopupMenu.Call(menu, tpmReturnCmd|tpmBottomAlign, uintptr(at.x), uintptr(at.y), 0, w.hwnd, 0)
	_, _, _ = pPostMessage.Call(w.hwnd, wmNull, 0, 0)

	entry, ok := chosen(entries, command)
	switch {
	case !ok:
	case entry.id == cmdAbout:
		w.showAbout()
	default:
		w.controller.ChooseColour(entry.label)
	}
}

func (w *Window) showAbout() {
	about := application.NewAbout()
	_, _, _ = pMessageBox.Call(w.hwnd, uintptr(unsafe.Pointer(wide(about.Text()))), uintptr(unsafe.Pointer(wide(about.Title))), mbIconInfo)
}

// ownIcon loads the icon embedded in this executable, falling back to the
// shell's generic application icon when the binary carries none: a build
// without an icon resource is normal before packaging, so this must not fail.
// Ported from ED Voyage Companion's tray.
func ownIcon() uintptr {
	if path, err := os.Executable(); err == nil {
		var large, small uintptr
		count, _, _ := pExtractIconEx.Call(uintptr(unsafe.Pointer(wide(path))), 0, uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), 1)
		if count > 0 && small != 0 {
			return small
		}
		if count > 0 && large != 0 {
			return large
		}
	}
	generic, _, _ := pLoadIcon.Call(0, idiApplication)
	return generic
}

func (w *Window) trayData() notifyIconData {
	data := notifyIconData{hwnd: w.hwnd, id: 1, flags: nifMessage | nifIcon | nifTip, cbMsg: wmTray, icon: w.trayIcon}
	data.cbSize = uint32(unsafe.Sizeof(data))
	copy(data.tip[:len(data.tip)-1], windowsString(application.ProductName))
	return data
}

// addTray shows the tray icon (FR-030); again after Explorer restarts (FR-024).
func (w *Window) addTray() {
	if w.trayIcon == 0 {
		w.trayIcon = ownIcon()
	}
	data := w.trayData()
	added, _, _ := pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&data)))
	w.trayAdded = added != 0
	if !w.trayAdded {
		w.log.Printf("the tray refused the icon")
	}
}

func (w *Window) removeTray() {
	if !w.trayAdded {
		return
	}
	data := w.trayData()
	_, _, _ = pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	w.trayAdded = false
}

// appBar registers or removes the window with the shell, which then sends it
// the fullscreen signal the taskbar hides on (FR-020, M-3).
func (w *Window) appBar(message uintptr) {
	data := appBarData{hwnd: w.hwnd, cbMsg: wmAppBar}
	data.cbSize = uint32(unsafe.Sizeof(data))
	if done, _, _ := pSHAppBarMessage.Call(message, uintptr(unsafe.Pointer(&data))); done == 0 && message == abmNew {
		w.log.Printf("the shell refused the appbar registration; fullscreen apps will not hide the strip")
	}
}
