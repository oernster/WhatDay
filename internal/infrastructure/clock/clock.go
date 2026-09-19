// Package clock reads the system clock for the application's Clock port.
package clock

import "time"

// System is the Windows system clock.
type System struct{}

// Now answers the current instant.
func (System) Now() time.Time { return time.Now() }
