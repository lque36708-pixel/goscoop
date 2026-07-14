package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ss/internal/config"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(optimizeCmd)
}

var optimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Compact all installed apps with LZX compression",
	Long: `Scans all installed apps and applies LZX compression to save disk space.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		entries, err := os.ReadDir(cfg.AppsDir)
		if err != nil {
			return fmt.Errorf("read apps dir: %w", err)
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

			fmt.Printf("Optimizing %s...\n", entry.Name())
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
			fmt.Printf("\nAll apps optimized: %s -> %s (saved %.1f%%)\n",
				formatSize(totalBefore), formatSize(totalAfter), pct)
		}

		return nil
	},
}
