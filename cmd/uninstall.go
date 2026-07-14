package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ss/internal/config"
	"ss/internal/persist"
	"ss/internal/progress"
	"ss/internal/shim"

	"github.com/spf13/cobra"
)

var uninstallPurge bool

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallPurge, "purge", "p", false, "Remove persist data as well")
	rootCmd.AddCommand(uninstallCmd)
}

func uninstallApp(app string, cfg *config.Config) error {
	appDir := cfg.AppDir(app)

	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		msg := fmt.Sprintf("'%s' is not installed", app)
		if similar := findSimilarApps(app, cfg.AppsDir); len(similar) > 0 {
			msg += fmt.Sprintf(". Did you mean %s?", similar)
		}
		return fmt.Errorf("%s", msg)
	}

	// Find version from version subdirectories
	version := "?"
	subs, _ := os.ReadDir(appDir)
	for _, sub := range subs {
		if sub.IsDir() && sub.Name() != "current" {
			version = sub.Name()
			break
		}
	}

	fmt.Printf("%sUninstalling%s %s'%s'%s (%s).\n",
		progress.Red+progress.Bold, progress.Reset,
		progress.Bold, app, progress.Reset,
		progress.Yellow+version+progress.Reset)

	// Stop running processes for this app
	stopAppProcesses(app, cfg)

	// Remove Start Menu shortcuts
	man, bucketName, err := findManifest(cfg, app)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not find manifest to remove shortcuts: %v\n", err)
	} else {
		shortcuts := getShortcuts(man)
		if len(shortcuts) == 0 {
			fmt.Printf("No shortcuts defined in manifest for %s (bucket: %s)\n", app, bucketName)
		}
		removeShortcuts(app, shortcuts, false)
	}

	// Remove shims
	shim.Remove(app, false)
	shim.Remove(app, true)

	// Unlink current
	current := cfg.CurrentDir(app)
	if link, err := os.Readlink(current); err == nil {
		if err := os.Remove(current); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not unlink %s: %v\n", current, err)
		} else {
			fmt.Printf("%sUnlinking%s %s => %s\n",
				progress.Cyan, progress.Reset,
				current, link)
		}
	}

	// Remove version dirs
	entries, _ := os.ReadDir(appDir)
	for _, e := range entries {
		if e.Name() == "current" {
			continue
		}
		if err := safeRemoveAll(filepath.Join(appDir, e.Name())); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n",
				filepath.Join(appDir, e.Name()), err)
		}
	}

	// If app dir is empty (only "current" or nothing), remove it
	if remaining, _ := os.ReadDir(appDir); len(remaining) <= 1 {
		if err := safeRemoveAll(appDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", appDir, err)
		}
	}

	// Remove persist data
	persistDir := persist.Dir(app, cfg.ScoopDir)
	if _, err := os.Stat(persistDir); err == nil {
		if uninstallPurge {
			if err := persist.Remove(persistDir); err != nil {
				return fmt.Errorf("removing persist: %w", err)
			}
			fmt.Printf("%sRemoved persist data for%s %s'%s'%s.\n",
				progress.Green, progress.Reset,
				progress.Bold, app, progress.Reset)
		} else {
			fmt.Printf("%sRemove persist data for%s %s'%s'%s? [y/N]: ",
				progress.Yellow, progress.Reset,
				progress.Bold, app, progress.Reset)
			reader := bufio.NewReader(os.Stdin)
			resp, _ := reader.ReadString('\n')
			resp = strings.TrimSpace(resp)
			if strings.EqualFold(resp, "y") || strings.EqualFold(resp, "yes") {
				if err := persist.Remove(persistDir); err != nil {
					return fmt.Errorf("removing persist: %w", err)
				}
				fmt.Printf("%sRemoved persist data for%s %s'%s'%s.\n",
					progress.Green, progress.Reset,
					progress.Bold, app, progress.Reset)
			}
		}
	}

	fmt.Printf("%s'%s'%s was uninstalled.\n",
		progress.Bold, app, progress.Reset)

	// Check if the app binary is still available after uninstall
	if stillAvailable(app) {
		fmt.Fprintf(os.Stderr, "%sNote:%s '%s' is still available on your system. It may have been installed separately (e.g. by the app's own installer or auto-updater).\n",
			progress.Yellow, progress.Reset, app)
	}

	return nil
}

func stillAvailable(app string) bool {
	// Use "where" (Windows equivalent of "which") to check if the binary is still on PATH
	if err := exec.Command("where", app).Run(); err == nil {
		return true
	}
	// Also check common standalone install locations for known apps
	switch strings.ToLower(app) {
	case "googlechrome", "chrome", "google-chrome":
		paths := []string{
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles")+" (x86)", "Google", "Chrome", "Application", "chrome.exe"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	return false
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <app> [apps...]",
	Short: "Uninstall one or more apps",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		for _, app := range args {
			if err := uninstallApp(app, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
		return nil
	},
}

func stopAppProcesses(app string, cfg *config.Config) {
	// Try common binary names based on app name
	targets := []string{app}

	// Try to read manifest for binary names
	if man, _, err := findManifest(cfg, app); err == nil {
		_, bins, _, _ := resolveManifest(man)
		for _, b := range bins {
			name := strings.TrimSuffix(filepath.Base(b), filepath.Ext(b))
			if name != "" {
				targets = append(targets, name)
			}
		}
	}

	for _, name := range targets {
		// taskkill /f /im <name>.exe
		cmd := exec.Command("taskkill", "/f", "/im", name+".exe")
		if output, err := cmd.CombinedOutput(); err == nil {
			fmt.Printf("Stopped %s.exe\n", name)
		} else {
			_ = output // process wasn't running, ignore
		}
	}
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 0; i < la; i++ {
		cur[0] = i + 1
		for j := 0; j < lb; j++ {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			cur[j+1] = min(min(cur[j]+1, prev[j+1]+1), prev[j]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func findSimilarApps(name, appsDir string) string {
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return ""
	}
	var close []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dist := levenshtein(strings.ToLower(name), strings.ToLower(e.Name()))
		if dist > 0 && dist <= 3 {
			close = append(close, "'"+e.Name()+"'")
		}
	}
	if len(close) == 0 {
		return ""
	}
	return strings.Join(close, ", ")
}
