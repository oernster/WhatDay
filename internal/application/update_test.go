package application_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/oernster/WhatDay/internal/application"
)

// fakeSource answers one release; err makes it fail instead.
type fakeSource struct {
	info application.ReleaseInfo
	err  error
}

func (s *fakeSource) LatestRelease() (application.ReleaseInfo, error) { return s.info, s.err }

const (
	running  = "1.1.0"
	release  = "v1.2.0"
	pageURL  = "https://github.com/oernster/WhatDay/releases/tag/v1.2.0"
	setupURL = "https://github.com/oernster/WhatDay/releases/download/v1.2.0/WhatDaySetup.exe"
)

func published(tag string, assets ...application.ReleaseAsset) application.ReleaseInfo {
	return application.ReleaseInfo{Version: tag, PageURL: pageURL, Assets: assets}
}

var setupAsset = application.ReleaseAsset{Name: "WhatDaySetup.exe", DownloadURL: setupURL}

func check(info application.ReleaseInfo, skipped string) application.UpdateStatus {
	return application.NewUpdateChecker(&fakeSource{info: info}, running).Check(skipped)
}

func TestANewerReleaseIsOfferedWithItsSetupProgram(t *testing.T) {
	t.Parallel()
	got := check(published(release, setupAsset), "")
	want := application.UpdateStatus{Outcome: application.UpdateAvailable, Current: running, Latest: "1.2.0", Tag: release, DownloadURL: setupURL}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAReleaseWithoutASetupProgramOffersItsPage(t *testing.T) {
	t.Parallel()
	notSetup := application.ReleaseAsset{Name: "notes.txt", DownloadURL: "https://example.com/notes.txt"}
	if got := check(published(release, notSetup), "").DownloadURL; got != pageURL {
		t.Fatalf("got %q, want the page %q", got, pageURL)
	}
	upper := application.ReleaseAsset{Name: "WHATDAYSETUP.EXE", DownloadURL: setupURL}
	if got := check(published(release, upper), "").DownloadURL; got != setupURL {
		t.Fatalf("an upper-case .EXE was not matched: %q", got)
	}
}

func TestASkippedReleaseIsSeenButNotOffered(t *testing.T) {
	t.Parallel()
	if got := check(published(release), release).Outcome; got != application.UpdateSkipped {
		t.Fatalf("got outcome %d, want skipped", got)
	}
	if got := check(published(release), "v1.1.5").Outcome; got != application.UpdateAvailable {
		t.Fatalf("a skip of another release suppressed this one: %d", got)
	}
}

func TestVersionsCompareNumberByNumber(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tag  string
		want application.UpdateOutcome
	}{
		{"v1.2.0", application.UpdateAvailable},
		{"V1.2.0", application.UpdateAvailable},
		{" 1.2.0 ", application.UpdateAvailable},
		{"2.0.0", application.UpdateAvailable},
		{"1.10.0", application.UpdateAvailable},
		{"1.1.0.1", application.UpdateAvailable},
		{"1.1.0", application.UpdateCurrent},
		{"v1.1.0", application.UpdateCurrent},
		{"1.0.9", application.UpdateCurrent},
		{"1.1", application.UpdateCurrent},
		{"1.2.0-beta", application.UpdateUnreachable},
		{"", application.UpdateUnreachable},
		{"latest", application.UpdateUnreachable},
	}
	for _, c := range cases {
		if got := check(published(c.tag), "").Outcome; got != c.want {
			t.Errorf("tag %q: got outcome %d, want %d", c.tag, got, c.want)
		}
	}
}

func TestAnUnreachableSourceSaysWhy(t *testing.T) {
	t.Parallel()
	got := application.NewUpdateChecker(&fakeSource{err: errors.New("no route to host")}, running).Check("")
	if got.Outcome != application.UpdateUnreachable || got.Current != running || got.Fault != "no route to host" {
		t.Fatalf("got %+v", got)
	}
	if got.Latest != "" || got.DownloadURL != "" {
		t.Fatalf("an unreachable check offered something: %+v", got)
	}
}

// A build run from source is 0.0.0-dev; it can compare with nothing, so it
// must never claim to be the latest version.
func TestARunningVersionThatIsNotNumbersCannotBeCompared(t *testing.T) {
	t.Parallel()
	got := application.NewUpdateChecker(&fakeSource{info: published(release)}, "0.0.0-dev").Check("")
	if got.Outcome != application.UpdateUnreachable || !strings.Contains(got.Fault, "0.0.0-dev") {
		t.Fatalf("got %+v", got)
	}
}

func TestRepliesFollowWhoAsked(t *testing.T) {
	t.Parallel()
	cases := []struct {
		outcome application.UpdateOutcome
		asked   bool
		want    application.UpdateReply
	}{
		{application.UpdateAvailable, false, application.ReplyPrompt},
		{application.UpdateAvailable, true, application.ReplyPrompt},
		{application.UpdateSkipped, false, application.ReplyNothing},
		{application.UpdateSkipped, true, application.ReplyPrompt},
		{application.UpdateCurrent, false, application.ReplyNothing},
		{application.UpdateCurrent, true, application.ReplyUpToDate},
		{application.UpdateUnreachable, false, application.ReplyNothing},
		{application.UpdateUnreachable, true, application.ReplyUnreachable},
	}
	for _, c := range cases {
		if got := (application.UpdateStatus{Outcome: c.outcome}).Reply(c.asked); got != c.want {
			t.Errorf("outcome %d, asked %v: got reply %d, want %d", c.outcome, c.asked, got, c.want)
		}
	}
}

func TestThePromptNamesBothVersions(t *testing.T) {
	t.Parallel()
	status := check(published(release), "")
	if got := status.Headline(); got != "WhatDay 1.2.0 is available." {
		t.Errorf("headline: %q", got)
	}
	if got := status.Detail(); got != "You are running 1.1.0." {
		t.Errorf("detail: %q", got)
	}
}

func TestASkipIsRememberedAcrossRuns(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	first := newRig(t, someInstant, store)
	first.ind.Start()
	first.ind.SkipVersion(release)
	if store.saved.SkippedVersion != release {
		t.Fatalf("saved %+v", store.saved)
	}
	second := newRig(t, someInstant, store)
	second.ind.Start()
	if got := second.ind.SkippedVersion(); got != release {
		t.Fatalf("the next run skips %q, want %q", got, release)
	}
	if len(first.log.about("skipping")) != 1 {
		t.Fatalf("log: %q", first.log.lines)
	}
}

func TestAnUnsavedSkipHoldsForThisRun(t *testing.T) {
	t.Parallel()
	r := newRig(t, someInstant, &fakeStore{saveErr: errDisk})
	r.ind.Start()
	r.ind.SkipVersion(release)
	if got := r.ind.SkippedVersion(); got != release {
		t.Fatalf("got %q", got)
	}
	if len(r.log.about("not saved")) != 1 {
		t.Fatalf("log: %q", r.log.lines)
	}
}
