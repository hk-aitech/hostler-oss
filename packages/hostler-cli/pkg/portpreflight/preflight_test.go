package portpreflight

import (
	"os"
	"path/filepath"
	"testing"
)

// preflight scope analyser unit tests.

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s failed: %v", name, err)
	}
}

func TestAnalyze_BasicCount(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", `package cmd

import "pkg/mailbox"

func cmdA() {
	mailbox.SendMessage("x")
	mailbox.ListInbox()
	mailbox.SendMessage("y")
}
`)
	writeFile(t, dir, "b.go", `package cmd

import "pkg/mailbox"

func cmdB() {
	mailbox.ListInbox()
}
`)

	r, err := Analyze("mailbox", dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Package != "mailbox" {
		t.Errorf("package name mismatch: %s", r.Package)
	}
	if r.CallsTotal != 4 {
		t.Errorf("expected 4 total calls, got %d", r.CallsTotal)
	}
	if r.FunctionsCount != 2 {
		t.Errorf("expected 2 unique functions, got %d", r.FunctionsCount)
	}
	// SendMessage 2 + ListInbox 2 — same count, sorted alphabetically by name.
	if r.Functions[0].Name != "ListInbox" || r.Functions[0].Count != 2 {
		t.Errorf("Functions[0] = %+v (expected ListInbox:2)", r.Functions[0])
	}
	if r.Functions[1].Name != "SendMessage" || r.Functions[1].Count != 2 {
		t.Errorf("Functions[1] = %+v (expected SendMessage:2)", r.Functions[1])
	}
	if r.FilesWithCalls != 2 || r.FilesScanned != 2 {
		t.Errorf("file stats: scanned=%d withCalls=%d", r.FilesScanned, r.FilesWithCalls)
	}
}

func TestAnalyze_ExcludesCompositionAndTestFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real.go", `package cmd
import "pkg/mailbox"
func real() { mailbox.Send("x") }
`)
	writeFile(t, dir, "composition.go", `package cmd
import "pkg/mailbox"
func wire() { mailbox.Send("composition"); mailbox.Send("composition2") }
`)
	writeFile(t, dir, "real_test.go", `package cmd
import "pkg/mailbox"
func TestX() { mailbox.Send("test") }
`)

	r, err := Analyze("mailbox", dir)
	if err != nil {
		t.Fatal(err)
	}
	// composition.go and _test.go are excluded. Only the 1 call from real.go counts.
	if r.CallsTotal != 1 {
		t.Errorf("expected CallsTotal 1 (composition/test excluded), got %d", r.CallsTotal)
	}
	if r.FilesScanned != 1 {
		t.Errorf("expected FilesScanned 1, got %d", r.FilesScanned)
	}
}

func TestAnalyze_WordBoundary(t *testing.T) {
	dir := t.TempDir()
	// otherMailbox.X() must not match (prefix boundary).
	writeFile(t, dir, "a.go", `package cmd
import "pkg/mailbox"

func cmd() {
	mailbox.Good()        // should match
	otherMailbox.Bad()    // should NOT match
}
`)
	r, err := Analyze("mailbox", dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.CallsTotal != 1 {
		t.Errorf("expected CallsTotal 1 (boundary match only), got %d — %+v", r.CallsTotal, r.Functions)
	}
	if r.Functions[0].Name != "Good" {
		t.Errorf("Functions[0].Name = %s (expected Good)", r.Functions[0].Name)
	}
}

func TestAnalyze_LowercaseFunctionIgnored(t *testing.T) {
	dir := t.TempDir()
	// Lowercase-leading functions are private and unreachable from cmd — excluded.
	writeFile(t, dir, "a.go", `package cmd
import "pkg/mailbox"
func cmd() {
	mailbox.privateFunc("x")
	mailbox.PublicFunc("y")
}
`)
	r, err := Analyze("mailbox", dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.CallsTotal != 1 {
		t.Errorf("expected CallsTotal 1 (public only), got %d", r.CallsTotal)
	}
}

func TestAnalyze_EmptyPackage(t *testing.T) {
	_, err := Analyze("", "/tmp")
	if err == nil {
		t.Error("expected error for empty package name")
	}
}

func TestAnalyze_NonexistentDir(t *testing.T) {
	_, err := Analyze("x", "/no/such/dir/T821")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}
