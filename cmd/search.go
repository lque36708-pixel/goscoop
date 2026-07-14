package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ss/internal/bucket"
	"ss/internal/config"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(searchCmd)
}

	var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search available apps",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		cfg := config.Load()
		if _, err := os.Stat(cfg.BucketsDir); os.IsNotExist(err) {
			return fmt.Errorf("no buckets directory at %s", cfg.BucketsDir)
		}

		cachePath := filepath.Join(cfg.CacheDir, "search-index.json")
		entries, err := bucket.LoadSearchIndex(cachePath)
		if err != nil {
			fmt.Printf("%sBuilding search index for the first time...%s\n",
				progress.Cyan+progress.Bold, progress.Reset)
			os.MkdirAll(cfg.CacheDir, 0755)
			if err := bucket.BuildSearchIndex(cfg.BucketsDir, cachePath); err != nil {
				return fmt.Errorf("build search index: %w", err)
			}
			entries, err = bucket.LoadSearchIndex(cachePath)
			if err != nil {
				return fmt.Errorf("load search index: %w", err)
			}
		}
		results := bucket.SearchIndex(entries, query)

		if len(results) == 0 {
			if query != "" {
				fmt.Printf("%sNo results found for%s %s'%s'%s.\n",
					progress.Red, progress.Reset,
					progress.Bold, query, progress.Reset)
			} else {
				fmt.Printf("%sNo apps found.%s\n", progress.Red, progress.Reset)
			}
			return nil
		}

		fmt.Printf("%sResults from local buckets...%s\n\n", progress.Cyan+progress.Bold, progress.Reset)
		for _, r := range results {
			desc := r.Desc
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Printf("  %s%-25s%s %s%-12s%s %s%-12s%s %s%s%s\n",
				progress.Bold, r.Name, progress.Reset,
				progress.Yellow, r.Version, progress.Reset,
				progress.Cyan, r.Bucket, progress.Reset,
				progress.White, desc, progress.Reset)
		}
		return nil
	},
}
