package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ss/internal/config"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		entries, err := os.ReadDir(cfg.AppsDir)
		if err != nil {
			fmt.Println("No apps installed yet.")
			return nil
		}

		if len(entries) == 0 {
			fmt.Println("No apps installed yet.")
			return nil
		}

		fmt.Printf("%sInstalled apps:%s\n\n", progress.Cyan+progress.Bold, progress.Reset)

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			version := "?"
			bucket := ""

			// Read version from version subdirectories
			subs, _ := os.ReadDir(filepath.Join(cfg.AppsDir, name))
			for _, sub := range subs {
				if sub.IsDir() && sub.Name() != "current" {
					version = sub.Name()
					break
				}
			}

			// Read bucket from install.json if exists
			installInfo := filepath.Join(cfg.AppsDir, name, "current", "install.json")
			if data, err := os.ReadFile(installInfo); err == nil {
				content := string(data)
				if idx := strings.Index(content, `"bucket"`); idx >= 0 {
					after := content[idx+len(`"bucket"`):]
					if colon := strings.IndexByte(after, ':'); colon >= 0 {
						val := strings.TrimSpace(after[colon+1:])
						if len(val) > 0 && val[0] == '"' {
							val = val[1:]
						}
						if end := strings.IndexByte(val, '"'); end >= 0 {
							bucket = val[:end]
						}
					}
				}
			}

			fmt.Printf("  %s%-20s%s %s%-12s%s %s%s%s\n",
				progress.Bold, name, progress.Reset,
				progress.Yellow, version, progress.Reset,
				progress.Cyan, bucket, progress.Reset)
		}
		return nil
	},
}
