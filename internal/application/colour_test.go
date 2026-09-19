package application_test

import (
	"strings"
	"testing"
)

func TestChooseColourRepaints(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	r.ind.ChooseColour("Blue")
	if got := r.view.lastColour(); got != "Blue" {
		t.Fatalf("painted %q, want Blue", got)
	}
	if store.saved.ColourName != "Blue" {
		t.Fatalf("saved %q, want Blue", store.saved.ColourName)
	}
}

func TestChooseUnknownColourIsIgnored(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	painted := len(r.view.colours)
	r.ind.ChooseColour("Magenta")
	if len(r.view.colours) != painted || store.saves != 0 {
		t.Fatalf("an unknown colour changed something: %d paints, %d saves", len(r.view.colours)-painted, store.saves)
	}
	if len(r.log.lines) != 1 || !strings.Contains(r.log.lines[0], "Magenta") {
		t.Fatalf("log: got %q, want one line naming Magenta", r.log.lines)
	}
}

func TestUnwritableKeepsSessionValue(t *testing.T) {
	t.Parallel()
	store := &fakeStore{saveErr: errDisk}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	r.ind.ChooseColour("Purple")
	if got := r.view.lastColour(); got != "Purple" {
		t.Fatalf("painted %q, want Purple despite the failed save", got)
	}
	if !r.ind.Menu()[6].Checked {
		t.Fatal("menu should still check Purple for this run")
	}
	if len(r.log.lines) != 1 || !strings.Contains(r.log.lines[0], errDisk.Error()) {
		t.Fatalf("log: got %q, want one line naming the error", r.log.lines)
	}
}
