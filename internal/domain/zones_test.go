package domain_test

import (
	"archive/zip"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/domain"
)

// The years every zone is walked across (FR-002, FR-003, Amendment 4).
const (
	zoneWalkFrom = 2000
	zoneWalkTo   = 2100
)

// skippedDates are the civil dates a zone left out altogether inside the walk,
// each checked against the tz database's own record: Samoa and Tokelau moved
// across the date line, going from Thursday 29 to Saturday 31 December 2011.
var skippedDates = map[string]bool{
	"Pacific/Apia 2011-12-30":    true,
	"Pacific/Fakaofo 2011-12-30": true,
}

// civil answers a time's wall date as an ordered yyyymmdd number, so dates
// compare without building a time on a date that may not exist.
func civil(t time.Time) int {
	const yearShift, monthShift = 10000, 100
	return t.Year()*yearShift + int(t.Month())*monthShift + t.Day()
}

// zoneNames lists every zone in the tz data this Go ships and embeds.
func zoneNames(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Fatalf("go env GOROOT: %v", err)
	}
	archive := filepath.Join(strings.TrimSpace(string(out)), "lib", "time", "zoneinfo.zip")
	reader, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatalf("opening %s: %v", archive, err)
	}
	defer func() { _ = reader.Close() }()
	var names []string
	for _, f := range reader.File {
		if !f.FileInfo().IsDir() {
			names = append(names, f.Name)
		}
	}
	const atLeast = 400 // the database holds about 600; far fewer means a bad read
	if len(names) < atLeast {
		t.Fatalf("found %d zones in %s, want at least %d", len(names), archive, atLeast)
	}
	return names
}

func TestEveryZoneMidnightChain(t *testing.T) {
	t.Parallel()
	skipped := map[string]bool{}
	for _, name := range zoneNames(t) {
		zone, err := time.LoadLocation(name)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		at := time.Date(zoneWalkFrom, 1, 1, 12, 0, 0, 0, zone)
		end := time.Date(zoneWalkTo, 1, 1, 0, 0, 0, 0, zone)
		for at.Before(end) {
			next := domain.NextMidnight(at, zone)
			today, tomorrow := at.In(zone), next.In(zone)
			expected := time.Date(today.Year(), today.Month(), today.Day()+1, 12, 0, 0, 0, time.UTC)
			switch {
			case !next.After(at):
				t.Fatalf("%s after %s: got %s, not later", name, today, tomorrow)
			case domain.DayName(next.Add(-time.Nanosecond), zone) != today.Weekday().String():
				t.Fatalf("%s after %s: got %s; the day had already changed before it", name, today, tomorrow)
			case civil(tomorrow) == civil(expected):
			case civil(tomorrow) > civil(expected):
				// A date this zone never had; which ones is checked below.
				skipped[name+" "+expected.Format("2006-01-02")] = true
			default:
				t.Fatalf("%s after %s: got %s, want %s", name, today, tomorrow, expected.Format("2006-01-02"))
			}
			at = next
		}
	}
	for date := range skipped {
		if !skippedDates[date] {
			t.Errorf("unexpected skipped date %s", date)
		}
	}
	for date := range skippedDates {
		if !skipped[date] {
			t.Errorf("expected %s to be skipped; it was not", date)
		}
	}
}

func TestMidnightInAGapStartsAtTheJump(t *testing.T) {
	t.Parallel()
	// America/Sao_Paulo moved its clocks from 00:00 to 01:00 on 2018-11-04,
	// so that Sunday began at 01:00 local, 03:00 UTC.
	zone, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	saturday := time.Date(2018, 11, 3, 20, 0, 0, 0, zone)
	want := time.Date(2018, 11, 4, 3, 0, 0, 0, time.UTC)
	if got := domain.NextMidnight(saturday, zone); !got.Equal(want) {
		t.Fatalf("got %s, want %s", got.UTC(), want)
	}
	if got := domain.DayName(want, zone); got != "Sunday" {
		t.Fatalf("day at the jump: got %s, want Sunday", got)
	}
}

func TestSamoaSkipsFriday(t *testing.T) {
	t.Parallel()
	zone, err := time.LoadLocation("Pacific/Apia")
	if err != nil {
		t.Fatal(err)
	}
	thursday := time.Date(2011, 12, 29, 12, 0, 0, 0, zone)
	next := domain.NextMidnight(thursday, zone)
	if got := domain.DayName(next, zone); got != "Saturday" {
		t.Fatalf("after Thursday 29 December 2011 in Samoa: got %s, want Saturday", got)
	}
}
