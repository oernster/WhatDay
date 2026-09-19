package settings

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oernster/WhatDay/internal/application"
	"github.com/oernster/WhatDay/internal/domain"
)

func storeIn(t *testing.T) *Store {
	t.Helper()
	return NewStoreAt(filepath.Join(t.TempDir(), application.ProductName, fileName))
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	want := application.Settings{ColourName: "Blue", Position: domain.Point{X: -120, Y: 3672}, HasPosition: true}
	if err := store.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, found, err := store.Load()
	if err != nil || !found || got != want {
		t.Fatalf("load: got %+v, %v, %v; want %+v, true, nil", got, found, err, want)
	}
}

func TestRoundTripWithoutPosition(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	want := application.Settings{ColourName: "Red"}
	if err := store.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got, found, err := store.Load(); err != nil || !found || got != want {
		t.Fatalf("load: got %+v, %v, %v; want %+v", got, found, err, want)
	}
}

func TestFileShapeIsTheContract(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := store.Save(application.Settings{ColourName: "Green", Position: domain.Point{X: 1, Y: 2}, HasPosition: true}); err != nil {
		t.Fatalf("save: %v", err)
	}
	raw, _ := os.ReadFile(store.Path())
	want := "{\n  \"colour\": \"Green\",\n  \"position\": {\n    \"x\": 1,\n    \"y\": 2\n  }\n}"
	if string(raw) != want {
		t.Fatalf("file holds %q, want %q", raw, want)
	}
}

func TestMissingFileGivesDefaults(t *testing.T) {
	t.Parallel()
	got, found, err := storeIn(t).Load()
	if err != nil || found || got != (application.Settings{}) {
		t.Fatalf("got %+v, %v, %v; want nothing, not found, no error", got, found, err)
	}
}

func TestCorruptFileGivesDefaults(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	if err := os.MkdirAll(filepath.Dir(store.Path()), dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.Path(), []byte("{not json"), filePerm); err != nil {
		t.Fatal(err)
	}
	_, found, err := store.Load()
	if found || err == nil || !strings.Contains(err.Error(), store.Path()) {
		t.Fatalf("got found %v, error %v; want a parse error naming the file", found, err)
	}
}

func TestUnreadableFileIsAnError(t *testing.T) {
	t.Parallel()
	// A folder where the file should be cannot be read as a file.
	store := storeIn(t)
	if err := os.MkdirAll(store.Path(), dirPerm); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Load(); found || err == nil || !strings.Contains(err.Error(), store.Path()) {
		t.Fatalf("got found %v, error %v; want a read error naming the file", found, err)
	}
}

func TestSaveIsAtomic(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	first := application.Settings{ColourName: "Amber"}
	if err := store.Save(first); err != nil {
		t.Fatalf("save: %v", err)
	}
	// A folder squatting on the temporary name makes the write fail midway.
	if err := os.Mkdir(store.Path()+writingSuffix, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(application.Settings{ColourName: "Purple"}); err == nil {
		t.Fatal("save through a blocked temporary file succeeded")
	}
	if got, _, err := store.Load(); err != nil || got != first {
		t.Fatalf("after a failed save: got %+v, %v; want the previous %+v", got, err, first)
	}
}

func TestFailedRenameLeavesNoTemporaryFile(t *testing.T) {
	t.Parallel()
	store := storeIn(t)
	// A non-empty folder at the target makes the rename fail.
	if err := os.MkdirAll(filepath.Join(store.Path(), "occupied"), dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(application.Settings{ColourName: "Red"}); err == nil || !strings.Contains(err.Error(), "replacing") {
		t.Fatalf("got %v, want a replace error", err)
	}
	if _, err := os.Stat(store.Path() + writingSuffix); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary file left behind: %v", err)
	}
}

func TestUnwritableFolderIsAnError(t *testing.T) {
	t.Parallel()
	// A file where the folder should be cannot be made into a folder.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, filePerm); err != nil {
		t.Fatal(err)
	}
	store := NewStoreAt(filepath.Join(blocker, application.ProductName, fileName))
	if err := store.Save(application.Settings{ColourName: "Red"}); err == nil || !strings.Contains(err.Error(), "creating") {
		t.Fatalf("got %v, want a create error", err)
	}
}

func TestNoFolderLoadsNothingAndRefusesToSave(t *testing.T) {
	t.Parallel()
	store := &Store{}
	if got, found, err := store.Load(); err != nil || found || got != (application.Settings{}) {
		t.Fatalf("load: got %+v, %v, %v", got, found, err)
	}
	if err := store.Save(application.Settings{}); !errors.Is(err, errNoFolder) {
		t.Fatalf("save: got %v, want %v", err, errNoFolder)
	}
}

func TestNewStoreUsesTheRoamingFolder(t *testing.T) {
	base := t.TempDir()
	t.Setenv("APPDATA", base)
	if got, want := NewStore().Path(), filepath.Join(base, application.ProductName, fileName); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNewStoreWithoutRoamingFolder(t *testing.T) {
	t.Setenv("APPDATA", "")
	if got := NewStore().Path(); got != "" {
		t.Fatalf("got %q, want no path", got)
	}
}
