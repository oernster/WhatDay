package application_test

import (
	"testing"

	"github.com/oernster/WhatDay/internal/application"
)

func TestMenuModel(t *testing.T) {
	t.Parallel()
	r := newRig(t, someInstant, &fakeStore{})
	r.ind.Start()
	r.ind.ChooseColour("Amber")
	want := []application.MenuItem{
		{Kind: application.MenuAbout, Label: "About WhatDay"},
		{Kind: application.MenuSeparator},
		{Kind: application.MenuColour, Label: "Red"},
		{Kind: application.MenuColour, Label: "Amber", Checked: true},
		{Kind: application.MenuColour, Label: "Green"},
		{Kind: application.MenuColour, Label: "Blue"},
		{Kind: application.MenuColour, Label: "Purple"},
		{Kind: application.MenuColour, Label: "Neutral"},
	}
	got := r.ind.Menu()
	if len(got) != len(want) {
		t.Fatalf("got %d items, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestAboutContent(t *testing.T) {
	t.Parallel()
	want := application.About{Name: "WhatDay", Version: "1.2.3", Author: "Oliver Ernster", Licence: "GPL-3.0"}
	if got := application.NewAbout("1.2.3"); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
