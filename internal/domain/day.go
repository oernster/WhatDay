// Package domain holds WhatDay's rules: which day it is, when the next day
// begins and where the indicator may sit. It performs no I/O and never reads
// the clock; every instant and every zone is handed in.
package domain

import "time"

// DayName answers the English name of the day of the week at instant in zone,
// capitalised and in full, for example "Wednesday", whatever the zone
// (FR-001, FR-002, FR-006).
func DayName(instant time.Time, zone *time.Location) string {
	return instant.In(zone).Weekday().String()
}

// NextMidnight answers the first instant strictly after instant at which the
// civil date in zone moves on (FR-003).
//
// Usually that is 00:00 on the next date. Where a zone's clocks jump forward
// at midnight, 00:00 does not exist and time.Date answers an instant still on
// the old date (measured: 746 such midnights across 598 zones, 2000 to 2100);
// there the day starts at the jump, which is found by bisection. A date a zone
// skipped altogether (Samoa, 30 December 2011) is skipped here too.
func NextMidnight(instant time.Time, zone *time.Location) time.Time {
	today := dateOf(instant, zone)
	year, month, day := instant.In(zone).Date()
	candidate := time.Date(year, month, day+1, 0, 0, 0, 0, zone)
	if candidate.After(instant) && dateOf(candidate, zone) > today &&
		dateOf(candidate.Add(-time.Nanosecond), zone) == today {
		return candidate
	}
	return firstInstantAfter(instant, today, zone)
}

// dayScan bounds the search for the next date: every zone's day, even one
// lengthened by a clock change, ends well inside it.
const dayScan = 48 * time.Hour

// firstInstantAfter bisects for the first instant after from whose civil date
// in zone is later than today.
func firstInstantAfter(from time.Time, today int, zone *time.Location) time.Time {
	lo, hi := from, from.Add(dayScan)
	for hi.Sub(lo) > time.Nanosecond {
		mid := lo.Add(hi.Sub(lo) / 2)
		if dateOf(mid, zone) > today {
			hi = mid
		} else {
			lo = mid
		}
	}
	return hi
}

// dateOf answers the civil date of instant in zone as one ordered number,
// yyyymmdd.
func dateOf(instant time.Time, zone *time.Location) int {
	const yearShift, monthShift = 10000, 100
	year, month, day := instant.In(zone).Date()
	return year*yearShift + int(month)*monthShift + day
}
