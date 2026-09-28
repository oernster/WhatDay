package ui

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
)

// Updater answers one update check; *application.UpdateChecker is one. It
// runs off the window's thread, since it waits on the network.
type Updater interface {
	Check(skipped string) application.UpdateStatus
}

// The automatic check runs a few seconds after the strip appears, so it never
// competes with starting up, then once a day while WhatDay runs (FR-038).
const (
	updateTimerID     = 2
	firstUpdateCheck  = 3 * time.Second
	updateCheckPeriod = 24 * time.Hour
)

// wmUpdateChecked tells the window a check has finished; the result waits in
// Window.results.
const wmUpdateChecked = 0x8003 // WM_APP + 3

// downloadRefused is said on screen when no browser opens for a download.
const downloadRefused = application.ProductName + " could not open a browser for the download."

// updateResult is one finished check and whether the user asked for it.
type updateResult struct {
	status application.UpdateStatus
	asked  bool
}

// updateState is the window's side of the update check. Only the window's
// thread reads or writes it; the worker only sends on results.
type updateState struct {
	updater Updater
	results chan application.UpdateStatus
	// checking spans a check from its start until its answer has been shown,
	// so a timer firing while a prompt is open cannot raise a second one.
	checking bool
	asked    bool
	// ask offers a release and tell says a sentence; both are fields so a test
	// can script the answers without opening a dialog.
	ask  func(status application.UpdateStatus) updateChoice
	tell func(text string)
}

func newUpdateState(updater Updater) updateState {
	return updateState{updater: updater, results: make(chan application.UpdateStatus, 1)}
}

// armUpdateCheck starts the automatic check's timer.
func (w *Window) armUpdateCheck(after time.Duration) {
	_, _, _ = pSetTimer.Call(w.hwnd, updateTimerID, uintptr(after/time.Millisecond), 0)
}

// updateTimerFired re-arms the daily check, then runs one.
func (w *Window) updateTimerFired() {
	w.armUpdateCheck(updateCheckPeriod)
	w.checkForUpdates(false)
}

// checkForUpdates starts a check on its own goroutine. An automatic check
// passes the skipped release; one the user asked for ignores it (FR-037). A
// request while a check is under way joins it; the user asking makes it
// theirs.
func (w *Window) checkForUpdates(asked bool) {
	if w.updates.checking {
		w.updates.asked = w.updates.asked || asked
		return
	}
	w.updates.checking, w.updates.asked = true, asked
	skipped := ""
	if !asked {
		skipped = w.controller.SkippedVersion()
	}
	go w.runCheck(skipped)
}

// runCheck is the worker. A panic is recovered here, on the goroutine that
// can raise it; it becomes an unreachable check that says why. The window
// therefore always hears back; a check the user asked for never goes quiet.
func (w *Window) runCheck(skipped string) {
	var status application.UpdateStatus
	defer func() {
		if r := recover(); r != nil {
			status = application.UpdateStatus{Outcome: application.UpdateUnreachable, Fault: fmt.Sprintf("the check panicked: %v", r)}
		}
		w.updates.results <- status
		if w.hwnd != 0 {
			_, _, _ = pPostMessage.Call(w.hwnd, wmUpdateChecked, 0, 0)
		}
	}()
	status = w.updates.updater.Check(skipped)
}

// updateChecked shows a finished check's answer on the window's thread.
func (w *Window) updateChecked() {
	result := updateResult{status: <-w.updates.results, asked: w.updates.asked}
	w.answer(result)
	w.updates.checking, w.updates.asked = false, false
}

// answer shows result as the application decides (FR-037, FR-038).
func (w *Window) answer(result updateResult) {
	status := result.status
	if status.Fault != "" {
		w.log.Printf("update check failed: %s", status.Fault)
	}
	switch status.Reply(result.asked) {
	case application.ReplyPrompt:
		w.log.Printf("%s %s is available", application.ProductName, status.Latest)
		w.actOn(w.updates.ask(status), status)
	case application.ReplyUpToDate:
		w.updates.tell(application.UpToDateText)
	case application.ReplyUnreachable:
		w.updates.tell(application.UnreachableText)
	}
}

// actOn carries out the user's answer to the prompt.
func (w *Window) actOn(choice updateChoice, status application.UpdateStatus) {
	switch choice {
	case chooseDownload:
		if err := w.open(status.DownloadURL); err != nil {
			w.log.Printf("download not opened: %v", err)
			w.alert(downloadRefused)
		}
	case chooseSkip:
		w.controller.SkipVersion(status.Tag)
	}
}

// inform shows text in a message box owned by the indicator.
func (w *Window) inform(text string) {
	_, _, _ = pMessageBox.Call(w.hwnd, uintptr(unsafe.Pointer(wide(text))), uintptr(unsafe.Pointer(wide(application.ProductName))), mbIconInfo)
}
