// Package setup holds the per-user install policy for the WhatDay setup
// program (FR-070 to FR-074): where WhatDay lives, extracting the payload
// plus which conversation a run of setup is having. The registry, shortcut
// and process work is in windows.go. Ported from PigeonPost's
// internal/installer; the setup program's app.go is a facade over it.
package setup

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/oernster/WhatDay/internal/application"
)

const (
	// AppName names the install folder, the registry keys and the shortcut.
	AppName = application.ProductName
	// ExeName is the installed application executable.
	ExeName = AppName + ".exe"
	// SetupName is the copy of the setup program kept in the install folder,
	// which the Apps list opens to modify or remove WhatDay.
	SetupName = AppName + "Setup.exe"
	// Publisher is recorded in the uninstall entry.
	Publisher = application.Author

	installSubdir = "Programs"
	dirPerm       = 0o755
	bytesPerKB    = 1024
	localData     = "LOCALAPPDATA"
)

var errNoLocalData = errors.New(localData + " is not set")

// InstallDir answers %LOCALAPPDATA%\Programs\WhatDay: per user, so setup
// never needs administrator rights (FR-070).
func InstallDir() (string, error) {
	base := os.Getenv(localData)
	if base == "" {
		return "", errNoLocalData
	}
	return filepath.Join(base, installSubdir, AppName), nil
}

// DataDirs answers the folders WhatDay itself writes: its settings under the
// roaming folder and its log under the local one. Uninstall removes both
// (FR-073).
func DataDirs() []string {
	var dirs []string
	if roaming, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(roaming, AppName))
	}
	if local := os.Getenv(localData); local != "" {
		dirs = append(dirs, filepath.Join(local, AppName))
	}
	return dirs
}

// ExtractZip extracts a zip archive into dest. Every entry is checked against
// dest before any is written, so a crafted archive cannot escape the install
// folder or leave a half-written install behind.
func ExtractZip(data []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("opening the payload: %w", err)
	}
	for _, file := range reader.File {
		if _, err := fenced(dest, file.Name); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dest, dirPerm); err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	for _, file := range reader.File {
		if err := extractEntry(file, dest); err != nil {
			return err
		}
	}
	return nil
}

// fenced answers where name lands under dest; an error when it would land
// outside it.
func fenced(dest, name string) (string, error) {
	target := filepath.Join(dest, name)
	inside := filepath.Clean(dest) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), inside) {
		return "", fmt.Errorf("unsafe path in the payload: %q", name)
	}
	return target, nil
}

func extractEntry(file *zip.File, dest string) error {
	target, _ := fenced(dest, file.Name) // checked before extraction began
	if file.FileInfo().IsDir() {
		return os.MkdirAll(target, dirPerm)
	}
	if err := os.MkdirAll(filepath.Dir(target), dirPerm); err != nil {
		return fmt.Errorf("creating the folder for %s: %w", target, err)
	}
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("opening %s in the payload: %w", file.Name, err)
	}
	defer func() { _ = src.Close() }()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, dirPerm)
	if err != nil {
		return fmt.Errorf("creating %s: %w", target, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	return nil
}

// DirSizeKB answers the size of a folder tree in kilobytes, for the Apps
// list's size column.
func DirSizeKB(dir string) (uint32, error) {
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("measuring %s: %w", dir, err)
	}
	return uint32(total / bytesPerKB), nil
}

// Route is which conversation a run of setup is having, decided once from
// one reading of the machine (installer house model).
type Route string

// The routes.
const (
	RouteInstall   Route = "install"
	RouteUpdate    Route = "update"
	RouteDowngrade Route = "downgrade"
	RouteManage    Route = "manage"
	RouteUninstall Route = "uninstall"
)

// Decide answers the route. An asked-for uninstall settles it first. Then
// nothing recorded installs; an older record updates; a newer record goes
// back; a match manages.
func Decide(uninstallAsked, installed bool, installedVersion, thisVersion string) Route {
	switch {
	case uninstallAsked:
		return RouteUninstall
	case !installed:
		return RouteInstall
	case newer(thisVersion, installedVersion):
		return RouteUpdate
	case newer(installedVersion, thisVersion):
		return RouteDowngrade
	}
	return RouteManage
}

// newer reports whether version a is strictly newer than b, comparing
// major.minor.patch numerically and ignoring any pre-release suffix.
func newer(a, b string) bool {
	pa, pb := parseVersion(a), parseVersion(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	core, _, _ := strings.Cut(strings.TrimSpace(v), "-")
	parts := strings.Split(core, ".")
	var out [3]int
	for i := 0; i < len(out) && i < len(parts); i++ {
		out[i], _ = strconv.Atoi(strings.TrimSpace(parts[i]))
	}
	return out
}
