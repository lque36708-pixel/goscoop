package cmd

import (
	"fmt"
	"os"

	"ss/internal/config"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset <app>",
	Short: "Reset an app (reinstall shims)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]
		cfg := config.Load()
		appDir := cfg.AppDir(app)

		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			return fmt.Errorf("'%s' isn't installed", app)
		}

		// Find version
		subs, _ := os.ReadDir(appDir)
		version := ""
		for _, sub := range subs {
			if sub.IsDir() && sub.Name() != "current" {
				version = sub.Name()
				break
			}
		}
		if version == "" {
			return fmt.Errorf("no version found for '%s'", app)
		}

		// Read manifest
		man, _, err := findManifest(cfg, app)
		if err != nil {
			return fmt.Errorf("find manifest: %w", err)
		}

		_, bins, _, _ := resolveManifest(man)
		if len(bins) == 0 {
			return fmt.Errorf("no bins defined for '%s'", app)
		}

		verDir := cfg.VersionDir(app, version)
		fmt.Printf("Resetting '%s' (%s)\n", app, version)

		for _, binRel := range bins {
			sp := progress.NewSpinner(fmt.Sprintf("Linking %s", binRel))
			sp.Start()
			if err := createShim(app, verDir, binRel); err != nil {
				sp.Fail(err.Error())
				return err
			}
			sp.Done("")
		}

		fmt.Printf("'%s' was reset successfully.\n", app)
		return nil
	},
}
