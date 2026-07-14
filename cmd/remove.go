package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func safeRemoveAll(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	abs, _ := filepath.Abs(dir)
	// PowerShell Remove-Item handles long paths, junctions, read-only files
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Remove-Item -LiteralPath '%s' -Recurse -Force", strings.ReplaceAll(abs, "'", "''")))
	return cmd.Run()
}
