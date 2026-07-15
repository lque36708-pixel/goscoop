package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ss/internal/config"

	"github.com/spf13/cobra"
)

var compressAll bool

func init() {
	compressCmd.Flags().BoolVarP(&compressAll, "all", "a", false, "Compress all installed apps")
	rootCmd.AddCommand(compressCmd)
}

var compressCmd = &cobra.Command{
	Use:   "compress [app]",
	Short: "Compact installed apps with LZX compression to save disk space",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		if len(args) == 1 {
			return compressApp(cfg, args[0])
		}

		if !compressAll {
			return fmt.Errorf("specify an app or use --all to compress all apps")
		}

		// Compress all apps
		entries, err := os.ReadDir(cfg.AppsDir)
		if err != nil {
			fmt.Println("No apps installed yet.")
			return nil
		}

		var totalBefore, totalAfter int64
		for _, entry := range entries {
			if !entry.IsDir() || entry.Name() == "scoop" {
				continue
			}

			appDir := filepath.Join(cfg.AppsDir, entry.Name())
			subs, err := os.ReadDir(appDir)
			if err != nil {
				continue
			}

			for _, sub := range subs {
				if !sub.IsDir() || sub.Name() == "current" {
					continue
				}
				verDir := filepath.Join(appDir, sub.Name())
				before, after := compressLZX(entry.Name(), verDir)
				totalBefore += before
				totalAfter += after
			}
		}

		if totalBefore > 0 {
			saved := totalBefore - totalAfter
			pct := float64(saved) / float64(totalBefore) * 100
			fmt.Printf("\nAll apps compressed: %s -> %s (saved %.1f%%)\n",
				formatSize(totalBefore), formatSize(totalAfter), pct)
		}

		return nil
	},
}

func compressApp(cfg *config.Config, app string) error {
	appDir := cfg.AppDir(app)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return fmt.Errorf("'%s' isn't installed", app)
	}

	subs, err := os.ReadDir(appDir)
	if err != nil {
		return err
	}

	found := false
	for _, sub := range subs {
		if !sub.IsDir() || sub.Name() == "current" {
			continue
		}
		verDir := filepath.Join(appDir, sub.Name())
		compressLZX(app, verDir)
		found = true
	}

	if !found {
		return fmt.Errorf("no version directory found for '%s'", app)
	}
	return nil
}
