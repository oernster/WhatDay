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

// Settings is what WhatDay remembers between runs (FR-050).
type Settings struct {
	ColourName  string
	Position    domain.Point
	HasPosition bool
}

// SettingsStore loads and saves Settings. Load answers found false with no
// error when nothing has been saved yet: absence is an answer, not a fault.
type SettingsStore interface {
	Load() (settings Settings, found bool, err error)
	Save(settings Settings) error
}

// Log records what happened, for reading after the fact.
type Log interface {
	Printf(format string, args ...any)
}
