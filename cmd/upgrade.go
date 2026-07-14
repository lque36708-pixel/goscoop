package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(upgradeCmd)
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Update goscoop to the latest version",
	Long:  `Downloads and replaces the current goscoop binary with the latest release from GitHub.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot determine binary path: %w", err)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return err
		}

		url := "https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe"
		fmt.Printf("%sDownloading%s latest goscoop...\n", progress.Cyan+progress.Bold, progress.Reset)

		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: %s", resp.Status)
		}

		newFile := exe + ".new"
		out, err := os.Create(newFile)
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}

		sp := progress.NewSpinner("Downloading")
		sp.Start()
		_, err = io.Copy(out, resp.Body)
		out.Close()
		if err != nil {
			os.Remove(newFile)
			sp.Fail(err.Error())
			return fmt.Errorf("write: %w", err)
		}
		sp.Done("")

		if err := os.Chmod(newFile, 0755); err != nil {
			os.Remove(newFile)
			return fmt.Errorf("chmod: %w", err)
		}

		// Write a bat script that waits and replaces the running binary
		batFile := filepath.Join(os.TempDir(), "goscoop-upgrade.bat")
		batContent := fmt.Sprintf(`@echo off
:retry
timeout /t 1 /nobreak >nul
move /y "%s" "%s" >nul 2>&1
if exist "%s" goto retry
start "" "%s"
del "%%~f0"
`, newFile, exe, newFile, exe)
		if err := os.WriteFile(batFile, []byte(batContent), 0644); err != nil {
			os.Remove(newFile)
			return fmt.Errorf("write upgrade script: %w", err)
		}

		// Launch the bat script and exit
		exec.Command("cmd", "/c", "start", "/min", "", batFile).Start()

		fmt.Printf("%sgoscoop has been updated! Restart your terminal or run the new binary.\n", progress.Green+progress.Bold)
		return nil
	},
}
