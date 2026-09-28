// Package application holds WhatDay's use cases. It depends on the domain
// and on the ports declared here; infrastructure implements the ports and the
// composition root wires them.
package application

import (
	"time"

	"github.com/oernster/WhatDay/internal/domain"
)

// Clock answers the current instant from the system clock.
type Clock interface {
	Now() time.Time
}

// Zones answers the timezone Windows is set to now (FR-002, Amendment 4). On
// failure it still answers a usable zone, the best it has, beside the error.
type Zones interface {
	Current() (*time.Location, error)
}

// View is the indicator as the use cases see it.
type View interface {
	// ShowDay paints the day name.
	ShowDay(name string)
	// SetColour paints the day name in colour from now on.
	SetColour(colour domain.Colour)
}

// Scheduler wakes the application once at an absolute instant by calling
// Indicator.Refresh. Arming it again replaces the previous wake-up.
type Scheduler interface {
	WakeAt(instant time.Time)
}

// Settings is what WhatDay remembers between runs (FR-050). SkippedVersion is
// the release tag the user chose to skip, exactly as it was released (FR-039).
type Settings struct {
	ColourName     string
	Position       domain.Point
	HasPosition    bool
	SkippedVersion string
}

// SettingsStore loads and saves Settings. Load answers found false with no
// error when nothing has been saved yet: absence is an answer, not a fault.
type SettingsStore interface {
	Load() (settings Settings, found bool, err error)
	Save(settings Settings) error
}

// ReleaseAsset names one file attached to a release.
type ReleaseAsset struct {
	Name        string
	DownloadURL string
}

// ReleaseInfo is the latest published release. Version is the tag exactly as
// released, a leading "v" included.
type ReleaseInfo struct {
	Version string
	PageURL string
	Assets  []ReleaseAsset
}

// ReleaseSource answers the latest published release (FR-037, FR-038). Only
// a published release that is neither a draft nor a prerelease is ever
// answered, so a tag pushed mid-development can never prompt. It is the one
// port that reaches the network (NFR-PRIV-001, Amendment 11).
type ReleaseSource interface {
	LatestRelease() (ReleaseInfo, error)
}

// Log records what happened, for reading after the fact.
type Log interface {
	Printf(format string, args ...any)
}
