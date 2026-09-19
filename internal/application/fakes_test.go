package application_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
	"github.com/oernster/WhatDay/internal/testsupport"
)

var errDisk = errors.New("disk refused")

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type fakeView struct {
	days    []string
	colours []domain.Colour
}

func (v *fakeView) ShowDay(name string)            { v.days = append(v.days, name) }
func (v *fakeView) SetColour(colour domain.Colour) { v.colours = append(v.colours, colour) }

func (v *fakeView) lastDay() string { return v.days[len(v.days)-1] }

func (v *fakeView) lastColour() string { return v.colours[len(v.colours)-1].Name }

type fakeScheduler struct{ wakes []time.Time }

func (s *fakeScheduler) WakeAt(instant time.Time) { s.wakes = append(s.wakes, instant) }

func (s *fakeScheduler) last() time.Time { return s.wakes[len(s.wakes)-1] }

// fakeStore holds settings in memory; loadErr and saveErr inject failures.
type fakeStore struct {
	saved   application.Settings
	found   bool
	loadErr error
	saveErr error
	saves   int
}

func (s *fakeStore) Load() (application.Settings, bool, error) {
	if s.loadErr != nil {
		return application.Settings{}, false, s.loadErr
	}
	return s.saved, s.found, nil
}

func (s *fakeStore) Save(settings application.Settings) error {
	s.saves++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved, s.found = settings, true
	return nil
}

type fakeLog struct{ lines []string }

func (l *fakeLog) Printf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

// about answers the lines that mention word.
func (l *fakeLog) about(word string) []string {
	var found []string
	for _, line := range l.lines {
		if strings.Contains(line, word) {
			found = append(found, line)
		}
	}
	return found
}

// checkedColour answers the colour the menu has checked; empty when none is.
// It finds the entry by kind, so a new menu entry cannot shift what it reads.
func checkedColour(menu []application.MenuItem) string {
	for _, item := range menu {
		if item.Kind == application.MenuColour && item.Checked {
			return item.Label
		}
	}
	return ""
}

// fakeZones answers zone and err, as a Zones port does: a usable zone always,
// an error beside it when the real zone could not be read.
type fakeZones struct {
	zone *time.Location
	err  error
}

func (z *fakeZones) Current() (*time.Location, error) { return z.zone, z.err }

// rig is an Indicator wired to fakes.
type rig struct {
	ind       *application.Indicator
	clock     *fakeClock
	zones     *fakeZones
	view      *fakeView
	scheduler *fakeScheduler
	store     *fakeStore
	log       *fakeLog
}

var london = testsupport.London

func newRig(t *testing.T, now time.Time, store *fakeStore) rig {
	t.Helper()
	r := rig{clock: &fakeClock{now: now}, zones: &fakeZones{zone: london(t)}, view: &fakeView{}, scheduler: &fakeScheduler{}, store: store, log: &fakeLog{}}
	r.ind = application.NewIndicator(r.clock, r.zones, r.view, r.scheduler, r.store, r.log)
	return r
}
