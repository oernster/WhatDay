package ui

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
	"golang.org/x/sys/windows"
)

// These tests never open a dialog, a browser or a connection: the checker,
// the prompt, the sentences and the opener are all replaced.

// stubController records the skip; the rest of Controller is unused here.
type stubController struct {
	skipped string
	skips   []string
}

func (c *stubController) Start()                       {}
func (c *stubController) Refresh()                     {}
func (c *stubController) Menu() []application.MenuItem { return nil }
func (c *stubController) ChooseColour(string)          {}
func (c *stubController) MovedTo(domain.Point)         {}
func (c *stubController) SkippedVersion() string       { return c.skipped }
func (c *stubController) SkipVersion(tag string)       { c.skips = append(c.skips, tag) }
func (c *stubController) Placement(domain.Size, []domain.Monitor, domain.Rect) domain.Point {
	return domain.Point{}
}

// fakeUpdater answers status, recording what it was asked to skip.
type fakeUpdater struct {
	status application.UpdateStatus
	asked  chan string
	panics bool
}

func (u *fakeUpdater) Check(skipped string) application.UpdateStatus {
	if u.asked != nil {
		u.asked <- skipped
	}
	if u.panics {
		panic("checker broke")
	}
	return u.status
}

// updateRig is a window whose update surfaces are recorded.
type updateRig struct {
	window     *Window
	controller *stubController
	log        *recordingLog
	choice     updateChoice
	offered    []application.UpdateStatus
	told       []string
	opened     []string
	alerted    []string
}

func newUpdateRig(updater Updater, openErr error) *updateRig {
	r := &updateRig{controller: &stubController{skipped: "v1.1.5"}, log: &recordingLog{}}
	r.window = &Window{
		controller: r.controller,
		log:        r.log,
		open:       func(address string) error { r.opened = append(r.opened, address); return openErr },
		alert:      func(text string) { r.alerted = append(r.alerted, text) },
		updates:    newUpdateState(updater),
	}
	r.window.updates.ask = func(status application.UpdateStatus) updateChoice {
		r.offered = append(r.offered, status)
		return r.choice
	}
	r.window.updates.tell = func(text string) { r.told = append(r.told, text) }
	return r
}

var offer = application.UpdateStatus{
	Outcome: application.UpdateAvailable, Current: "1.1.0", Latest: "1.2.0", Tag: "v1.2.0",
	DownloadURL: "https://github.com/oernster/WhatDay/releases/download/v1.2.0/WhatDaySetup.exe",
}

// finish hands status to the window as a finished check would.
func (r *updateRig) finish(status application.UpdateStatus, asked bool) {
	r.window.updates.checking, r.window.updates.asked = true, asked
	r.window.updates.results <- status
	r.window.updateChecked()
}

func TestTheWorkerHandsBackWhatTheCheckerAnswered(t *testing.T) {
	t.Parallel()
	r := newUpdateRig(&fakeUpdater{status: offer}, nil)
	r.window.runCheck("")
	if got := <-r.window.updates.results; got != offer {
		t.Fatalf("got %+v", got)
	}
}

func TestAPanickingCheckIsUnreachableAndSaysWhy(t *testing.T) {
	t.Parallel()
	r := newUpdateRig(&fakeUpdater{panics: true}, nil)
	r.window.runCheck("")
	got := <-r.window.updates.results
	if got.Outcome != application.UpdateUnreachable || !strings.Contains(got.Fault, "checker broke") {
		t.Fatalf("got %+v", got)
	}
}

func TestAnAutomaticCheckPassesTheSkipAndOneAskedForDoesNot(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		asked bool
		want  string
	}{{false, "v1.1.5"}, {true, ""}} {
		updater := &fakeUpdater{asked: make(chan string, 1)}
		r := newUpdateRig(updater, nil)
		r.window.checkForUpdates(c.asked)
		select {
		case got := <-updater.asked:
			if got != c.want {
				t.Errorf("asked %v: skipped %q, want %q", c.asked, got, c.want)
			}
		case <-time.After(time.Second):
			t.Fatalf("asked %v: the checker was never called", c.asked)
		}
		<-r.window.updates.results
	}
}

func TestARequestDuringACheckJoinsIt(t *testing.T) {
	t.Parallel()
	r := newUpdateRig(&fakeUpdater{}, nil)
	r.window.updates.checking = true
	r.window.checkForUpdates(true)
	if !r.window.updates.asked {
		t.Fatal("asking during an automatic check did not make it the user's")
	}
	r.window.checkForUpdates(false)
	if !r.window.updates.asked {
		t.Fatal("a timer during an asked check made it automatic again")
	}
	if len(r.window.updates.results) != 0 {
		t.Fatal("a second check was started")
	}
}

func TestEachAnswerToThePromptIsCarriedOut(t *testing.T) {
	t.Parallel()
	download := newUpdateRig(nil, nil)
	download.choice = chooseDownload
	download.finish(offer, false)
	if len(download.opened) != 1 || download.opened[0] != offer.DownloadURL || len(download.alerted) != 0 {
		t.Fatalf("download opened %q, alerted %q", download.opened, download.alerted)
	}

	skip := newUpdateRig(nil, nil)
	skip.choice = chooseSkip
	skip.finish(offer, false)
	if len(skip.controller.skips) != 1 || skip.controller.skips[0] != "v1.2.0" || len(skip.opened) != 0 {
		t.Fatalf("skip remembered %q, opened %q", skip.controller.skips, skip.opened)
	}

	later := newUpdateRig(nil, nil)
	later.choice = chooseLater
	later.finish(offer, false)
	if len(later.offered) != 1 || len(later.opened) != 0 || len(later.controller.skips) != 0 {
		t.Fatalf("later offered %d, opened %q, skipped %q", len(later.offered), later.opened, later.controller.skips)
	}
	if later.window.updates.checking {
		t.Fatal("the check still reads as under way after its answer")
	}
}

func TestADownloadWithNoBrowserSaysSo(t *testing.T) {
	t.Parallel()
	r := newUpdateRig(nil, errors.New("no browser"))
	r.choice = chooseDownload
	r.finish(offer, false)
	if len(r.alerted) != 1 || r.alerted[0] != downloadRefused {
		t.Fatalf("alerted %q", r.alerted)
	}
}

func TestOnlyACheckTheUserAskedForSpeaksWhenThereIsNothingNew(t *testing.T) {
	t.Parallel()
	current := application.UpdateStatus{Outcome: application.UpdateCurrent}
	failed := application.UpdateStatus{Outcome: application.UpdateUnreachable, Fault: "no route to host"}
	cases := []struct {
		status application.UpdateStatus
		asked  bool
		want   []string
	}{
		{current, false, nil},
		{failed, false, nil},
		{current, true, []string{application.UpToDateText}},
		{failed, true, []string{application.UnreachableText}},
	}
	for _, c := range cases {
		r := newUpdateRig(nil, nil)
		r.finish(c.status, c.asked)
		if strings.Join(r.told, "|") != strings.Join(c.want, "|") || len(r.offered) != 0 {
			t.Errorf("%+v asked %v: told %q, offered %d", c.status, c.asked, r.told, len(r.offered))
		}
		if c.status.Fault != "" && len(r.log.lines) != 1 {
			t.Errorf("%+v: logged %q, want the fault once", c.status, r.log.lines)
		}
	}
}

func TestPressedButtonsMapToAnswers(t *testing.T) {
	t.Parallel()
	cases := map[int32]updateChoice{
		idDownload: chooseDownload, idYes: chooseDownload,
		idSkip: chooseSkip, idNo: chooseSkip,
		idLater: chooseLater, 2: chooseLater, 0: chooseLater,
	}
	for pressed, want := range cases {
		if got := choiceFor(pressed); got != want {
			t.Errorf("button %d: got %d, want %d", pressed, got, want)
		}
	}
}

// TestTheDialogRecordIsPackedAsMeasured reads every field back from the
// offsets the probe measured, so a slip in one fails here rather than as a
// dialog Windows refuses.
func TestTheDialogRecordIsPackedAsMeasured(t *testing.T) {
	t.Parallel()
	const parent = 0x1234
	d := newTaskDialog(parent, "WhatDay", offer.Headline(), offer.Detail(), promptButtons)
	le := binary.LittleEndian
	// text finds the string a field points at among those the record keeps
	// alive, so the test never turns a number back into a pointer.
	text := func(at []byte) string {
		for _, p := range d.strings {
			if uint64(uintptr(unsafe.Pointer(p))) == le.Uint64(at) {
				return windows.UTF16PtrToString(p)
			}
		}
		return "(not one of the dialog's strings)"
	}
	if got := le.Uint32(d.config[offsetSize:]); got != configSize || len(d.config) != configSize {
		t.Errorf("size %d, record %d bytes", got, len(d.config))
	}
	if got := le.Uint64(d.config[offsetParent:]); got != parent {
		t.Errorf("parent %#x", got)
	}
	if got := le.Uint32(d.config[offsetFlags:]); got != tdfAllowCancellation {
		t.Errorf("flags %#x", got)
	}
	if got := le.Uint64(d.config[offsetIcon:]); got != tdInformationIcon {
		t.Errorf("icon %#x", got)
	}
	for at, want := range map[int]string{offsetTitle: "WhatDay", offsetHeadline: "WhatDay 1.2.0 is available.", offsetContent: "You are running 1.1.0."} {
		if got := text(d.config[at:]); got != want {
			t.Errorf("offset %d: %q, want %q", at, got, want)
		}
	}
	if got := le.Uint32(d.config[offsetButtonCount:]); int(got) != len(promptButtons) {
		t.Errorf("button count %d", got)
	}
	if got := uintptr(le.Uint64(d.config[offsetButtons:])); got != uintptr(unsafe.Pointer(&d.buttons[0])) {
		t.Errorf("buttons at %#x", got)
	}
	if got := le.Uint32(d.config[offsetDefault:]); got != idDownload {
		t.Errorf("default button %d", got)
	}
	for i, b := range promptButtons {
		record := d.buttons[i*buttonSize:]
		if id := int32(le.Uint32(record)); id != b.id || text(record[offsetButtonLabel:]) != b.label {
			t.Errorf("button %d: id %d label %q", i, id, text(record[offsetButtonLabel:]))
		}
	}
}

func TestADialogWithNoButtonsPointsAtNone(t *testing.T) {
	t.Parallel()
	d := newTaskDialog(0, "WhatDay", "a", "b", nil)
	if binary.LittleEndian.Uint64(d.config[offsetButtons:]) != 0 {
		t.Fatal("a dialog with no buttons points at some")
	}
}
