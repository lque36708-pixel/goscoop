package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ss/internal/config"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	bucketCmd.AddCommand(bucketListCmd)
	bucketCmd.AddCommand(bucketAddCmd)
	bucketCmd.AddCommand(bucketRmCmd)
	rootCmd.AddCommand(bucketCmd)
}

var bucketCmd = &cobra.Command{
	Use:   "bucket",
	Short: "Manage buckets",
}

var bucketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed buckets",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		if err := ensureDefaultBuckets(cfg); err != nil {
			fmt.Println(err)
			return nil
		}
		entries, err := os.ReadDir(cfg.BucketsDir)
		if err != nil {
			return fmt.Errorf("read buckets: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No buckets installed.")
			return nil
		}

		fmt.Println("Installed buckets:")
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			// Count manifests
			manifestDir := filepath.Join(cfg.BucketsDir, name, "bucket")
			manifestCount := 0
			if entries2, err := os.ReadDir(manifestDir); err == nil {
				for range entries2 {
					manifestCount++
				}
			}

			var source string
			gitDir := filepath.Join(cfg.BucketsDir, name, ".git")
			if cfgFile, err := os.ReadFile(filepath.Join(gitDir, "config")); err == nil {
				for _, line := range strings.Split(string(cfgFile), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "url = ") {
						source = strings.TrimPrefix(line, "url = ")
						break
					}
				}
			}

			if source != "" {
				fmt.Printf("  %-20s %4d manifests  %s\n", name, manifestCount, source)
			} else {
				fmt.Printf("  %-20s %4d manifests\n", name, manifestCount)
			}
		}
		return nil
	},
}

var bucketAddCmd = &cobra.Command{
	Use:   "add <name> [<repo>]",
	Short: "Add a bucket",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		repo := name
		if len(args) > 1 {
			repo = args[1]
		} else {
			repo = fmt.Sprintf("https://github.com/ScoopInstaller/%s", name)
		}

		cfg := config.Load()
		bucketDir := filepath.Join(cfg.BucketsDir, name)

		if _, err := os.Stat(bucketDir); err == nil {
			return fmt.Errorf("bucket '%s' already exists", name)
		}

		sp := progress.NewSpinner(fmt.Sprintf("Cloning %s bucket", name))
		sp.Start()

		gitCmd := exec.Command("git", "clone", repo, bucketDir)
		if output, err := gitCmd.CombinedOutput(); err != nil {
			sp.Fail(string(output))
			return fmt.Errorf("clone %s: %w", repo, err)
		}

		sp.Done("")
		return nil
	},
}

var bucketRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a bucket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg := config.Load()
		bucketDir := filepath.Join(cfg.BucketsDir, name)

		if _, err := os.Stat(bucketDir); os.IsNotExist(err) {
			return fmt.Errorf("bucket '%s' doesn't exist", name)
		}

		if err := safeRemoveAll(bucketDir); err != nil {
			return fmt.Errorf("removing bucket: %w", err)
		}
		fmt.Printf("Removed bucket '%s'.\n", name)
		return nil
	},
}
