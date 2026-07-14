package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ss/internal/config"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check for outdated apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		entries, err := os.ReadDir(cfg.AppsDir)
		if err != nil {
			return fmt.Errorf("read apps: %w", err)
		}

		fmt.Printf("%sChecking for outdated apps...%s\n", progress.Cyan, progress.Reset)
		found := false

		for _, e := range entries {
			if !e.IsDir() || e.Name() == "scoop" {
				continue
			}
			app := e.Name()

			// Find version from directory
			subs, _ := os.ReadDir(filepath.Join(cfg.AppsDir, app))
			installedVersion := ""
			for _, sub := range subs {
				if sub.IsDir() && sub.Name() != "current" {
					installedVersion = sub.Name()
					break
				}
			}
			if installedVersion == "" {
				continue
			}

			// Find latest version from bucket manifest
			man, _, err := findManifest(cfg, app)
			if err != nil {
				continue
			}

			latestVersion := man.Version
			if latestVersion != "" && latestVersion != installedVersion && compareVersions(latestVersion, installedVersion) > 0 {
				fmt.Printf("  %s%-20s%s %s%-12s%s %s-> %s%s\n",
					progress.Bold, app, progress.Reset,
					progress.Yellow, installedVersion, progress.Reset,
					progress.Green, latestVersion, progress.Reset)
				found = true
			}
		}

		if !found {
			fmt.Printf("  %sAll apps are up to date.%s\n", progress.Green, progress.Reset)
		}

		return nil
	},
}

// compareVersions returns >0 if a > b, <0 if a < b, 0 if equal
func compareVersions(a, b string) int {
	clean := func(v string) string {
		v = strings.TrimLeft(v, "vV")
		return v
	}
	a, b = clean(a), clean(b)

	sa := strings.Split(a, ".")
	sb := strings.Split(b, ".")

	maxLen := len(sa)
	if len(sb) > maxLen {
		maxLen = len(sb)
	}

	for i := 0; i < maxLen; i++ {
		var va, vb int
		if i < len(sa) {
			fmt.Sscanf(sa[i], "%d", &va)
		}
		if i < len(sb) {
			fmt.Sscanf(sb[i], "%d", &vb)
		}
		if va != vb {
			return va - vb
		}
	}
	return 0
}

func init() {
	// sort command list for consistent help output
	sort.Slice(rootCmd.Commands(), func(i, j int) bool {
		return rootCmd.Commands()[i].Use < rootCmd.Commands()[j].Use
	})
}
