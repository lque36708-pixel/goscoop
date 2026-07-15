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
var uninstallSelf bool

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallPurge, "purge", "p", false, "Remove persist data as well")
	uninstallCmd.Flags().BoolVar(&uninstallSelf, "self", false, "Uninstall goscoop and remove all scoop data")
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
	return exec.Command("where", app).Run() == nil
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <app> [apps...]",
	Short: "Uninstall one or more apps",
	Long: `Uninstall one or more apps, or remove goscoop entirely with --self.

Use --self to remove goscoop, all apps, buckets, cache, and persist data.`,
	Args: cobra.MaximumNArgs(100),
	RunE: func(cmd *cobra.Command, args []string) error {
		if uninstallSelf {
			if len(args) > 0 {
				return fmt.Errorf("--self cannot be combined with app names")
			}
			return uninstallSelfCmd()
		}
		if len(args) == 0 {
			return fmt.Errorf("specify an app to uninstall, or use --self to remove goscoop entirely")
		}
		cfg := config.Load()
		for _, app := range args {
			if err := uninstallApp(app, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
		return nil
	},
}

func uninstallSelfCmd() error {
	fmt.Printf("%sWARNING:%s This will remove goscoop and ALL scoop data:\n", progress.Red+progress.Bold, progress.Reset)
	fmt.Printf("  - All installed apps\n")
	fmt.Printf("  - All buckets\n")
	fmt.Printf("  - Download cache\n")
	fmt.Printf("  - Persist data\n")
	fmt.Printf("  - The goscoop binary itself\n")
	fmt.Printf("\n%sAre you sure? [y/N]:%s ", progress.Yellow, progress.Reset)

	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)
	if !strings.EqualFold(resp, "y") && !strings.EqualFold(resp, "yes") {
		fmt.Println("Cancelled.")
		return nil
	}

	cfg := config.Load()

	// Remove scoop root directory
	if _, err := os.Stat(cfg.ScoopDir); err == nil {
		fmt.Printf("Removing %s ...\n", cfg.ScoopDir)
		if err := safeRemoveAll(cfg.ScoopDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", cfg.ScoopDir, err)
		} else {
			fmt.Printf("Removed %s\n", cfg.ScoopDir)
		}
	}

	// Remove the standalone install directory if it exists
	goscoopDir := filepath.Join(os.Getenv("USERPROFILE"), "goscoop")
	if _, err := os.Stat(goscoopDir); err == nil {
		fmt.Printf("Removing %s ...\n", goscoopDir)
		if err := safeRemoveAll(goscoopDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", goscoopDir, err)
		} else {
			fmt.Printf("Removed %s\n", goscoopDir)
		}
	}

	// Remove shims from PATH integration (the goscoop shims dir)
	shimsDir := filepath.Join(os.Getenv("USERPROFILE"), "scoop", "shims")
	if _, err := os.Stat(shimsDir); err == nil {
		if err := safeRemoveAll(shimsDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", shimsDir, err)
		}
	}

	// Remove shims dir from PATH
	psCmd := fmt.Sprintf(
		`$p = [Environment]::GetEnvironmentVariable('PATH', 'User');`+
			` $new = ($p -split ';' | Where-Object { $_ -ne '%s' }) -join ';';`+
			` [Environment]::SetEnvironmentVariable('PATH', $new, 'User')`,
		shimsDir)
	exec.Command("powershell", "-Command", psCmd).Run()

	fmt.Printf("\n%sgoscoop has been removed.%s\n", progress.Green+progress.Bold, progress.Reset)
	fmt.Printf("To clean up your PATH, remove %%USERPROFILE%%\\goscoop from your PATH environment variable.\n")
	return nil
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
