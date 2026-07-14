package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ss/internal/config"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(holdCmd)
	rootCmd.AddCommand(unholdCmd)
}

var holdCmd = &cobra.Command{
	Use:   "hold <app>",
	Short: "Hold an app (prevent updates)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]
		cfg := config.Load()
		appDir := cfg.AppDir(app)

		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			return fmt.Errorf("'%s' isn't installed", app)
		}

		holdFile := filepath.Join(appDir, ".hold")
		os.WriteFile(holdFile, []byte(""), 0644)
		fmt.Printf("'%s' is now held.\n", app)
		return nil
	},
}

var unholdCmd = &cobra.Command{
	Use:   "unhold <app>",
	Short: "Unhold an app (allow updates)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]
		cfg := config.Load()
		appDir := cfg.AppDir(app)

		holdFile := filepath.Join(appDir, ".hold")
		if err := os.Remove(holdFile); os.IsNotExist(err) {
			return fmt.Errorf("'%s' isn't held", app)
		}
		fmt.Printf("'%s' is now unheld.\n", app)
		return nil
	},
}
