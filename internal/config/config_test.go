package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Default(t *testing.T) {
	os.Unsetenv("SCOOP")
	home, _ := os.UserHomeDir()
	cfg := Load()
	want := filepath.Join(home, "scoop")
	if cfg.ScoopDir != want {
		t.Errorf("ScoopDir = %q, want %q", cfg.ScoopDir, want)
	}
	if cfg.AppsDir != filepath.Join(want, "apps") {
		t.Errorf("AppsDir = %q", cfg.AppsDir)
	}
	if cfg.BucketsDir != filepath.Join(want, "buckets") {
		t.Errorf("BucketsDir = %q", cfg.BucketsDir)
	}
	if cfg.CacheDir != filepath.Join(want, "cache") {
		t.Errorf("CacheDir = %q", cfg.CacheDir)
	}
}

func TestLoad_CustomSCOOP(t *testing.T) {
	os.Setenv("SCOOP", `D:\custom\scoop`)
	defer os.Unsetenv("SCOOP")
	cfg := Load()
	want := `D:\custom\scoop`
	if cfg.ScoopDir != want {
		t.Errorf("ScoopDir = %q, want %q", cfg.ScoopDir, want)
	}
}

func TestAppDir(t *testing.T) {
	cfg := &Config{ScoopDir: `C:\scoop`, AppsDir: `C:\scoop\apps`}
	got := cfg.AppDir("firefox")
	want := `C:\scoop\apps\firefox`
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
}

func TestVersionDir(t *testing.T) {
	cfg := &Config{AppsDir: `C:\scoop\apps`}
	got := cfg.VersionDir("firefox", "100.0")
	want := `C:\scoop\apps\firefox\100.0`
	if got != want {
		t.Errorf("VersionDir = %q, want %q", got, want)
	}
}

func TestCurrentDir(t *testing.T) {
	cfg := &Config{AppsDir: `C:\scoop\apps`}
	got := cfg.CurrentDir("firefox")
	want := `C:\scoop\apps\firefox\current`
	if got != want {
		t.Errorf("CurrentDir = %q, want %q", got, want)
	}
}
