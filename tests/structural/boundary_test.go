// Package structural enforces the architecture with tests rather than convention
// (REQUIREMENTS.md C-3, NFR-MAINT-002, NFR-PRIV-001). Ported from ED Voyage
// Companion's tests/structural.
//
// Every assertion here has been proved to bite by planting a violation and
// watching it fail. An assertion never seen to fail is not yet a guard.
package structural

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lineLimit is the module-size cap. dangerBand is five per cent below it: a
// file that lands between them is refactored down to safeLanding rather than
// left one edit away from breaching, because shaving a line or two buys
// nothing.
const (
	lineLimit   = 400
	dangerBand  = lineLimit - lineLimit/20
	safeLanding = 350
)

// modulePath prefixes every internal import.
const modulePath = "github.com/oernster/WhatDay/"

// compositionRoot names the files allowed to import both application and
// infrastructure.
var compositionRoot = map[string]bool{"cmd/whatday/main.go": true}

// forbiddenInCore names imports that would give the domain or the
// application I/O or a platform: those belong to infrastructure.
var forbiddenInCore = []string{
	"os", "os/exec", "path/filepath", "io/ioutil", "math/rand", "math/rand/v2",
	"syscall", "unsafe", "golang.org/x/sys/windows",
}

// forbiddenCallsInCore read the wall clock or a global random source, which
// would make behaviour irreproducible; the clock arrives through a port.
var forbiddenCallsInCore = []string{"time.Now", "time.Since", "time.Until", "rand.Intn", "rand.Float64"}

// skippedDirs are never walked: they hold other people's code or no source.
var skippedDirs = map[string]bool{".git": true, "node_modules": true, "venv": true}

// repoRoot walks up from the test's directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for range 6 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not find go.mod above the test directory")
	return ""
}

// goFiles returns every Go source file in the repository.
func goFiles(t *testing.T) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(repoRoot(t), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking repository: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("no Go files found, the walk is wrong")
	}
	return found
}

// importsOf parses a file and returns its import paths.
func importsOf(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	out := make([]string, 0, len(parsed.Imports))
	for _, item := range parsed.Imports {
		out = append(out, strings.Trim(item.Path.Value, `"`))
	}
	return out
}

// layerOf returns which layer under internal/ a file belongs to.
func layerOf(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) >= 2 && parts[0] == "internal" {
		return parts[1]
	}
	return ""
}

func relativeTo(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

func isTest(path string) bool { return strings.HasSuffix(path, "_test.go") }

// internalLayer answers the layer an import path points into; empty when it
// is not one of ours.
func internalLayer(imported string) string {
	inner, ours := strings.CutPrefix(imported, modulePath+"internal/")
	if !ours {
		return ""
	}
	layer, _, _ := strings.Cut(inner, "/")
	return layer
}

func TestDomainDependsOnNothingOfOurs(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "domain" || isTest(path) {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if layer := internalLayer(imported); layer != "" && layer != "domain" {
				t.Errorf("%s imports %s: the domain depends on nothing", relativeTo(root, path), imported)
			}
		}
	}
}

func TestApplicationDependsOnDomainOnly(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if layerOf(root, path) != "application" || isTest(path) {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if layer := internalLayer(imported); layer != "" && layer != "domain" && layer != "application" {
				t.Errorf("%s imports %s: the application depends on the domain and its own ports", relativeTo(root, path), imported)
			}
		}
	}
}

func TestCoreIsPure(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		layer := layerOf(root, path)
		if (layer != "domain" && layer != "application") || isTest(path) {
			continue
		}
		for _, imported := range importsOf(t, path) {
			for _, banned := range forbiddenInCore {
				if imported == banned {
					t.Errorf("%s imports %q: %s performs no I/O and touches no platform", relativeTo(root, path), banned, layer)
				}
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, call := range forbiddenCallsInCore {
			if strings.Contains(string(raw), call+"(") {
				t.Errorf("%s calls %s: the clock arrives through the Clock port", relativeTo(root, path), call)
			}
		}
	}
}

func TestUIAndInfrastructureStayApart(t *testing.T) {
	root := repoRoot(t)
	apart := map[string]string{"ui": "infrastructure", "infrastructure": "ui"}
	for _, path := range goFiles(t) {
		layer := layerOf(root, path)
		other, governed := apart[layer]
		if !governed || isTest(path) {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if internalLayer(imported) == other {
				t.Errorf("%s imports %s: %s never reaches into %s", relativeTo(root, path), imported, layer, other)
			}
		}
	}
}

func TestCompositionRootIsWhitelisted(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if isTest(path) {
			continue
		}
		var application, infrastructure bool
		for _, imported := range importsOf(t, path) {
			application = application || internalLayer(imported) == "application"
			infrastructure = infrastructure || internalLayer(imported) == "infrastructure"
		}
		if application && infrastructure && !compositionRoot[relativeTo(root, path)] {
			t.Errorf("%s wires application to infrastructure: only the composition root may", relativeTo(root, path))
		}
	}
}

func TestNoNetworkImports(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		for _, imported := range importsOf(t, path) {
			if imported == "net" || strings.HasPrefix(imported, "net/") {
				t.Errorf("%s imports %s: WhatDay makes no network connections (NFR-PRIV-001)", relativeTo(root, path), imported)
			}
		}
	}
}

func TestTestSupportIsForTestsOnly(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if isTest(path) || layerOf(root, path) == "testsupport" {
			continue
		}
		for _, imported := range importsOf(t, path) {
			if internalLayer(imported) == "testsupport" {
				t.Errorf("%s imports %s: test support is for tests only", relativeTo(root, path), imported)
			}
		}
	}
}

// lineCount counts the lines in a file exactly: a final newline ends the last
// line rather than starting another.
func lineCount(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	text := string(raw)
	count := strings.Count(text, "\n")
	if text != "" && !strings.HasSuffix(text, "\n") {
		count++
	}
	return count
}

func TestNoFileExceedsLineLimit(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if count := lineCount(t, path); count > lineLimit {
			t.Errorf("%s has %d lines, over the %d limit", relativeTo(root, path), count, lineLimit)
		}
	}
}

func TestNoFileInDangerBand(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		count := lineCount(t, path)
		// The band is 381 to 399: a file of exactly lineLimit lines is at the
		// cap, which the cap allows.
		if count > dangerBand && count < lineLimit {
			t.Errorf("%s has %d lines, inside the danger band %d to %d: reduce it to %d or fewer",
				relativeTo(root, path), count, dangerBand+1, lineLimit-1, safeLanding)
		}
	}
}

func TestEveryExportedTypeIsDocumented(t *testing.T) {
	root := repoRoot(t)
	for _, path := range goFiles(t) {
		if isTest(path) {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE || general.Doc != nil {
				continue
			}
			for _, spec := range general.Specs {
				typed, ok := spec.(*ast.TypeSpec)
				if ok && typed.Name.IsExported() && typed.Doc == nil {
					t.Errorf("%s: exported type %s has no doc comment", relativeTo(root, path), typed.Name.Name)
				}
			}
		}
	}
}
