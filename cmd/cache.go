package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ss/internal/config"

	"github.com/spf13/cobra"
)

func init() {
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheRmCmd)
	rootCmd.AddCommand(cacheCmd)
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage download cache",
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached downloads",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		entries, err := os.ReadDir(cfg.CacheDir)
		if err != nil {
			return fmt.Errorf("read cache: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("Cache is empty.")
			return nil
		}

		type cacheEntry struct {
			file string
			size int64
			time time.Time
		}

		var apps []cacheEntry
		var totalSize int64
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			apps = append(apps, cacheEntry{
				file: e.Name(),
				size: info.Size(),
				time: info.ModTime(),
			})
			totalSize += info.Size()
		}

		sort.Slice(apps, func(i, j int) bool {
			return apps[i].time.Before(apps[j].time)
		})

		fmt.Printf("Total cached: %s (%d files)\n\n", formatSize(totalSize), len(apps))
		for _, a := range apps {
			appName := extractAppName(a.file)
			fmt.Printf("  %-30s %-10s  %s\n", appName, formatSize(a.size), a.time.Format("2006-01-02 15:04"))
		}
		return nil
	},
}

var cacheRmCmd = &cobra.Command{
	Use:   "rm [app]",
	Short: "Remove cached downloads",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		if len(args) == 0 {
			// Remove all cache
			entries, _ := os.ReadDir(cfg.CacheDir)
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				os.Remove(filepath.Join(cfg.CacheDir, e.Name()))
			}
			fmt.Println("Cache cleared.")
			return nil
		}

		app := args[0]
		entries, _ := os.ReadDir(cfg.CacheDir)
		removed := 0
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasPrefix(strings.ToLower(e.Name()), strings.ToLower(app)) {
				os.Remove(filepath.Join(cfg.CacheDir, e.Name()))
				removed++
			}
		}

		if removed == 0 {
			return fmt.Errorf("no cached files found for '%s'", app)
		}
		fmt.Printf("Removed %d cached file(s) for '%s'.\n", removed, app)
		return nil
	},
}

func extractAppName(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Handle Scoop hash-based cache files
	if strings.Contains(name, "#") {
		parts := strings.SplitN(name, "#", 3)
		if parts[0] != "" {
			return parts[0]
		}
	}

	// Remove suffixes like -win64, -x64, -portable, -msys etc.
	trimmed := name
	for _, suffix := range []string{"-win64", "-win32", "-x64", "-x86", "-amd64", "-portable", "-msys"} {
		if strings.HasSuffix(trimmed, suffix) {
			trimmed = strings.TrimSuffix(trimmed, suffix)
		}
	}

	// Try to strip version number (segment starting with digit)
	parts := strings.Split(trimmed, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) > 0 && parts[i][0] >= '0' && parts[i][0] <= '9' {
			trimmed = strings.Join(parts[:i], "-")
			break
		}
	}

	if trimmed == "" {
		return name // fallback to raw filename
	}
	return trimmed
}
