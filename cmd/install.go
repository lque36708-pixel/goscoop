package cmd

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"ss/internal/bucket"
	"ss/internal/config"
	"ss/internal/download"
	"ss/internal/extract"
	"ss/internal/persist"
	"ss/internal/powershell"
	"ss/internal/progress"

	"github.com/spf13/cobra"
)

var installGlobal bool

func init() {
	installCmd.Flags().BoolVarP(&installGlobal, "global", "g", false, "Install globally")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install <app>",
	Short: "Install an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return installApp(config.Load(), args[0])
	},
}

func installApp(cfg *config.Config, app string) error {
	man, bucketName, err := findManifest(cfg, app)
	if err != nil {
		return err
	}

	// Install dependencies first
	if len(man.Depends) > 0 {
		fmt.Printf("%sInstalling%s dependencies for %s'%s'%s: %s\n",
			progress.Cyan+progress.Bold, progress.Reset,
			progress.Yellow, app, progress.Reset,
			strings.Join(man.Depends, ", "))
		for _, dep := range man.Depends {
			depDir := cfg.AppDir(dep)
			if _, err := os.Stat(depDir); os.IsNotExist(err) {
				fmt.Printf("%sInstalling dependency%s %s'%s'%s...\n",
					progress.Cyan+progress.Bold, progress.Reset,
					progress.Yellow, dep, progress.Reset)
				if err := installApp(cfg, dep); err != nil {
					return fmt.Errorf("dependency '%s': %w", dep, err)
				}
			} else {
				fmt.Printf("%sDependency%s %s'%s'%s is already installed.\n",
					progress.Cyan, progress.Reset,
					progress.Yellow, dep, progress.Reset)
			}
		}
	}

	version := man.Version
	urls, binList, extractDir, isInno := resolveManifest(man)

	fmt.Printf("%sInstalling%s %s'%s'%s (%s) %s[64bit]%s from %s'%s'%s bucket\n",
		progress.Cyan+progress.Bold, progress.Reset,
		progress.Bold, app, progress.Reset,
		version,
		progress.Cyan, progress.Reset,
		progress.Yellow, bucketName, progress.Reset)

	verDir := cfg.VersionDir(app, version)
	safeRemoveAll(verDir)
	os.MkdirAll(verDir, 0755)

	scoopDir := cfg.ScoopDir
	if installGlobal {
		scoopDir = filepath.Join(os.Getenv("ProgramData"), "scoop")
	}
	persistDir := persist.Dir(app, scoopDir)

	// Download and extract each URL
	var firstFilename string
	for i, ui := range urls {
		filename, err := downloadAppFile(cfg, ui, verDir)
		if err != nil {
			return err
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

		// Extract (or run InnoSetup unpack)
		if i == 0 && isInno {
			if err := extract.InnoSetup(filename, verDir, extractDir); err != nil {
				return fmt.Errorf("innosetup extract: %w", err)
			}
		} else {
			if err := extract.Archive(filename, verDir, extractDir); err != nil {
				return fmt.Errorf("extract: %w", err)
			}
		}
	}

	// pre_install
	psVars := powershell.Vars{
		Dir:          verDir,
		PersistDir:   persistDir,
		App:          app,
		Version:      version,
		Bucket:       bucketName,
		BucketsDir:   cfg.BucketsDir,
		Architecture: "64bit",
		Fname:        filepath.Base(firstFilename),
	}
	if err := powershell.RunScripts(man.PreInstall, psVars); err != nil {
		return fmt.Errorf("pre_install: %w", err)
	}

	// Persist data
	if len(man.Persist) > 0 {
		persistEntries := make([]persist.PersistEntry, len(man.Persist))
		for i, p := range man.Persist {
			persistEntries[i] = persist.PersistEntry{Source: p.Source, Target: p.Target}
		}
		persist.EnsureDir(persistDir)
		if err := persist.Setup(app, verDir, persistDir, persistEntries); err != nil {
			return fmt.Errorf("persist: %w", err)
		}
	}

	// Create shims for each binary
	for _, binRel := range binList {
		sp := progress.NewSpinner(fmt.Sprintf("Linking %s", filepath.Base(binRel)))
		sp.Start()
		if err := createShim(app, verDir, binRel); err != nil {
			sp.Fail(err.Error())
			return err
		}
		sp.Done("")
	}

	// post_install
	if err := powershell.RunScripts(man.PostInstall, psVars); err != nil {
		return fmt.Errorf("post_install: %w", err)
	}

	// Create Start Menu shortcuts
	shortcuts := getShortcuts(man)
	createShortcuts(app, shortcuts, verDir, installGlobal)

	// Link current
	linkCurrent(cfg, app, version)
	writeInstallInfo(cfg, app, version, bucketName)

	// LZX compression
	compressLZX(app, verDir)

	fmt.Printf("%s'%s'%s (%s%s%s) was installed successfully!\n",
		progress.Green+progress.Bold, app, progress.Reset,
		progress.Bold, version, progress.Reset)

	// Show notes
	if len(man.Notes) > 0 {
		fmt.Printf("\n%sNotes:%s\n", progress.Cyan+progress.Bold, progress.Reset)
		for _, n := range man.Notes {
			if strings.TrimSpace(n) != "" {
				fmt.Printf("  %s\n", n)
			}
		}
	}
	return nil
}

type urlInfo struct {
	URL      string
	Hash     string
	IsInno   bool
}

func resolveManifest(man *bucket.Manifest) ([]urlInfo, []string, string, bool) {
	extractDir := ""
	isInno := man.InnoSetup
	var urls []urlInfo
	var bins []string

	if man.Architecture != nil {
		if arch, ok := (*man.Architecture)["64bit"]; ok {
			if arch.ExtractDir != "" {
				extractDir = arch.ExtractDir
			} else {
				extractDir = man.ExtractDir
			}
			if arch.InnoSetup != nil {
				isInno = *arch.InnoSetup
			}
			bins = resolveBins(arch.Bin, man.Bin)
			urls = buildURLs(arch.URL, arch.Hash, man.Hash, isInno)
			return urls, bins, extractDir, isInno
		}
	}

	bins = resolveBins(man.Bin, nil)
	if man.ExtractDir != "" {
		extractDir = man.ExtractDir
	}
	urls = buildURLs(man.URL, man.Hash, nil, isInno)
	return urls, bins, extractDir, isInno
}

func buildURLs(urls bucket.StringOrArray, hashes, fallbackHashes bucket.StringOrArray, isInno bool) []urlInfo {
	var result []urlInfo
	for i, raw := range urls {
		clean := raw
		if idx := strings.Index(raw, "#"); idx >= 0 {
			clean = raw[:idx]
		}
		hash := ""
		if i < len(hashes) && hashes[i] != "" {
			hash = hashes[i]
		} else if len(fallbackHashes) > 0 && i < len(fallbackHashes) {
			hash = fallbackHashes[i]
		}
		result = append(result, urlInfo{URL: clean, Hash: hash, IsInno: isInno})
	}
	if len(result) == 0 {
		result = append(result, urlInfo{URL: "", Hash: "", IsInno: isInno})
	}
	return result
}

func resolveBins(bin, fallback bucket.BinList) []string {
	if len(bin) > 0 {
		return bin.Paths()
	}
	if len(fallback) > 0 {
		return fallback.Paths()
	}
	return nil
}

func findManifest(cfg *config.Config, app string) (*bucket.Manifest, string, error) {
	entries, err := os.ReadDir(cfg.BucketsDir)
	if err != nil {
		return nil, "", fmt.Errorf("read buckets: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bucketName := entry.Name()
		manifestDir := filepath.Join(cfg.BucketsDir, bucketName, "bucket")
		if _, err := os.Stat(manifestDir); os.IsNotExist(err) {
			manifestDir = filepath.Join(cfg.BucketsDir, bucketName)
		}
		manifestPath := filepath.Join(manifestDir, app+".json")
		if _, err := os.Stat(manifestPath); err == nil {
			man, err := bucket.ReadManifest(manifestPath)
			if err != nil {
				return nil, "", fmt.Errorf("parse manifest: %w", err)
			}
			return man, bucketName, nil
		}
	}
	return nil, "", fmt.Errorf("couldn't find manifest for '%s'", app)
}

func downloadAppFile(cfg *config.Config, ui urlInfo, verDir string) (string, error) {
	if ui.URL == "" {
		return "", fmt.Errorf("no download URL")
	}
	cachePath := download.CachePath(cfg.CacheDir, ui.URL)
	if _, err := os.Stat(cachePath); err == nil {
		fmt.Println("Loading", filepath.Base(cachePath), "from cache.")
		return cachePath, nil
	}
	return cachePath, download.File(ui.URL, cachePath)
}

func verifyHash(path, expectedHash string) error {
	expectedHash = strings.ToLower(strings.TrimSpace(expectedHash))
	if expectedHash == "" {
		return nil
	}

	var hashFunc func() hash.Hash
	hashStr := expectedHash

	switch {
	case strings.HasPrefix(expectedHash, "sha256:"):
		hashFunc = sha256.New
		hashStr = expectedHash[7:]
	case strings.HasPrefix(expectedHash, "sha1:"):
		hashFunc = sha1.New
		hashStr = expectedHash[5:]
	case strings.HasPrefix(expectedHash, "md5:"):
		hashFunc = md5.New
		hashStr = expectedHash[4:]
	default:
		hashFunc = sha256.New
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := hashFunc()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != hashStr {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, got)
	}
	return nil
}

func createShim(app, dir, binRel string) error {
	cfg := config.Load()
	shimsDir := filepath.Join(cfg.ScoopDir, "shims")
	os.MkdirAll(shimsDir, 0755)

	binPath := filepath.Join(dir, binRel)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		binPath = findBinary(dir, binRel)
		if binPath == "" {
			return fmt.Errorf("binary not found: %s", binRel)
		}
	}

	base := strings.TrimSuffix(filepath.Base(binRel), filepath.Ext(binRel))
	ext := strings.ToLower(filepath.Ext(binPath))

	if ext == ".exe" || ext == ".com" {
		shimFile := filepath.Join(shimsDir, base+".shim")
		shimContent := fmt.Sprintf("path = \"%s\"\r\n", binPath)
		if err := os.WriteFile(shimFile, []byte(shimContent), 0644); err != nil {
			return err
		}
		shimExe := filepath.Join(shimsDir, base+".exe")
		if _, err := os.Stat(shimExe); os.IsNotExist(err) {
			scoopShim := filepath.Join(cfg.ScoopDir, "apps", "scoop", "current",
				"supporting", "shims", "71", "shim.exe")
			if data, err := os.ReadFile(scoopShim); err == nil {
				os.WriteFile(shimExe, data, 0755)
			}
		}
	} else {
		shimPath := filepath.Join(shimsDir, base+".cmd")
		content := "@echo off\r\n" +
			`"%~dp0..\apps\` + app + `\current\` + binRel + `" %*` + "\r\n"
		if err := os.WriteFile(shimPath, []byte(content), 0755); err != nil {
			return err
		}
	}
	return nil
}

func findBinary(dir, binRel string) string {
	baseName := strings.ToLower(filepath.Base(binRel))
	ext := strings.ToLower(filepath.Ext(binRel))

	var best string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		lower := strings.ToLower(filepath.Base(path))
		if lower == baseName {
			best = path
			return filepath.SkipAll
		}
		if strings.HasSuffix(lower, ext) && strings.Contains(lower, strings.TrimSuffix(baseName, ext)) {
			best = path
		}
		return nil
	})
	return best
}

func linkCurrent(cfg *config.Config, app, version string) {
	current := cfg.CurrentDir(app)
	verDir := cfg.VersionDir(app, version)
	os.Remove(current)
	os.Symlink(verDir, current)
	fmt.Printf("Linking %s => %s\n", current, verDir)
}

func writeInstallInfo(cfg *config.Config, app, version, bucket string) {
	info := fmt.Sprintf(`{
    "bucket": "%s",
    "architecture": "64bit"
}
`, bucket)
	current := cfg.CurrentDir(app)
	os.WriteFile(filepath.Join(current, "install.json"), []byte(info), 0644)
}

func getShortcuts(man *bucket.Manifest) []bucket.ShortcutEntry {
	if man.Architecture != nil {
		if arch, ok := (*man.Architecture)["64bit"]; ok && len(arch.Shortcuts) > 0 {
			return []bucket.ShortcutEntry(arch.Shortcuts)
		}
	}
	return []bucket.ShortcutEntry(man.Shortcuts)
}

func createShortcuts(app string, shortcuts []bucket.ShortcutEntry, verDir string, global bool) {
	if len(shortcuts) == 0 {
		return
	}

	startMenuDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Scoop Apps")
	if global {
		startMenuDir = filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs", "Scoop Apps")
	}
	os.MkdirAll(startMenuDir, 0755)

	for _, sc := range shortcuts {
		targetPath := filepath.Join(verDir, sc.Target)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			found := findBinary(verDir, sc.Target)
			if found == "" {
				continue
			}
			targetPath = found
		}

		sp := progress.NewSpinner(fmt.Sprintf("Creating shortcut for %s", sc.Name))
		sp.Start()

		name := sc.Name
		subDir := filepath.Dir(name)
		shortcutDir := startMenuDir
		if subDir != "." {
			shortcutDir = filepath.Join(startMenuDir, subDir)
			os.MkdirAll(shortcutDir, 0755)
			name = filepath.Base(name)
		}

		script := fmt.Sprintf(`
$ws = New-Object -ComObject WScript.Shell
$sc = $ws.CreateShortcut('%s')
$sc.TargetPath = '%s'
$sc.WorkingDirectory = '%s'
$sc.Save()
`, filepath.Join(shortcutDir, name+".lnk"), targetPath, filepath.Dir(targetPath))

		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			sp.Fail(err.Error())
			continue
		}
		sp.Done("")
	}
}

func removeShortcuts(app string, shortcuts []bucket.ShortcutEntry, global bool) {
	if len(shortcuts) == 0 {
		return
	}

	startMenuDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Scoop Apps")
	if global {
		startMenuDir = filepath.Join(os.Getenv("ProgramData"), "Microsoft", "Windows", "Start Menu", "Programs", "Scoop Apps")
	}

	for _, sc := range shortcuts {
		name := sc.Name
		shortcutPath := filepath.Join(startMenuDir, name+".lnk")
		if _, err := os.Stat(shortcutPath); err == nil {
			if err := os.Remove(shortcutPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove shortcut %s: %v\n", shortcutPath, err)
			} else {
				fmt.Printf("Removing shortcut for %s\n", name)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: shortcut not found at %s\n", shortcutPath)
		}
	}
}

func dirSize(dir string) int64 {
	var total int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func compactDirSizes(dir string) (before, after int64) {
	out, err := exec.Command("compact", "/s:"+dir).Output()
	if err != nil {
		s := dirSize(dir)
		return s, s
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == ":" {
			b, err1 := strconv.ParseInt(strings.ReplaceAll(fields[0], ",", ""), 10, 64)
			a, err2 := strconv.ParseInt(strings.ReplaceAll(fields[2], ",", ""), 10, 64)
			if err1 == nil && err2 == nil {
				before += b
				after += a
			}
		}
	}
	if before == 0 {
		s := dirSize(dir)
		return s, s
	}
	return
}

func compressLZX(app, dir string) (currentBefore, currentAfter int64) {
	_, currentBefore = compactDirSizes(dir)
	if currentBefore == 0 {
		return
	}

	sp := progress.NewSpinner(fmt.Sprintf("Compressing %s", app))
	sp.Start()

	cmd := exec.Command("compact", "/c", fmt.Sprintf("/s:%s", dir), "/exe:lzx")
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		sp.Fail("app is running, skipping")
		return
	}
	sp.Done("")

	_, currentAfter = compactDirSizes(dir)
	if currentBefore > currentAfter {
		saved := currentBefore - currentAfter
		pct := float64(saved) / float64(currentBefore) * 100
		fmt.Printf("%sCompressed %s%s: %s -> %s (saved %s%.1f%%%s)\n",
			progress.Green+progress.Bold, progress.Reset,
			app,
			formatSize(currentBefore), formatSize(currentAfter),
			progress.Green, pct, progress.Reset)
	} else {
		currentAfter = currentBefore
		fmt.Printf("%s%s%s is already compressed (%s)\n",
			progress.Bold, app, progress.Reset,
			formatSize(currentBefore))
	}
	return
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20))
	default:
		return fmt.Sprintf("%.1f KB", float64(bytes)/(1<<10))
	}
}
