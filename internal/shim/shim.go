package shim

import (
	"os"
	"path/filepath"
	"strings"
)

func Remove(app string, global bool) error {
	shimsDir := shimsDir(global)
	entries, _ := os.ReadDir(shimsDir)
	appPrefix := `apps\` + app + `\`
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(name, app) {
			os.Remove(filepath.Join(shimsDir, name))
			continue
		}
		if strings.HasSuffix(name, ".shim") || strings.HasSuffix(name, ".cmd") {
			data, err := os.ReadFile(filepath.Join(shimsDir, name))
			if err == nil && strings.Contains(string(data), appPrefix) {
				os.Remove(filepath.Join(shimsDir, name))
				if strings.HasSuffix(name, ".shim") {
					base := strings.TrimSuffix(name, ".shim")
					os.Remove(filepath.Join(shimsDir, base+".exe"))
					os.Remove(filepath.Join(shimsDir, base+".ignore"))
				}
			}
		}
	}
	return nil
}

func shimsDir(global bool) string {
	if global {
		return filepath.Join(os.Getenv("ProgramData"), "scoop", "shims")
	}
	scoopDir := os.Getenv("SCOOP")
	if scoopDir == "" {
		home, _ := os.UserHomeDir()
		scoopDir = filepath.Join(home, "scoop")
	}
	return filepath.Join(scoopDir, "shims")
}
