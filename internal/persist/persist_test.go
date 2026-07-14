package persist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	got := Dir("myapp", `C:\scoop`)
	want := `C:\scoop\persist\myapp`
	if got != want {
		t.Errorf("Dir = %q, want %q", got, want)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist", "myapp")
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("EnsureDir did not create directory")
	}
}

func TestSetup_NewPersist(t *testing.T) {
	base := t.TempDir()
	verDir := filepath.Join(base, "apps", "myapp", "1.0")
	persistDir := filepath.Join(base, "persist", "myapp")
	os.MkdirAll(verDir, 0755)
	os.WriteFile(filepath.Join(verDir, "config.cfg"), []byte("data"), 0644)
	entries := []PersistEntry{{Source: "config.cfg", Target: "config.cfg"}}

	if err := Setup("myapp", verDir, persistDir, entries); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(persistDir, "config.cfg")); os.IsNotExist(err) {
		t.Error("persist file not created")
	}
}

func TestCleanup(t *testing.T) {
	base := t.TempDir()
	verDir := filepath.Join(base, "ver")
	os.MkdirAll(verDir, 0755)
	os.WriteFile(filepath.Join(verDir, "config.cfg"), []byte("data"), 0644)
	entries := []PersistEntry{{Source: "config.cfg", Target: "config.cfg"}}

	Cleanup(verDir, entries)
	if _, err := os.Stat(filepath.Join(verDir, "config.cfg")); !os.IsNotExist(err) {
		t.Error("Cleanup did not remove file")
	}
}

func TestRemove(t *testing.T) {
	base := t.TempDir()
	persistDir := filepath.Join(base, "persist", "myapp")
	os.MkdirAll(persistDir, 0755)
	os.WriteFile(filepath.Join(persistDir, "data.txt"), []byte("data"), 0644)

	if err := Remove(persistDir); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if _, err := os.Stat(persistDir); !os.IsNotExist(err) {
		t.Error("Remove did not delete persist dir")
	}
}

func TestRemove_NonExistent(t *testing.T) {
	err := Remove(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Log("Remove on non-existent returned nil (may create dir)")
	}
}
