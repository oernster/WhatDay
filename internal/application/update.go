package application

import (
	"fmt"
	"strconv"
	"strings"
)

// Update wording, decided here rather than in Win32 code (FR-037, FR-038).
const (
	UpdatesLabel    = "Check for updates"
	DownloadLabel   = "Download"
	SkipLabel       = "Skip This Version"
	LaterLabel      = "Later"
	UpToDateText    = "You are running the latest version."
	UnreachableText = "The update check could not reach GitHub. Please try again later."
	// FallbackKey explains the plain message box's buttons, which cannot be
	// renamed, on a Windows that cannot show the three named ones.
	FallbackKey = "Yes downloads it. No skips this version. Cancel asks again later."
)

// setupSuffix picks the setup program from a release's files; WhatDay is a
// Windows application only, so it is the one file it ever offers.
const setupSuffix = ".exe"

// tagPrefix is the optional prefix on a release tag, as in v1.2.0.
const tagPrefix = "v"

// versionSeparator divides a version's numbers.
const versionSeparator = "."

// UpdateOutcome says what one update check found.
type UpdateOutcome int

// The outcomes of an update check.
const (
	UpdateUnreachable UpdateOutcome = iota
	UpdateCurrent
	UpdateAvailable
	UpdateSkipped
)

// UpdateStatus is what one update check found. Latest is the release's
// version for display; Tag is the same release exactly as tagged, which is
// what a skip remembers. DownloadURL is the setup program where the release
// carries one, else the release page. Fault says why a check was unreachable,
// for the log.
type UpdateStatus struct {
	Outcome     UpdateOutcome
	Current     string
	Latest      string
	Tag         string
	DownloadURL string
	Fault       string
}

// UpdateReply is what the user is shown for a status.
type UpdateReply int

// The replies to an update check.
const (
	ReplyNothing UpdateReply = iota
	ReplyPrompt
	ReplyUpToDate
	ReplyUnreachable
)

// Reply answers what to show (FR-037, FR-038). A check the user asked for
// ignores a skip and reports every outcome; an automatic one speaks only to
// offer a release the user has not skipped and is silent otherwise.
func (s UpdateStatus) Reply(asked bool) UpdateReply {
	switch {
	case s.Outcome == UpdateAvailable, asked && s.Outcome == UpdateSkipped:
		return ReplyPrompt
	case !asked:
		return ReplyNothing
	case s.Outcome == UpdateCurrent:
		return ReplyUpToDate
	}
	return ReplyUnreachable
}

// Headline is the prompt's first line.
func (s UpdateStatus) Headline() string {
	return fmt.Sprintf("%s %s is available.", ProductName, s.Latest)
}

// Detail is the prompt's second line.
func (s UpdateStatus) Detail() string {
	return fmt.Sprintf("You are running %s.", s.Current)
}

// UpdateChecker compares the running version with the latest release.
type UpdateChecker struct {
	source  ReleaseSource
	current string
}

// NewUpdateChecker checks source against current, the running version.
func NewUpdateChecker(source ReleaseSource, current string) *UpdateChecker {
	return &UpdateChecker{source: source, current: current}
}

// Check asks the source for the latest release. A release equal to skipped
// is reported as skipped rather than available, so the same version never
// prompts twice unasked (FR-039). A source that fails is unreachable; so is a
// version that is not dotted numbers on either side. WhatDay cannot tell, so
// it never claims an update or the latest version.
func (c *UpdateChecker) Check(skipped string) UpdateStatus {
	status := UpdateStatus{Outcome: UpdateUnreachable, Current: c.current}
	info, err := c.source.LatestRelease()
	if err != nil {
		status.Fault = err.Error()
		return status
	}
	latest, latestOK := versionParts(info.Version)
	current, currentOK := versionParts(c.current)
	if !latestOK || !currentOK {
		status.Fault = fmt.Sprintf("cannot compare release %q with running %q", info.Version, c.current)
		return status
	}
	status.Latest = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(info.Version)), tagPrefix)
	status.Tag = info.Version
	status.DownloadURL = downloadFor(info)
	switch {
	case !newer(latest, current):
		status.Outcome = UpdateCurrent
	case info.Version == skipped:
		status.Outcome = UpdateSkipped
	default:
		status.Outcome = UpdateAvailable
	}
	return status
}

// downloadFor answers the release's setup program, else its page.
func downloadFor(info ReleaseInfo) string {
	for _, asset := range info.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), setupSuffix) {
			return asset.DownloadURL
		}
	}
	return info.PageURL
}

// newer reports whether latest is strictly later than current, number by
// number; with every shared number equal, the longer version is the later.
func newer(latest, current []int) bool {
	for i := 0; i < len(latest) && i < len(current); i++ {
		if latest[i] != current[i] {
			return latest[i] > current[i]
		}
	}
	return len(latest) > len(current)
}

// versionParts reads a dotted version with an optional leading v; false when
// any part is not a number.
func versionParts(version string) ([]int, bool) {
	text := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(version)), tagPrefix)
	fields := strings.Split(text, versionSeparator)
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, true
}
