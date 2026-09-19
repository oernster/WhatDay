package application

import (
	"time"

	"github.com/oernster/WhatDay/internal/domain"
)

// Indicator is the one use-case object behind the indicator and its menu.
type Indicator struct {
	clock     Clock
	zone      *time.Location
	view      View
	scheduler Scheduler
	store     SettingsStore
	log       Log
	settings  Settings
}

// NewIndicator wires the use cases to their ports. zone is Europe/London,
// loaded by the composition root from the embedded zone data.
func NewIndicator(clock Clock, zone *time.Location, view View, scheduler Scheduler, store SettingsStore, log Log) *Indicator {
	return &Indicator{clock: clock, zone: zone, view: view, scheduler: scheduler, store: store, log: log}
}

// Start loads the settings, paints the colour and the day, then arms the
// wake-up for the next London midnight (FR-001, FR-041, FR-052, FR-053).
func (ind *Indicator) Start() {
	ind.settings = ind.loadSettings()
	colour, _ := domain.ColourNamed(ind.settings.ColourName)
	ind.settings.ColourName = colour.Name
	ind.view.SetColour(colour)
	ind.Refresh()
}

func (ind *Indicator) loadSettings() Settings {
	defaults := Settings{ColourName: domain.DefaultColourName}
	loaded, found, err := ind.store.Load()
	switch {
	case err != nil:
		ind.log.Printf("settings unreadable, using defaults: %v", err)
		return defaults
	case !found:
		return defaults
	}
	if _, known := domain.ColourNamed(loaded.ColourName); !known {
		ind.log.Printf("saved colour %q is not in the palette, using %s", loaded.ColourName, domain.DefaultColourName)
	}
	return loaded
}

// Refresh paints the London day of the current instant and re-arms the
// wake-up for the next London midnight. It is the single response to the
// midnight wake-up, to resume from sleep and to a clock change (FR-003,
// FR-004, FR-005). A wake-up that fires early re-arms the same midnight, so
// it cannot show the wrong day.
func (ind *Indicator) Refresh() {
	now := ind.clock.Now()
	ind.view.ShowDay(domain.DayName(now, ind.zone))
	ind.scheduler.WakeAt(domain.NextMidnight(now, ind.zone))
}

// ChooseColour repaints the day name in the named colour and saves the
// choice (FR-033, FR-050). A save that fails keeps the choice for this run
// and says so in the log (FR-054).
func (ind *Indicator) ChooseColour(name string) {
	colour, known := domain.ColourNamed(name)
	if !known {
		ind.log.Printf("ignored unknown colour %q", name)
		return
	}
	ind.settings.ColourName = colour.Name
	ind.view.SetColour(colour)
	ind.save()
}

// Placement answers where the indicator goes for its size on the monitors
// attached now (FR-021, FR-022).
func (ind *Indicator) Placement(size domain.Size, monitors []domain.Monitor, primaryWork domain.Rect) domain.Point {
	return domain.Place(ind.settings.Position, ind.settings.HasPosition, size, monitors, primaryWork)
}

// MovedTo saves the position a drag ended at (FR-050).
func (ind *Indicator) MovedTo(position domain.Point) {
	ind.settings.Position = position
	ind.settings.HasPosition = true
	ind.save()
}

func (ind *Indicator) save() {
	if err := ind.store.Save(ind.settings); err != nil {
		ind.log.Printf("settings not saved, kept for this run: %v", err)
	}
}
