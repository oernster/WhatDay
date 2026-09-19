package zone

import (
	"errors"
	"strings"
	"testing"
	"time"
)

var errPlanted = errors.New("planted")

// fake builds a Windows whose readers answer from the fields given.
type fake struct {
	info      info
	readErr   error
	iana      map[string]string
	loadErr   error
	ianaCalls int
}

func (f *fake) windows(fallback *time.Location) *Windows {
	return &Windows{
		read: func() (info, error) { return f.info, f.readErr },
		toIANA: func(key string) (string, error) {
			f.ianaCalls++
			if name, ok := f.iana[key]; ok {
				return name, nil
			}
			return "", errNoIANAName
		},
		load: func(name string) (*time.Location, error) {
			if f.loadErr != nil {
				return nil, f.loadErr
			}
			return time.LoadLocation(name)
		},
		fallback: fallback,
	}
}

var keys = map[string]string{
	"GMT Standard Time":         "Europe/London",
	"Eastern Standard Time":     "America/New_York",
	"AUS Eastern Standard Time": "Australia/Sydney",
}

func TestReadsAndCachesTheZone(t *testing.T) {
	t.Parallel()
	f := &fake{info: info{key: "GMT Standard Time"}, iana: keys}
	w := f.windows(time.UTC)
	for range 3 {
		if got, err := w.Current(); err != nil || got.String() != "Europe/London" {
			t.Fatalf("got %v, %v; want Europe/London", got, err)
		}
	}
	if f.ianaCalls != 1 {
		t.Fatalf("asked ICU %d times for one key, want once", f.ianaCalls)
	}
}

func TestFollowsAChangeOfZone(t *testing.T) {
	t.Parallel()
	f := &fake{info: info{key: "GMT Standard Time"}, iana: keys}
	w := f.windows(time.UTC)
	_, _ = w.Current()
	f.info.key = "AUS Eastern Standard Time"
	if got, err := w.Current(); err != nil || got.String() != "Australia/Sydney" {
		t.Fatalf("got %v, %v; want Australia/Sydney", got, err)
	}
}

func TestFailureAnswersTheFallbackFirst(t *testing.T) {
	t.Parallel()
	f := &fake{readErr: errPlanted, iana: keys}
	got, err := f.windows(time.UTC).Current()
	if !errors.Is(err, errPlanted) || got != time.UTC {
		t.Fatalf("got %v, %v; want the fallback and the error", got, err)
	}
}

func TestFailureAnswersTheLastGoodZone(t *testing.T) {
	t.Parallel()
	f := &fake{info: info{key: "Eastern Standard Time"}, iana: keys}
	w := f.windows(time.UTC)
	_, _ = w.Current()
	cases := []struct {
		name  string
		plant func()
		want  string
	}{
		{"unknown key", func() { f.info.key = "No Such Zone" }, "naming"},
		{"load fails", func() { f.info.key = "GMT Standard Time"; f.loadErr = errPlanted }, "loading Europe/London"},
		{"read fails", func() { f.readErr = errPlanted }, "reading"},
	}
	for _, c := range cases {
		c.plant()
		got, err := w.Current()
		if err == nil || !strings.Contains(err.Error(), c.want) || got.String() != "America/New_York" {
			t.Errorf("%s: got %v, %v; want America/New_York and an error mentioning %q", c.name, got, err, c.want)
		}
	}
}

func TestDaylightSavingOffKeepsStandardTime(t *testing.T) {
	t.Parallel()
	// Windows states UK standard time as bias 0; New York as bias 300.
	f := &fake{info: info{key: "Eastern Standard Time", dstDisabled: true, biasMinutes: 300}, iana: keys}
	got, err := f.windows(time.UTC).Current()
	if err != nil {
		t.Fatal(err)
	}
	// In July New York observes daylight saving; with it switched off the
	// clock stays five hours behind UTC.
	july := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	const fiveHoursWest = -5 * 60 * 60
	if _, offset := july.In(got).Zone(); offset != fiveHoursWest {
		t.Fatalf("offset %d, want %d", offset, fiveHoursWest)
	}
	if f.ianaCalls != 0 {
		t.Fatal("asked ICU for a zone whose rules are not used")
	}
}

// The real Windows path on this machine: reads a key and names it.
func TestRealWindowsZoneResolves(t *testing.T) {
	t.Parallel()
	got, err := NewWindows(time.UTC).Current()
	if err != nil || got == nil || got == time.UTC {
		t.Fatalf("got %v, %v; want the machine's zone", got, err)
	}
}

func TestRealICUNamesKnownKeys(t *testing.T) {
	t.Parallel()
	for key, want := range keys {
		if got, err := icuIANA(key); err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", key, got, err, want)
		}
	}
	if _, err := icuIANA("No Such Zone"); !errors.Is(err, errNoIANAName) {
		t.Errorf("unknown key: got %v, want %v", err, errNoIANAName)
	}
	if _, err := icuIANA("bad\x00key"); err == nil {
		t.Error("a key holding NUL was accepted")
	}
}
