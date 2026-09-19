// Package zone answers the timezone Windows is set to, as a Go location built
// from the tz data embedded in the binary (FR-002, Amendment 4, C-4).
//
// Windows names its zones by key ("GMT Standard Time"); Windows' own ICU
// translates a key to the IANA name ("Europe/London"), measured on the
// reference machine on 2026-09-19 for four keys. Go's time.Local cannot be
// used: it is read once when the process starts, so a laptop that changes zone
// while WhatDay runs would keep the old one.
package zone

import (
	"errors"
	"fmt"
	"time"
	_ "time/tzdata" // the zone rules travel with the binary (C-4)
	"unsafe"

	"golang.org/x/sys/windows"
)

// info is what Windows reports about its current zone.
type info struct {
	key string
	// dstDisabled is true when "Adjust for daylight saving time
	// automatically" is off: the Windows clock then keeps standard time all
	// year, so the IANA rules would disagree with it.
	dstDisabled bool
	// biasMinutes is standard time's offset as Windows states it: UTC equals
	// local time plus the bias.
	biasMinutes int32
}

// Windows answers the zone Windows is set to now.
type Windows struct {
	read     func() (info, error)
	toIANA   func(key string) (string, error)
	load     func(name string) (*time.Location, error)
	fallback *time.Location
	key      string
	last     *time.Location
}

// NewWindows reads the real Windows settings. fallback is answered, with an
// error, until a zone has been read successfully.
func NewWindows(fallback *time.Location) *Windows {
	return &Windows{read: readWindows, toIANA: icuIANA, load: time.LoadLocation, fallback: fallback}
}

// Current answers the zone Windows is set to. On failure it answers the last
// zone it read, else the fallback, beside the error.
func (w *Windows) Current() (*time.Location, error) {
	now, err := w.read()
	if err != nil {
		return w.best(), fmt.Errorf("reading the Windows timezone: %w", err)
	}
	if now.dstDisabled {
		const secondsPerMinute = 60
		fixed := time.FixedZone(now.key+" (no daylight saving)", -int(now.biasMinutes)*secondsPerMinute)
		w.key, w.last = "", fixed
		return fixed, nil
	}
	if now.key == w.key && w.last != nil {
		return w.last, nil
	}
	name, err := w.toIANA(now.key)
	if err != nil {
		return w.best(), fmt.Errorf("naming the Windows timezone %q: %w", now.key, err)
	}
	location, err := w.load(name)
	if err != nil {
		return w.best(), fmt.Errorf("loading %s for the Windows timezone %q: %w", name, now.key, err)
	}
	w.key, w.last = now.key, location
	return location, nil
}

func (w *Windows) best() *time.Location {
	if w.last != nil {
		return w.last
	}
	return w.fallback
}

// dynamicZone is DYNAMIC_TIME_ZONE_INFORMATION.
type dynamicZone struct {
	bias            int32
	standardName    [32]uint16
	standardDate    [8]uint16
	standardBias    int32
	daylightName    [32]uint16
	daylightDate    [8]uint16
	daylightBias    int32
	keyName         [128]uint16
	dynamicDisabled uint8
}

// zoneIDInvalid is TIME_ZONE_ID_INVALID, the failure answer.
const zoneIDInvalid = 0xFFFFFFFF

var (
	pGetDynamicZone = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetDynamicTimeZoneInformation")
	pICUZoneID      = windows.NewLazySystemDLL("icu.dll").NewProc("ucal_getTimeZoneIDForWindowsID")
)

func readWindows() (info, error) {
	var z dynamicZone
	answer, _, callErr := pGetDynamicZone.Call(uintptr(unsafe.Pointer(&z)))
	if uint32(answer) == zoneIDInvalid {
		return info{}, callErr
	}
	return info{
		key:         windows.UTF16ToString(z.keyName[:]),
		dstDisabled: z.dynamicDisabled != 0,
		biasMinutes: z.bias + z.standardBias,
	}, nil
}

// ianaCapacity is room for the longest IANA name, with plenty to spare.
const ianaCapacity = 128

// errNoIANAName says ICU knows no IANA name for a Windows key.
var errNoIANAName = errors.New("no IANA name for it")

// icuIANA asks Windows' ICU for the IANA name of a Windows zone key.
func icuIANA(key string) (string, error) {
	if err := pICUZoneID.Find(); err != nil {
		return "", err
	}
	id, err := windows.UTF16FromString(key)
	if err != nil {
		return "", err
	}
	out := make([]uint16, ianaCapacity)
	var status int32
	length, _, _ := pICUZoneID.Call(
		uintptr(unsafe.Pointer(&id[0])), uintptr(len(id)-1),
		0, // no region: the key's default zone
		uintptr(unsafe.Pointer(&out[0])), uintptr(len(out)),
		uintptr(unsafe.Pointer(&status)))
	if status > 0 {
		return "", fmt.Errorf("ICU status %d", status)
	}
	if length == 0 {
		return "", errNoIANAName
	}
	return windows.UTF16ToString(out[:length]), nil
}
