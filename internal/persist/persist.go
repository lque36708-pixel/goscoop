package persist

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Dir(app string, cfgDir string) string {
	return filepath.Join(cfgDir, "persist", app)
}

func EnsureDir(persistDir string) error {
	return os.MkdirAll(persistDir, 0755)
}

func Setup(app string, versionDir, persistDir string, entries []PersistEntry) error {
	for _, entry := range entries {
		sourcePath := filepath.Join(versionDir, entry.Source)
		targetPath := filepath.Join(persistDir, entry.Target)

		if _, err := os.Stat(targetPath); err == nil {
			// Persist data exists — link it
			if _, err := os.Stat(sourcePath); err == nil {
				backup := sourcePath + ".original"
				os.RemoveAll(backup)
				os.Rename(sourcePath, backup)
			}
			if err := link(sourcePath, targetPath); err != nil {
				return fmt.Errorf("link %s: %w", entry.Source, err)
			}
		} else if _, err := os.Stat(sourcePath); err == nil {
			// Move source to persist dir, then link
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			if err := os.Rename(sourcePath, targetPath); err != nil {
				return fmt.Errorf("move %s: %w", entry.Source, err)
			}
			if err := link(sourcePath, targetPath); err != nil {
				return fmt.Errorf("link after move: %w", err)
			}
		} else {
			// Neither exists — create empty dir in persist, then link
			os.MkdirAll(targetPath, 0755)
			if err := link(sourcePath, targetPath); err != nil {
				return fmt.Errorf("link new: %w", err)
			}
		}
	}
	return nil
}

func Cleanup(versionDir string, entries []PersistEntry) {
	for _, entry := range entries {
		sourcePath := filepath.Join(versionDir, entry.Source)
		os.Remove(sourcePath)
	}
}

func Remove(persistDir string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Remove-Item -LiteralPath '%s' -Recurse -Force", strings.ReplaceAll(persistDir, "'", "''")))
	return cmd.Run()
}

func link(source, target string) error {
	os.MkdirAll(filepath.Dir(source), 0755)

	// Remove stale link if exists
	os.Remove(source)

	info, err := os.Stat(target)
	if err != nil {
		return err
	}

	if info.IsDir() {
		// Directory junction via mklink /J
		absSource, _ := filepath.Abs(source)
		cmd := exec.Command("cmd", "/c", "mklink", "/J", absSource, target)
		return cmd.Run()
	}

	// Hard link for files
	return os.Link(target, source)
}

type PersistEntry struct {
	Source string
	Target string
}
