package ui

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
)

// httpsScheme is the only kind of address handed to the desktop.
const httpsScheme = "https://"

// donateRefused is said on screen when no browser opens, so the entry never
// looks as though it did nothing.
const donateRefused = application.ProductName + " could not open a browser for the support page."

// openExternal hands address to the desktop, which opens it in the user's
// browser. WhatDay fetches nothing itself (NFR-PRIV-001). Anything but an
// https address is refused before the desktop sees it.
func openExternal(address string) error {
	if !strings.HasPrefix(address, httpsScheme) {
		return fmt.Errorf("refusing to open %q: only https addresses go to the browser", address)
	}
	answer, _, _ := pShellExecute.Call(0, uintptr(unsafe.Pointer(wide("open"))), uintptr(unsafe.Pointer(wide(address))), 0, 0, swShowNormal)
	if answer <= shellExecuteFailed {
		return fmt.Errorf("the desktop declined to open %s (ShellExecute answered %d)", address, answer)
	}
	return nil
}

// donate opens the support page (FR-036). A desktop that declines is said
// out loud, in the log and on screen.
func (w *Window) donate() {
	if err := w.open(application.DonateURL); err != nil {
		w.log.Printf("support page not opened: %v", err)
		w.alert(donateRefused)
	}
}

// warn shows text in a message box owned by the indicator.
func (w *Window) warn(text string) {
	_, _, _ = pMessageBox.Call(w.hwnd, uintptr(unsafe.Pointer(wide(text))), uintptr(unsafe.Pointer(wide(application.ProductName))), mbIconWarning)
}
