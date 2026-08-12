package main

import (
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.txt")
	writeFile(t, existing, "x")

	if !fileExists(existing) {
		t.Errorf("fileExists(%q) = false, want true", existing)
	}
	if fileExists(filepath.Join(dir, "missing.txt")) {
		t.Errorf("fileExists(missing) = true, want false")
	}
}

func TestCommandExists(t *testing.T) {
	if !commandExists("go") {
		t.Error(`commandExists("go") = false, want true (needed to build the tests themselves)`)
	}
	if commandExists("definitely-not-a-real-command-xyz") {
		t.Error("commandExists returned true for a nonexistent command")
	}
}

func TestGeneratePassword(t *testing.T) {
	a, err := generatePassword(32)
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	b, err := generatePassword(32)
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	if a == "" {
		t.Fatal("generatePassword returned an empty string")
	}
	if a == b {
		t.Error("two calls to generatePassword returned the same result")
	}
}
