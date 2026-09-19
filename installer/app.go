package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/oernster/WhatDay/internal/infrastructure/setup"
)

// App is the Wails facade for the setup program; the install policy lives in
// internal/infrastructure/setup. Ported from PigeonPost's installer.
type App struct {
	ctx            context.Context
	payload        []byte
	version        string
	uninstallAsked bool
}

// uninstallFlag is what the Apps list's uninstall entry passes.
const uninstallFlag = "-uninstall"

// NewApp builds the facade; run with -uninstall, setup opens on removal.
func NewApp(payload []byte, version string) *App {
	asked := len(os.Args) > 1 && os.Args[1] == uninstallFlag
	return &App{payload: payload, version: version, uninstallAsked: asked}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// State is one reading of the machine, which decides the whole conversation.
type State struct {
	Route            setup.Route `json:"route"`
	Installed        bool        `json:"installed"`
	InstalledVersion string      `json:"installedVersion"`
	ThisVersion      string      `json:"thisVersion"`
	InstallDir       string      `json:"installDir"`
	StepLog          string      `json:"stepLog"`
}

// Progress is emitted on the "progress" event while work runs.
type Progress struct {
	Pct int    `json:"pct"`
	Msg string `json:"msg"`
}

// DetectState reads the machine once and decides the route.
func (a *App) DetectState() State {
	dir, _ := setup.InstallDir()
	installed, found := setup.InstalledVersion()
	route := setup.Decide(a.uninstallAsked, found, installed, a.version)
	setup.Step("setup %s started: route %s, installed %q", a.version, route, installed)
	return State{Route: route, Installed: found, InstalledVersion: installed, ThisVersion: a.version, InstallDir: dir, StepLog: setup.StepLogPath()}
}

// AppRunning reports whether WhatDay is running, asked before any file is touched.
func (a *App) AppRunning() bool { return setup.AppRunning() }

// CloseRunningApp ends the running WhatDay so setup can go on.
func (a *App) CloseRunningApp() error {
	setup.Step("closing the running %s", setup.AppName)
	return a.logged(setup.CloseRunningApp())
}

// Progress weights, cumulative, in per cent. Weighted by time rather than
// by step count: Stellody measured a shortcut at about half a second, far
// longer than extracting a few megabytes, so the shortcut gets the widest
// share. WhatDay's own timings are in the step log to re-weight from.
const (
	pctExtracted  = 20
	pctSetupCopy  = 35
	pctRegistered = 45
	pctShortcut   = 90
	pctDone       = 100
)

// Install installs, updates, goes back or repairs: every route writes the
// same files and entries (FR-060, FR-070 to FR-072).
func (a *App) Install() error {
	if setup.AppRunning() {
		return a.logged(setup.ErrAppRunning)
	}
	dir, err := setup.InstallDir()
	if err != nil {
		return a.logged(err)
	}
	exe := filepath.Join(dir, setup.ExeName)
	a.progress(0, "Extracting "+setup.AppName)
	if err := setup.ExtractZip(a.payload, dir); err != nil {
		return a.logged(err)
	}
	a.progress(pctExtracted, "Keeping a copy of setup for Modify and Uninstall")
	setupCopy := filepath.Join(dir, setup.SetupName)
	if err := copySelf(setupCopy); err != nil {
		return a.logged(err)
	}
	a.progress(pctSetupCopy, "Registering "+setup.AppName)
	size, _ := setup.DirSizeKB(dir)
	record := setup.Record{Version: a.version, InstallDir: dir, SetupExe: setupCopy, IconPath: exe, EstimatedKB: size}
	if err := setup.WriteRecord(record); err != nil {
		return a.logged(err)
	}
	a.progress(pctRegistered, "Adding the Start Menu shortcut")
	if err := setup.CreateShortcut(exe, dir); err != nil {
		return a.logged(err)
	}
	a.progress(pctShortcut, "Starting "+setup.AppName+" when you sign in")
	if err := setup.SetStartAtLogin(exe, true); err != nil {
		return a.logged(err)
	}
	a.progress(pctDone, "Done")
	return nil
}

// Uninstall removes the shortcut, the login entry, the Apps list entry,
// WhatDay's settings and log, then the install folder (FR-073).
func (a *App) Uninstall() error {
	if setup.AppRunning() {
		return a.logged(setup.ErrAppRunning)
	}
	dir, err := setup.InstallDir()
	if err != nil {
		return a.logged(err)
	}
	a.progress(0, "Removing the shortcut and the login entry")
	setup.RemoveShortcut()
	if err := setup.SetStartAtLogin("", false); err != nil {
		return a.logged(err)
	}
	a.progress(pctSetupCopy, "Removing the Apps list entry")
	if err := setup.RemoveRecord(); err != nil {
		setup.Step("no Apps list entry to remove: %v", err)
	}
	a.progress(pctRegistered, "Removing settings")
	for _, data := range setup.DataDirs() {
		if err := os.RemoveAll(data); err != nil {
			return a.logged(fmt.Errorf("removing %s: %w", data, err))
		}
	}
	a.progress(pctShortcut, "Removing the program files")
	if err := setup.ScheduleDirDeletion(dir); err != nil {
		return a.logged(fmt.Errorf("scheduling the removal of %s: %w", dir, err))
	}
	a.progress(pctDone, "Done")
	return nil
}

// LaunchApp starts WhatDay after setup, when asked.
func (a *App) LaunchApp() error {
	setup.Step("launching %s", setup.AppName)
	return a.logged(setup.LaunchApp())
}

// Quit closes setup.
func (a *App) Quit() { wailsruntime.Quit(a.ctx) }

func (a *App) progress(pct int, msg string) {
	setup.Step("%3d%%  %s", pct, msg)
	wailsruntime.EventsEmit(a.ctx, "progress", Progress{Pct: pct, Msg: msg})
}

// logged records an error in the step log and answers it unchanged.
func (a *App) logged(err error) error {
	if err != nil {
		setup.Step("failed: %v", err)
	}
	return err
}

// copySelf copies this setup program to target, unless it is already running
// from there (setup opened from the Apps list, then Repair).
func copySelf(target string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding setup itself: %w", err)
	}
	if strings.EqualFold(filepath.Clean(self), filepath.Clean(target)) {
		return nil
	}
	in, err := os.Open(self)
	if err != nil {
		return fmt.Errorf("reading setup: %w", err)
	}
	defer func() { _ = in.Close() }()
	const exePerm = 0o755
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, exePerm)
	if err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	return nil
}
