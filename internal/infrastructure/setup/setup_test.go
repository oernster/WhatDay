package setup

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecideEveryRoute(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name              string
		uninstall, exists bool
		installed, this   string
		want              Route
	}{
		{"nothing recorded", false, false, "", "0.1.0", RouteInstall},
		{"older recorded", false, true, "0.1.0", "0.2.0", RouteUpdate},
		{"newer recorded", false, true, "0.3.0", "0.2.0", RouteDowngrade},
		{"same recorded", false, true, "0.2.0", "0.2.0", RouteManage},
		{"uninstall asked first", true, true, "0.1.0", "0.2.0", RouteUninstall},
		{"uninstall asked with nothing recorded", true, false, "", "0.2.0", RouteUninstall},
		{"numeric, not text, comparison", false, true, "0.9.0", "0.10.0", RouteUpdate},
		{"pre-release suffix ignored", false, true, "1.0.0-rc1", "1.0.0", RouteManage},
	}
	for _, c := range cases {
		if got := Decide(c.uninstall, c.exists, c.installed, c.this); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

// zipOf builds an archive holding the named files.
func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractZipWritesTheFiles(t *testing.T) {
	t.Parallel()
	dest := filepath.Join(t.TempDir(), AppName)
	data := zipOf(t, map[string]string{ExeName: "exe", "sub/readme.txt": "hi", "empty/": ""})
	if err := ExtractZip(data, dest); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{ExeName: "exe", "sub/readme.txt": "hi"} {
		if got, err := os.ReadFile(filepath.Join(dest, name)); err != nil || string(got) != want {
			t.Errorf("%s: got %q, %v", name, got, err)
		}
	}
	if info, err := os.Stat(filepath.Join(dest, "empty")); err != nil || !info.IsDir() {
		t.Errorf("folder entry: %v", err)
	}
}

func TestExtractZipRefusesAnEscapeBeforeWritingAnything(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dest := filepath.Join(root, AppName)
	data := zipOf(t, map[string]string{"a-first.txt": "fine", "../escaped.txt": "bad"})
	err := ExtractZip(data, dest)
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("got %v, want an unsafe path refusal", err)
	}
	if _, err := os.Stat(filepath.Join(root, "escaped.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the escaping entry was written")
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Error("the install folder was started before the payload was checked")
	}
}

func TestExtractZipRefusesAPayloadThatIsNotAZip(t *testing.T) {
	t.Parallel()
	if err := ExtractZip([]byte("not a zip"), t.TempDir()); err == nil {
		t.Fatal("a broken payload was accepted")
	}
}

func TestExtractZipIntoAFileFails(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ExtractZip(zipOf(t, map[string]string{ExeName: "exe"}), filepath.Join(blocker, AppName)); err == nil {
		t.Fatal("extracting under a file succeeded")
	}
}

func TestDirSizeKB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const kilobytes = 3
	if err := os.WriteFile(filepath.Join(dir, "f"), make([]byte, kilobytes*bytesPerKB), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := DirSizeKB(dir); err != nil || got != kilobytes {
		t.Fatalf("got %d, %v; want %d", got, err, kilobytes)
	}
	if _, err := DirSizeKB(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("a missing folder was measured")
	}
}

func TestInstallDirAndDataDirs(t *testing.T) {
	base := t.TempDir()
	t.Setenv(localData, base)
	t.Setenv("APPDATA", filepath.Join(base, "Roaming"))
	if got, err := InstallDir(); err != nil || got != filepath.Join(base, installSubdir, AppName) {
		t.Errorf("install dir: got %q, %v", got, err)
	}
	want := []string{filepath.Join(base, "Roaming", AppName), filepath.Join(base, AppName)}
	if got := DataDirs(); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("data dirs: got %q, want %q", got, want)
	}
	t.Setenv(localData, "")
	if _, err := InstallDir(); !errors.Is(err, errNoLocalData) {
		t.Errorf("without %s: got %v", localData, err)
	}
}

func TestStepAppendsToTheStepLog(t *testing.T) {
	t.Setenv("TMP", t.TempDir())
	Step("first %d", 1)
	Step("second")
	raw, err := os.ReadFile(StepLogPath())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "first 1") || !strings.HasSuffix(lines[1], "second") {
		t.Fatalf("got %q", lines)
	}
}

func TestQuotedWritesThePathAsWindowsDoes(t *testing.T) {
	path := filepath.Join(`C:\Users\Someone\AppData\Local`, installSubdir, AppName, ExeName)
	got := quoted(path)
	if strings.Contains(got, `\\`) {
		t.Fatalf("%s: a registry path needs single separators", got)
	}
	if got != `"`+path+`"` {
		t.Fatalf("got %s, want the path in plain quotes", got)
	}
}

func TestRunTargetReadsTheEntryBack(t *testing.T) {
	path := filepath.Join(`C:\Users\Some One\AppData\Local`, installSubdir, AppName, ExeName)
	for _, stored := range []string{quoted(path), quoted(path) + " -flag", "  " + quoted(path) + "  ", path} {
		if got := runTarget(stored); got != path {
			t.Errorf("runTarget(%q) = %q, want %q", stored, got, path)
		}
	}
	// A value with an opening quote and no closing one is taken whole, less
	// the stray quote, rather than cut short.
	if got := runTarget(`"` + path); got != path {
		t.Errorf("an unclosed quote read back as %q", got)
	}
}

func TestStartsAtLoginOnlyForAnEntryNamingARealFile(t *testing.T) {
	real := filepath.Join(t.TempDir(), ExeName)
	if err := os.WriteFile(real, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), ExeName)
	entry := func(value string, err error) func() (string, error) {
		return func() (string, error) { return value, err }
	}
	cases := []struct {
		name string
		read func() (string, error)
		want bool
	}{
		{"an entry naming the installed file", entry(quoted(real), nil), true},
		{"an entry naming a file that has gone", entry(quoted(gone), nil), false},
		{"no entry", entry("", errors.New("not found")), false},
	}
	for _, c := range cases {
		if got := startsAtLogin(c.read, fileExists); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// The process tests never name WhatDay.exe: the owner's own copy is usually
// running. The suite must neither end it nor measure differently for it.

func TestProcessIDsFindsThisTest(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if got := processIDs(strings.ToUpper(filepath.Base(self))); len(got) == 0 {
		t.Fatalf("the running test binary %s was not found", filepath.Base(self))
	}
}

func TestCloseAllWithNothingRunningAnswersAtOnce(t *testing.T) {
	if err := closeAll("WhatDay-no-such-program.exe"); err != nil {
		t.Fatalf("closing nothing: %v", err)
	}
}

func TestAppRunningAsksAboutWhatDay(t *testing.T) {
	if got, want := AppRunning(), len(processIDs(ExeName)) > 0; got != want {
		t.Fatalf("AppRunning %v, processIDs says %v", got, want)
	}
}
