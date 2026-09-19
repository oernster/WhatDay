package application_test

import (
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
)

var someInstant = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

func TestDefaults(t *testing.T) {
	t.Parallel()
	r := newRig(t, someInstant, &fakeStore{})
	r.ind.Start()
	if got := r.view.lastColour(); got != domain.DefaultColourName {
		t.Fatalf("colour: got %q, want %q", got, domain.DefaultColourName)
	}
	if got := r.log.about("settings"); len(got) != 0 {
		t.Fatalf("absence is not a fault, yet logged %q", r.log.lines)
	}
}

func TestSavedColourIsUsed(t *testing.T) {
	t.Parallel()
	store := &fakeStore{saved: application.Settings{ColourName: "Green"}, found: true}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	if got := r.view.lastColour(); got != "Green" {
		t.Fatalf("got %q, want Green", got)
	}
}

func TestUnreadableSettingsGiveDefaultsAndLog(t *testing.T) {
	t.Parallel()
	r := newRig(t, someInstant, &fakeStore{loadErr: errDisk})
	r.ind.Start()
	if got := r.view.lastColour(); got != domain.DefaultColourName {
		t.Fatalf("colour: got %q, want %q", got, domain.DefaultColourName)
	}
	if got := r.log.about(errDisk.Error()); len(got) != 1 {
		t.Fatalf("log: got %q, want one line naming the error", r.log.lines)
	}
}

func TestUnknownSavedColourGivesDefaultAndLog(t *testing.T) {
	t.Parallel()
	store := &fakeStore{saved: application.Settings{ColourName: "Magenta"}, found: true}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	if got := r.view.lastColour(); got != domain.DefaultColourName {
		t.Fatalf("colour: got %q, want %q", got, domain.DefaultColourName)
	}
	if got := r.log.about("Magenta"); len(got) != 1 {
		t.Fatalf("log: got %q, want one line naming Magenta", r.log.lines)
	}
	menu := r.ind.Menu()
	if !menu[2].Checked || menu[2].Label != domain.DefaultColourName {
		t.Fatalf("menu should check the default colour, got %+v", menu[2])
	}
}

func TestMovedToSavesPosition(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	r.ind.MovedTo(domain.Point{X: 100, Y: 200})
	want := application.Settings{ColourName: domain.DefaultColourName, Position: domain.Point{X: 100, Y: 200}, HasPosition: true}
	if store.saved != want {
		t.Fatalf("saved %+v, want %+v", store.saved, want)
	}
}

func TestPlacementUsesSavedPosition(t *testing.T) {
	t.Parallel()
	work := domain.Rect{Left: 0, Top: 0, Right: 3440, Bottom: 1392}
	monitors := []domain.Monitor{{Bounds: domain.Rect{Left: 0, Top: 0, Right: 3440, Bottom: 1440}, Work: work}}
	size := domain.Size{W: 177, H: 48}
	store := &fakeStore{saved: application.Settings{ColourName: "Red", Position: domain.Point{X: 100, Y: 200}, HasPosition: true}, found: true}
	r := newRig(t, someInstant, store)
	r.ind.Start()
	if got, want := r.ind.Placement(size, monitors, work), (domain.Point{X: 100, Y: 200}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPlacementWithoutSavedPositionIsDefault(t *testing.T) {
	t.Parallel()
	work := domain.Rect{Left: 0, Top: 0, Right: 3440, Bottom: 1392}
	monitors := []domain.Monitor{{Bounds: domain.Rect{Left: 0, Top: 0, Right: 3440, Bottom: 1440}, Work: work}}
	size := domain.Size{W: 177, H: 48}
	r := newRig(t, someInstant, &fakeStore{})
	r.ind.Start()
	if got, want := r.ind.Placement(size, monitors, work), (domain.Point{X: 3263, Y: 1344}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
