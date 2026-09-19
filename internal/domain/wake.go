package domain

import "time"

// WakeDelay answers how long to wait before looking at the clock again on the
// way to target; due reports that target has already been reached. The wait
// never exceeds maxGap. A timer that drifts from the wall clock (or pauses
// while the machine sleeps) is corrected at the next look rather than trusted
// for hours (FR-003).
func WakeDelay(now, target time.Time, maxGap time.Duration) (wait time.Duration, due bool) {
	if !now.Before(target) {
		return 0, true
	}
	return min(target.Sub(now), maxGap), false
}
