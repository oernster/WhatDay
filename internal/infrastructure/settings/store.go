// Package settings keeps WhatDay's settings in %APPDATA%\WhatDay\settings.json
// (FR-050 to FR-054). Ported from ED Voyage Companion's config store; unlike
// that store, Load reports a file it cannot read, because WhatDay's log must
// say why the defaults were used (FR-053).
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
)

// fileName is what the store writes inside the product's roaming folder.
const fileName = "settings.json"

const (
	dirPerm  = 0o755
	filePerm = 0o644
	// writingSuffix names the temporary file an atomic save writes first.
	writingSuffix = ".writing"
)

// stored is the file's shape, kept apart from application.Settings on
// purpose: the file is a format and the port is a type. The json names are
// the contract and they are here.
type stored struct {
	Colour   string    `json:"colour"`
	Position *position `json:"position,omitempty"`
}

type position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Store reads and writes the settings file at one path.
type Store struct{ path string }

// NewStore builds a store at the product's file inside the user's roaming
// configuration folder. A machine with no such folder is not a reason to
// refuse to start: the store then loads nothing and saving says why.
func NewStore() *Store {
	base, err := os.UserConfigDir()
	if err != nil {
		return &Store{}
	}
	return NewStoreAt(filepath.Join(base, application.ProductName, fileName))
}

// NewStoreAt builds a store at path.
func NewStoreAt(path string) *Store { return &Store{path: path} }

// Path is where the settings are kept.
func (s *Store) Path() string { return s.path }

// errNoFolder says the machine has no configuration folder.
var errNoFolder = errors.New("no configuration folder on this machine")

// Load answers what is stored. A missing file is found false with no error;
// a file that cannot be read or parsed is an error naming the file.
func (s *Store) Load() (application.Settings, bool, error) {
	if s.path == "" {
		return application.Settings{}, false, nil
	}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return application.Settings{}, false, nil
	}
	if err != nil {
		return application.Settings{}, false, fmt.Errorf("reading %s: %w", s.path, err)
	}
	var held stored
	if err := json.Unmarshal(raw, &held); err != nil {
		return application.Settings{}, false, fmt.Errorf("parsing %s: %w", s.path, err)
	}
	loaded := application.Settings{ColourName: held.Colour}
	if held.Position != nil {
		loaded.Position = domain.Point{X: held.Position.X, Y: held.Position.Y}
		loaded.HasPosition = true
	}
	return loaded, true, nil
}

// Save writes the settings through a temporary file in the same folder, then
// renames it over the target, so an interrupted save leaves the previous
// file intact (FR-051).
func (s *Store) Save(chosen application.Settings) error {
	if s.path == "" {
		return errNoFolder
	}
	held := stored{Colour: chosen.ColourName}
	if chosen.HasPosition {
		held.Position = &position{X: chosen.Position.X, Y: chosen.Position.Y}
	}
	// The encode cannot fail: stored holds a string and two ints. The error is
	// discarded rather than checked, because a branch nothing can reach is a
	// branch nothing can test.
	raw, _ := json.MarshalIndent(held, "", "  ")

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	temporary := s.path + writingSuffix
	if err := os.WriteFile(temporary, raw, filePerm); err != nil {
		return fmt.Errorf("writing %s: %w", temporary, err)
	}
	if err := os.Rename(temporary, s.path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("replacing %s: %w", s.path, err)
	}
	return nil
}
