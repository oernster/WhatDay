package ui

import (
	"errors"
	"fmt"
	"testing"

	"github.com/oernster/WhatDay/internal/application"
)

// These tests never open a browser: the window's opener is replaced, while
// the real one is only ever handed an address it refuses.

type recordingLog struct{ lines []string }

func (l *recordingLog) Printf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

// donateRig is a window whose opener and alert are recorded.
type donateRig struct {
	window *Window
	log    *recordingLog
	asked  []string
	said   []string
}

func newDonateRig(refuse error) *donateRig {
	r := &donateRig{log: &recordingLog{}}
	r.window = &Window{
		log: r.log,
		open: func(address string) error {
			r.asked = append(r.asked, address)
			return refuse
		},
		alert: func(text string) { r.said = append(r.said, text) },
	}
	return r
}

func TestDonateAsksForWhatDaysAddressOnly(t *testing.T) {
	t.Parallel()
	r := newDonateRig(nil)
	r.window.donate()
	if len(r.asked) != 1 || r.asked[0] != application.DonateURL {
		t.Fatalf("asked for %q, want only %q", r.asked, application.DonateURL)
	}
	if len(r.said) != 0 || len(r.log.lines) != 0 {
		t.Fatalf("a successful open said %q and logged %q", r.said, r.log.lines)
	}
}

func TestDonateRefusedSaysSo(t *testing.T) {
	t.Parallel()
	r := newDonateRig(errors.New("no browser"))
	r.window.donate()
	if len(r.said) != 1 || r.said[0] != donateRefused {
		t.Fatalf("said %q, want %q", r.said, donateRefused)
	}
	if len(r.log.lines) != 1 {
		t.Fatalf("logged %q, want one line", r.log.lines)
	}
}

func TestOpenExternalRefusesAnythingButHTTPS(t *testing.T) {
	t.Parallel()
	for _, address := range []string{"http://example.com", "file:///C:/Windows", "calc.exe", ""} {
		if err := openExternal(address); err == nil {
			t.Errorf("%q was handed to the desktop", address)
		}
	}
}

func TestNewWindowOpensThroughTheDesktop(t *testing.T) {
	t.Parallel()
	w := NewWindow(&recordingLog{}, nil)
	if w.open == nil || w.alert == nil {
		t.Fatal("a new window cannot open the support page or say it failed")
	}
}
