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
		{Kind: application.MenuSeparator},
		{Kind: application.MenuQuit, Label: "Quit WhatDay"},
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
	about := application.NewAbout()
	if about.Title != "About WhatDay" {
		t.Errorf("title: got %q", about.Title)
	}
	want := "WhatDay\n" +
		"© 2026 Oliver Ernster\n" +
		"\n" +
		"Built with:\n" +
		"Go, BSD 3-Clause, © 2009 The Go Authors\n" +
		"golang.org/x/sys, BSD 3-Clause, © 2009 The Go Authors\n" +
		"IANA Time Zone Database, public domain"
	if got := about.Text(); got != want {
		t.Errorf("text:\ngot  %q\nwant %q", got, want)
	}
}

func TestAboutCreditsAreACopy(t *testing.T) {
	t.Parallel()
	first := application.NewAbout()
	first.Credits[0].Work = "changed"
	if application.NewAbout().Credits[0].Work != "Go" {
		t.Fatal("changing a returned About changed the credits")
	}
}
