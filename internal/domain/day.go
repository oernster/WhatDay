// Package domain holds WhatDay's rules: which day it is, when the next day
// begins and where the indicator may sit. It performs no I/O and never reads
// the clock; every instant and every zone is handed in.
package domain

import "time"

// ZoneName is the IANA zone WhatDay tells the day in (FR-002).
const ZoneName = "Europe/London"

// DayName answers the English name of the day of the week at instant in zone,
// capitalised and in full, for example "Wednesday" (FR-001, FR-002, FR-006).
func DayName(instant time.Time, zone *time.Location) string {
	return instant.In(zone).Weekday().String()
}

// NextMidnight answers the first instant strictly after instant at which the
// civil date in zone changes (FR-003). Days of 23 and 25 hours around a
// daylight saving change are handled by building the next civil date rather
// than adding a fixed duration.
func NextMidnight(instant time.Time, zone *time.Location) time.Time {
	year, month, day := instant.In(zone).Date()
	return time.Date(year, month, day+1, 0, 0, 0, 0, zone)
}
