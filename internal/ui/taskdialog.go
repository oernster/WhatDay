package ui

import (
	"encoding/binary"
	"runtime"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
	"golang.org/x/sys/windows"
)

// comctl32 exports TaskDialogIndirect only in version 6, which a process gets
// by asking for it in its manifest (assets/WhatDay.manifest). Measured
// 2026-09-28: without the manifest the export is absent; with it the call
// answers S_OK.
var (
	comctl      = windows.NewLazySystemDLL("comctl32.dll")
	pTaskDialog = comctl.NewProc("TaskDialogIndirect")
)

// commctrl.h packs TASKDIALOGCONFIG and TASKDIALOG_BUTTON to one byte, so on
// x64 their pointers sit at offsets no Go struct would give them. The records
// are written byte by byte at these offsets instead. Measured 2026-09-28 by a
// probe that showed a dialog built this way, captured it and read back every
// field in its place.
const (
	configSize        = 160
	offsetSize        = 0
	offsetParent      = 4
	offsetFlags       = 20
	offsetTitle       = 28
	offsetIcon        = 36
	offsetHeadline    = 44
	offsetContent     = 52
	offsetButtonCount = 60
	offsetButtons     = 64
	offsetDefault     = 72

	buttonSize        = 12
	offsetButtonLabel = 4

	// tdfAllowCancellation lets Esc and the close box answer idCancel.
	tdfAllowCancellation = 0x0008
	// tdInformationIcon is MAKEINTRESOURCE(-3), the system's information icon.
	tdInformationIcon = 0xFFFD
)

// The prompt's button identifiers, then the message box's Yes and No. Esc and
// the close box answer IDCANCEL, which is Later like anything unnamed here.
const (
	idDownload = 100 + iota
	idSkip
	idLater
	idYes = 6
	idNo  = 7
)

// mbYesNoCancel is the fallback prompt's button set.
const mbYesNoCancel = 0x3

// updateChoice is what the user answered the prompt with.
type updateChoice int

const (
	chooseLater updateChoice = iota
	chooseDownload
	chooseSkip
)

// dialogButton is one of the prompt's buttons.
type dialogButton struct {
	id    int32
	label string
}

// promptButtons are the prompt's three answers, in order (FR-038).
var promptButtons = []dialogButton{
	{idDownload, application.DownloadLabel},
	{idSkip, application.SkipLabel},
	{idLater, application.LaterLabel},
}

// taskDialog is a TASKDIALOGCONFIG ready to hand over. The strings are kept
// here so the collector cannot free them while Windows reads the record.
type taskDialog struct {
	config  []byte
	buttons []byte
	strings []*uint16
}

func (d *taskDialog) text(s string) uint64 {
	p := wide(s)
	d.strings = append(d.strings, p)
	return uint64(uintptr(unsafe.Pointer(p)))
}

// newTaskDialog builds the record for a dialog owned by parent.
func newTaskDialog(parent uintptr, title, headline, content string, buttons []dialogButton) *taskDialog {
	d := &taskDialog{config: make([]byte, configSize), buttons: make([]byte, len(buttons)*buttonSize)}
	le := binary.LittleEndian
	for i, b := range buttons {
		le.PutUint32(d.buttons[i*buttonSize:], uint32(b.id))
		le.PutUint64(d.buttons[i*buttonSize+offsetButtonLabel:], d.text(b.label))
	}
	le.PutUint32(d.config[offsetSize:], configSize)
	le.PutUint64(d.config[offsetParent:], uint64(parent))
	le.PutUint32(d.config[offsetFlags:], tdfAllowCancellation)
	le.PutUint64(d.config[offsetTitle:], d.text(title))
	le.PutUint64(d.config[offsetIcon:], tdInformationIcon)
	le.PutUint64(d.config[offsetHeadline:], d.text(headline))
	le.PutUint64(d.config[offsetContent:], d.text(content))
	le.PutUint32(d.config[offsetButtonCount:], uint32(len(buttons)))
	if len(buttons) > 0 {
		le.PutUint64(d.config[offsetButtons:], uint64(uintptr(unsafe.Pointer(&d.buttons[0]))))
		le.PutUint32(d.config[offsetDefault:], uint32(buttons[0].id))
	}
	return d
}

// show runs the dialog; it answers the button pressed and whether Windows
// showed it at all.
func (d *taskDialog) show() (int32, bool) {
	var pressed int32
	result, _, _ := pTaskDialog.Call(uintptr(unsafe.Pointer(&d.config[0])), uintptr(unsafe.Pointer(&pressed)), 0, 0)
	runtime.KeepAlive(d)
	return pressed, result == 0
}

// choiceFor maps a pressed button, from either dialog, to the user's answer.
// Anything else, the close box included, is Later.
func choiceFor(pressed int32) updateChoice {
	switch pressed {
	case idDownload, idYes:
		return chooseDownload
	case idSkip, idNo:
		return chooseSkip
	}
	return chooseLater
}

// askUpdate offers status's release (FR-038): the three named buttons where
// Windows has them, else a Yes, No, Cancel box whose text says which is which.
func (w *Window) askUpdate(status application.UpdateStatus) updateChoice {
	if pTaskDialog.Find() == nil {
		dialog := newTaskDialog(w.hwnd, application.ProductName, status.Headline(), status.Detail(), promptButtons)
		if pressed, shown := dialog.show(); shown {
			return choiceFor(pressed)
		}
		w.log.Printf("the update dialog was refused; asking with a message box")
	}
	text := status.Headline() + "\n" + status.Detail() + "\n\n" + application.FallbackKey
	pressed, _, _ := pMessageBox.Call(w.hwnd, uintptr(unsafe.Pointer(wide(text))), uintptr(unsafe.Pointer(wide(application.ProductName))), mbIconInfo|mbYesNoCancel)
	return choiceFor(int32(pressed))
}
