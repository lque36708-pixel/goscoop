package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestShims(t *testing.T) string {
	dir := t.TempDir()
	os.Setenv("SCOOP", filepath.Join(dir, "scoop"))
	return dir
}

func writeShim(t *testing.T, shimsDir, base string, path string) {
	t.Helper()
	content := "path = \"" + path + "\"\r\n"
	if err := os.WriteFile(filepath.Join(shimsDir, base+".shim"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	copy := make([]byte, 5)
	if err := os.WriteFile(filepath.Join(shimsDir, base+".exe"), copy, 0755); err != nil {
		t.Fatal(err)
	}
}

func writeCmdShim(t *testing.T, shimsDir, base, app string) {
	t.Helper()
	content := "@echo off\r\n\"%~dp0..\\apps\\" + app + "\\current\\tool.exe\" %*\r\n"
	if err := os.WriteFile(filepath.Join(shimsDir, base+".cmd"), []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestRemove_ByName(t *testing.T) {
	dir := setupTestShims(t)
	shimsDir := filepath.Join(dir, "scoop", "shims")
	os.MkdirAll(shimsDir, 0755)
	writeShim(t, shimsDir, "myapp", `C:\scoop\apps\myapp\current\myapp.exe`)

	if err := Remove("myapp", false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	entries, _ := os.ReadDir(shimsDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "myapp") {
			t.Errorf("leftover: %s", e.Name())
		}
	}
}

func TestRemove_ByContent(t *testing.T) {
	dir := setupTestShims(t)
	shimsDir := filepath.Join(dir, "scoop", "shims")
	os.MkdirAll(shimsDir, 0755)
	writeShim(t, shimsDir, "chrome", `C:\scoop\apps\googlechrome\current\chrome.exe`)

	if err := Remove("googlechrome", false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	entries, _ := os.ReadDir(shimsDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "chrome") {
			t.Errorf("leftover: %s", e.Name())
		}
	}
}

func TestRemove_CmdShimByContent(t *testing.T) {
	dir := setupTestShims(t)
	shimsDir := filepath.Join(dir, "scoop", "shims")
	os.MkdirAll(shimsDir, 0755)
	writeCmdShim(t, shimsDir, "tool", "myapp")

	if err := Remove("myapp", false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	entries, _ := os.ReadDir(shimsDir)
	for _, e := range entries {
		if e.Name() == "tool.cmd" {
			t.Error("tool.cmd was not removed")
		}
	}
}

func TestRemove_PreservesUnrelatedShims(t *testing.T) {
	dir := setupTestShims(t)
	shimsDir := filepath.Join(dir, "scoop", "shims")
	os.MkdirAll(shimsDir, 0755)
	writeShim(t, shimsDir, "chrome", `C:\scoop\apps\googlechrome\current\chrome.exe`)
	writeShim(t, shimsDir, "firefox", `C:\scoop\apps\firefox\current\firefox.exe`)

	if err := Remove("googlechrome", false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shimsDir, "firefox.shim")); os.IsNotExist(err) {
		t.Error("firefox shim was incorrectly removed")
	}
	if _, err := os.Stat(filepath.Join(shimsDir, "firefox.exe")); os.IsNotExist(err) {
		t.Error("firefox exe was incorrectly removed")
	}
}

func TestRemove_NonExistentApp(t *testing.T) {
	dir := setupTestShims(t)
	shimsDir := filepath.Join(dir, "scoop", "shims")
	os.MkdirAll(shimsDir, 0755)
	writeShim(t, shimsDir, "firefox", `C:\scoop\apps\firefox\current\firefox.exe`)

	if err := Remove("nonexistent", false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shimsDir, "firefox.shim")); os.IsNotExist(err) {
		t.Error("firefox shim was incorrectly removed")
	}
}

func TestRemove_EmptyShimsDir(t *testing.T) {
	dir := setupTestShims(t)
	shimsDir := filepath.Join(dir, "scoop", "shims")
	os.MkdirAll(shimsDir, 0755)

	if err := Remove("myapp", false); err != nil {
		t.Fatalf("Remove on empty dir failed: %v", err)
	}
}
