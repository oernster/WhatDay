package application_test

import (
	"errors"
	"fmt"
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

// rig is an Indicator wired to fakes.
type rig struct {
	ind       *application.Indicator
	clock     *fakeClock
	view      *fakeView
	scheduler *fakeScheduler
	store     *fakeStore
	log       *fakeLog
}

var london = testsupport.London

func newRig(t *testing.T, now time.Time, store *fakeStore) rig {
	t.Helper()
	zone := london(t)
	r := rig{clock: &fakeClock{now: now}, view: &fakeView{}, scheduler: &fakeScheduler{}, store: store, log: &fakeLog{}}
	r.ind = application.NewIndicator(r.clock, zone, r.view, r.scheduler, r.store, r.log)
	return r
}
