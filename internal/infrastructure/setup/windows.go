package setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// ErrAppRunning says WhatDay is running, so files must not be touched until
// it closes.
var ErrAppRunning = errors.New(AppName + " is running; close it and try again")

// ErrAppStillRunning says WhatDay was asked to close and had not after the
// wait.
var ErrAppStillRunning = errors.New(AppName + " could not be closed; close it from its tray menu and try again")

const (
	// terminateWait bounds the wait for WhatDay to exit after it is ended.
	terminateWait = 5 * time.Second
	// terminatePoll is how often the wait looks again.
	terminatePoll = 100 * time.Millisecond
	// forcedExitCode is the exit code given to a process setup ends.
	forcedExitCode = 1

	uninstallKeyPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\` + AppName
	runKeyPath       = `Software\Microsoft\Windows\CurrentVersion\Run`
	shortcutName     = AppName + ".lnk"
	stepLogName      = AppName + "Setup.log"
	stepStamp        = "2006-01-02 15:04:05"
	logPerm          = 0o644
)

// hidden keeps a child process from flashing a console window over setup.
func hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

// AppRunning reports whether a WhatDay.exe process is running.
func AppRunning() bool { return len(processIDs(ExeName)) > 0 }

// processIDs answers every running process with the executable name given,
// compared case-insensitively. A snapshot failure answers none, so it never
// blocks a legitimate install.
func processIDs(exeName string) []uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer func() { _ = windows.CloseHandle(snapshot) }()
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil
	}
	var pids []uint32
	for {
		if strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), exeName) {
			pids = append(pids, entry.ProcessID)
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			return pids
		}
	}
}

// CloseRunningApp ends every WhatDay.exe by image name, never by process
// tree, then waits for the executable to be released.
func CloseRunningApp() error { return closeAll(ExeName) }

// closeAll ends every process running exeName, then waits up to
// terminateWait for none to be left. It takes the name so the suite can
// exercise it without touching a WhatDay the owner has open.
func closeAll(exeName string) error {
	for _, pid := range processIDs(exeName) {
		if handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid); err == nil {
			_ = windows.TerminateProcess(handle, forcedExitCode)
			_ = windows.CloseHandle(handle)
		}
	}
	deadline := time.Now().Add(terminateWait)
	for len(processIDs(exeName)) > 0 {
		if time.Now().After(deadline) {
			return ErrAppStillRunning
		}
		time.Sleep(terminatePoll)
	}
	return nil
}

// LaunchApp starts the installed WhatDay detached, so it outlives setup.
func LaunchApp() error {
	dir, err := InstallDir()
	if err != nil {
		return err
	}
	cmd := exec.Command(filepath.Join(dir, ExeName))
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", AppName, err)
	}
	return cmd.Process.Release()
}

// Record is what the uninstall entry holds.
type Record struct {
	Version, InstallDir, SetupExe, IconPath string
	EstimatedKB                             uint32
}

// WriteRecord registers WhatDay in the current user's Apps list. Modify opens
// setup on its manage screen, where Repair lives (FR-071).
func WriteRecord(r Record) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, uninstallKeyPath, registry.WRITE)
	if err != nil {
		return fmt.Errorf("creating the uninstall entry: %w", err)
	}
	defer func() { _ = key.Close() }()
	values := map[string]string{
		"DisplayName":     AppName,
		"DisplayVersion":  r.Version,
		"InstallLocation": r.InstallDir,
		"UninstallString": fmt.Sprintf("%q -uninstall", r.SetupExe),
		"ModifyPath":      fmt.Sprintf("%q", r.SetupExe),
		"DisplayIcon":     r.IconPath,
		"Publisher":       Publisher,
	}
	for name, value := range values {
		if err := key.SetStringValue(name, value); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}
	for name, value := range map[string]uint32{"NoRepair": 1, "EstimatedSize": r.EstimatedKB} {
		if err := key.SetDWordValue(name, value); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}
	return nil
}

// RemoveRecord deletes the uninstall entry.
func RemoveRecord() error {
	return registry.DeleteKey(registry.CURRENT_USER, uninstallKeyPath)
}

// InstalledVersion answers the recorded version and whether WhatDay is installed.
func InstalledVersion() (string, bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer func() { _ = key.Close() }()
	value, _, err := key.GetStringValue("DisplayVersion")
	return value, err == nil
}

// SetStartAtLogin adds or removes the Run entry that starts WhatDay when the
// user signs in (FR-060).
func SetStartAtLogin(exePath string, enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening the Run key: %w", err)
	}
	defer func() { _ = key.Close() }()
	if !enabled {
		_ = key.DeleteValue(AppName)
		return nil
	}
	if err := key.SetStringValue(AppName, fmt.Sprintf("%q", exePath)); err != nil {
		return fmt.Errorf("writing the Run entry: %w", err)
	}
	return nil
}

// StartMenuDir answers the current user's Start Menu Programs folder.
func StartMenuDir() (string, error) {
	roaming := os.Getenv("APPDATA")
	if roaming == "" {
		return "", errors.New("APPDATA is not set")
	}
	return filepath.Join(roaming, "Microsoft", "Windows", "Start Menu", "Programs"), nil
}

// CreateShortcut writes the Start Menu shortcut through the Windows Script
// Host, which is how WhatDay is started again after Quit.
func CreateShortcut(exePath, workDir string) error {
	dir, err := StartMenuDir()
	if err != nil {
		return err
	}
	link := filepath.Join(dir, shortcutName)
	script := fmt.Sprintf(`$s=(New-Object -ComObject WScript.Shell).CreateShortcut(%q);`+
		`$s.TargetPath=%q;$s.IconLocation=%q;$s.WorkingDirectory=%q;$s.Save()`,
		link, exePath, exePath, workDir)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = hidden()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating %s: %w: %s", link, err, out)
	}
	return nil
}

// RemoveShortcut deletes the Start Menu shortcut.
func RemoveShortcut() {
	if dir, err := StartMenuDir(); err == nil {
		_ = os.Remove(filepath.Join(dir, shortcutName))
	}
}

// ScheduleDirDeletion starts a hidden shell that waits for setup, running
// from inside dir, to exit and release its own executable, then removes dir.
func ScheduleDirDeletion(dir string) error {
	line := fmt.Sprintf(`ping 127.0.0.1 -n 3 >nul & rmdir /s /q "%s"`, dir)
	cmd := exec.Command("cmd", "/C", line)
	cmd.SysProcAttr = hidden()
	return cmd.Start()
}

// StepLogPath is where Step writes: the temporary folder, since uninstall
// removes WhatDay's own folders and must not leave a log behind in them.
func StepLogPath() string { return filepath.Join(os.TempDir(), stepLogName) }

// Step writes one line to the step log, opening and closing the file each
// time so every step is on disk before the next begins. The worst setup
// failures never raise; this is how they are found.
func Step(format string, args ...any) {
	file, err := os.OpenFile(StepLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, logPerm)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	_, _ = fmt.Fprintf(file, "%s  %s\n", time.Now().Format(stepStamp), fmt.Sprintf(format, args...))
}
