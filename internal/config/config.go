package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ScoopDir  string
	AppsDir   string
	BucketsDir string
	CacheDir  string
}

func Load() *Config {
	scoopDir := os.Getenv("SCOOP")
	if scoopDir == "" {
		home, _ := os.UserHomeDir()
		scoopDir = filepath.Join(home, "scoop")
	}
	return &Config{
		ScoopDir:   scoopDir,
		AppsDir:    filepath.Join(scoopDir, "apps"),
		BucketsDir: filepath.Join(scoopDir, "buckets"),
		CacheDir:   filepath.Join(scoopDir, "cache"),
	}
}

func (c *Config) AppDir(name string) string {
	return filepath.Join(c.AppsDir, name)
}

func (c *Config) VersionDir(name, version string) string {
	return filepath.Join(c.AppsDir, name, version)
}

func (c *Config) CurrentDir(name string) string {
	return filepath.Join(c.AppsDir, name, "current")
}
