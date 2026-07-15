package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ss/internal/bucket"
	"ss/internal/config"
	"ss/internal/extract"
	git2 "ss/internal/git"
	"ss/internal/persist"
	"ss/internal/powershell"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update [app]",
	Short: "Update buckets and apps",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		updateBuckets(cfg)

		if len(args) > 0 {
			return updateApp(cfg, args[0])
		}

		// If no app specified, update all outdated apps
		entries, _ := os.ReadDir(cfg.AppsDir)
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "scoop" {
				continue
			}
			app := e.Name()

			// Skip held apps
			holdFile := filepath.Join(cfg.AppsDir, app, ".hold")
			if _, err := os.Stat(holdFile); err == nil {
				continue
			}

			subs, _ := os.ReadDir(filepath.Join(cfg.AppsDir, app))
			installedVersion := ""
			for _, sub := range subs {
				if sub.IsDir() && sub.Name() != "current" {
					installedVersion = sub.Name()
					break
				}
			}
			if installedVersion == "" {
				continue
			}
			man, _, err := findManifest(cfg, app)
			if err != nil {
				continue
			}
			latestVersion := man.Version
			if latestVersion != "" && latestVersion != installedVersion && compareVersions(latestVersion, installedVersion) > 0 {
				if err := updateApp(cfg, app); err != nil {
					fmt.Fprintf(os.Stderr, "  update %s: %s\n", app, err)
				}
			}
		}

		return nil
	},
}

func updateBuckets(cfg *config.Config) error {
	if err := ensureDefaultBuckets(cfg); err != nil {
		fmt.Println(err)
		return nil
	}
	entries, err := os.ReadDir(cfg.BucketsDir)
	if err != nil {
		return fmt.Errorf("read buckets: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bucketDir := filepath.Join(cfg.BucketsDir, entry.Name())
		gitDir := filepath.Join(bucketDir, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			continue
		}

		spinner := progress.NewSpinner(fmt.Sprintf("Updating %s bucket", entry.Name()))
		spinner.Start()

		if err := git2.Pull(bucketDir, nil); err != nil {
			spinner.Fail(err.Error())
			continue
		}
		spinner.Done("")
	}

	// Rebuild search index (non-fatal on failure)
	cachePath := filepath.Join(cfg.CacheDir, "search-index.json")
	if err := bucket.BuildSearchIndex(cfg.BucketsDir, cachePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not rebuild search index: %v\n", err)
	}

	return nil
}

func updateApp(cfg *config.Config, app string) error {
	appDir := cfg.AppDir(app)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return fmt.Errorf("'%s' isn't installed", app)
	}

	// Read current install info
	installInfoPath := filepath.Join(cfg.CurrentDir(app), "install.json")
	currentVersion := ""
	currentBucket := ""
	if data, err := os.ReadFile(installInfoPath); err == nil {
		var info struct {
			Bucket       string `json:"bucket"`
			Architecture string `json:"architecture"`
			Version      string `json:"version"`
		}
		if err := json.Unmarshal(data, &info); err == nil {
			currentBucket = info.Bucket
			currentVersion = info.Version
		}
	}

	// Find version from directory if not in install.json
	if currentVersion == "" {
		subs, _ := os.ReadDir(appDir)
		for _, sub := range subs {
			if sub.IsDir() && sub.Name() != "current" {
				currentVersion = sub.Name()
				break
			}
		}
	}

	// Find latest manifest
	man, bucketName, err := findManifest(cfg, app)
	if err != nil {
		return fmt.Errorf("find manifest: %w", err)
	}

	latestVersion := man.Version
	if currentBucket != "" {
		bucketName = currentBucket
	}

	if latestVersion == currentVersion {
		fmt.Printf("%s'%s'%s is already up to date (version %s).\n",
			progress.Bold, app, progress.Reset,
			progress.Yellow+currentVersion+progress.Reset)
		return nil
	}

	fmt.Printf("%sUpdating%s %s'%s'%s (%s -> %s) from %s'%s'%s bucket\n",
		progress.Cyan+progress.Bold, progress.Reset,
		progress.Bold, app, progress.Reset,
		progress.Yellow+currentVersion+progress.Reset,
		progress.Green+latestVersion+progress.Reset,
		progress.Yellow, bucketName, progress.Reset)

	// Resolve manifest details
	urls, binList, extractDir, isInno := resolveManifest(man)

	// Download latest version
	verDir := cfg.VersionDir(app, latestVersion)
	safeRemoveAll(verDir)
	os.MkdirAll(verDir, 0755)

	var firstFilename string
	for i, ui := range urls {
		filename, err := downloadAppFile(cfg, ui, verDir)
		if err != nil {
			return fmt.Errorf("download: %w", err)
		}
		if i == 0 {
			firstFilename = filename
		}

		// Verify hash
		if ui.Hash != "" {
			if err := verifyHash(filename, ui.Hash); err != nil {
				return err
			}
		}

		// Extract
		if i == 0 && isInno {
			if err := extract.InnoSetup(filename, verDir, extractDir); err != nil {
				return fmt.Errorf("extract (innosetup): %w", err)
			}
		} else {
			if err := extract.Archive(filename, verDir, extractDir); err != nil {
				return fmt.Errorf("extract: %w", err)
			}
		}
	}

	// pre_install
	persistDir := persist.Dir(app, cfg.ScoopDir)
	psVars := powershell.Vars{
		Dir:          verDir,
		PersistDir:   persistDir,
		App:          app,
		Version:      latestVersion,
		BucketsDir:   cfg.BucketsDir,
		Architecture: "64bit",
		Fname:        filepath.Base(firstFilename),
		Bucket:     bucketName,
	}
	if err := powershell.RunScripts(man.PreInstall, psVars); err != nil {
		return fmt.Errorf("pre_install: %w", err)
	}

	// Persist data migration
	if len(man.Persist) > 0 {
		persistEntries := make([]persist.PersistEntry, len(man.Persist))
		for i, p := range man.Persist {
			persistEntries[i] = persist.PersistEntry{Source: p.Source, Target: p.Target}
		}
		persist.EnsureDir(persistDir)

		// Clear any existing persist links in the new version dir
		persist.Cleanup(verDir, persistEntries)

		// Set up persist
		if err := persist.Setup(app, verDir, persistDir, persistEntries); err != nil {
			return fmt.Errorf("persist: %w", err)
		}
	}

	// Create shims
	for _, binRel := range binList {
		sp := progress.NewSpinner(fmt.Sprintf("Linking %s", filepath.Base(binRel)))
		sp.Start()
		if err := createShim(app, verDir, binRel); err != nil {
			sp.Fail(err.Error())
			fmt.Fprintf(os.Stderr, "  warning: %s\n", err)
			continue
		}
		sp.Done("")
	}

	// post_install
	if err := powershell.RunScripts(man.PostInstall, psVars); err != nil {
		return fmt.Errorf("post_install: %w", err)
	}

	// Create Start Menu shortcuts
	shortcuts := getShortcuts(man)
	createShortcuts(app, shortcuts, verDir, false)

	// Unlink old current first, then link new
	current := cfg.CurrentDir(app)
	oldLink, _ := os.Readlink(current)
	os.Remove(current)
	os.Symlink(verDir, current)

	// Write install info (updated version)
	installInfo := fmt.Sprintf(`{
    "bucket": "%s",
    "architecture": "64bit",
    "version": "%s"
}
`, bucketName, latestVersion)
	os.WriteFile(filepath.Join(current, "install.json"), []byte(installInfo), 0644)

	// LZX compression
	compressLZX(app, verDir)

	fmt.Printf("%s'%s'%s was updated successfully!\n",
		progress.Green+progress.Bold, app, progress.Reset)

	// Optionally remove old version
	if oldLink != "" && oldLink != verDir {
		if strings.HasPrefix(oldLink, cfg.VersionDir(app, "")) {
			if _, err := os.Stat(oldLink); err == nil {
				safeRemoveAll(oldLink)
			}
		}
	}

	return nil
}
