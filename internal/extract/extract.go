package extract

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ss/internal/progress"
)

func Archive(src, dest, extractDir string) error {
	_ = os.MkdirAll(dest, 0755)

	ext := strings.ToLower(filepath.Ext(src))
	name := filepath.Base(src)

	switch {
	case ext == ".zip":
		return zipArchive(src, dest, extractDir, name)
	case ext == ".7z":
		return sevenZArchive(src, dest, extractDir, name)
	case ext == ".gz" || ext == ".xz" || ext == ".bz2" || ext == ".lzma":
		if strings.HasSuffix(strings.ToLower(src), ".tar.gz") ||
			strings.HasSuffix(strings.ToLower(src), ".tgz") {
			return tarArchive(src, dest, extractDir, name, "gz")
		}
		if strings.HasSuffix(strings.ToLower(src), ".tar.xz") {
			return tarArchive(src, dest, extractDir, name, "xz")
		}
		if strings.HasSuffix(strings.ToLower(src), ".tar.bz2") {
			return tarArchive(src, dest, extractDir, name, "bz2")
		}
		if strings.HasSuffix(strings.ToLower(src), ".tar.lzma") {
			return tarArchive(src, dest, extractDir, name, "lzma")
		}
		return sevenZArchive(src, dest, extractDir, name)
	case ext == ".tar":
		return tarArchive(src, dest, extractDir, name, "")
	case ext == ".msi":
		return msiArchive(src, dest, extractDir, name)
	default:
		return copyAsIs(src, dest, filepath.Base(src))
	}
}

func moveUpAfterExtract(dest, extractDir string) {
	if extractDir == "" {
		return
	}
	srcDir := filepath.Join(dest, extractDir)
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		return
	}
	entries, _ := os.ReadDir(srcDir)
	for _, e := range entries {
		oldPath := filepath.Join(srcDir, e.Name())
		newPath := filepath.Join(dest, e.Name())
		if _, err := os.Stat(newPath); err == nil {
			os.RemoveAll(newPath)
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			data, _ := os.ReadFile(oldPath)
			os.WriteFile(newPath, data, 0755)
		}
	}
	os.RemoveAll(srcDir)
}

func countFiles(dir string) int {
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func zipArchive(src, dest, extractDir, name string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	spinner := progress.NewSpinner(fmt.Sprintf("Extracting %s", name))
	spinner.Start()

	extractPrefix := extractDir
	if extractPrefix != "" && !strings.HasSuffix(extractPrefix, "/") {
		extractPrefix += "/"
	}

	for _, f := range r.File {
		entryName := f.Name
		if extractPrefix != "" {
			if !strings.HasPrefix(entryName, extractPrefix) {
				continue
			}
			entryName = strings.TrimPrefix(entryName, extractPrefix)
		}
		fpath := filepath.Join(dest, entryName)
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0755)
		rc, err := f.Open()
		if err != nil {
			spinner.Fail(err.Error())
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		out, err := os.Create(fpath)
		if err != nil {
			rc.Close()
			spinner.Fail(err.Error())
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			spinner.Fail(err.Error())
			return err
		}
	}

	spinner.Done(fmt.Sprintf("(%d files)", countFiles(dest)))
	return nil
}

func sevenZArchive(src, dest, extractDir, name string) error {
	spinner := progress.NewSpinner(fmt.Sprintf("Extracting %s", name))
	spinner.Start()

	cmd := exec.Command("7z", "x", src, fmt.Sprintf("-o%s", dest), "-y")
	if output, err := cmd.CombinedOutput(); err != nil {
		spinner.Fail(err.Error())
		return fmt.Errorf("7z extract: %w\n%s", err, string(output))
	}

	moveUpAfterExtract(dest, extractDir)

	// Recursively extract any archives left behind (e.g. .tar.lzma -> .tar)
	extractNestedArchives(dest, spinner)

	spinner.Done(fmt.Sprintf("(%d files)", countFiles(dest)))
	return nil
}

func extractNestedArchives(dest string, spinner *progress.Spinner) {
	archiveExts := map[string]bool{
		".tar": true, ".gz": true, ".xz": true, ".bz2": true,
		".lzma": true, ".tgz": true, ".zip": true, ".7z": true,
	}
	filepath.Walk(dest, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == dest {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !archiveExts[ext] {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".tar.lzma") ||
			strings.HasSuffix(strings.ToLower(path), ".tar.gz") ||
			strings.HasSuffix(strings.ToLower(path), ".tar.xz") ||
			strings.HasSuffix(strings.ToLower(path), ".tar.bz2") {
			return nil
		}
		// Try to extract nested archive
		cmd := exec.Command("7z", "x", path, fmt.Sprintf("-o%s", dest), "-y")
		if output, err := cmd.CombinedOutput(); err == nil {
			os.Remove(path)
		} else {
			_ = output
		}
		return nil
	})
}

func tarArchive(src, dest, extractDir, name, compression string) error {
	spinner := progress.NewSpinner(fmt.Sprintf("Extracting %s", name))
	spinner.Start()

	cmd := exec.Command("7z", "x", src, fmt.Sprintf("-o%s", dest), "-y")
	if output, err := cmd.CombinedOutput(); err != nil {
		spinner.Fail(err.Error())
		return fmt.Errorf("tar extract: %w\n%s", err, string(output))
	}

	moveUpAfterExtract(dest, extractDir)
	extractNestedArchives(dest, spinner)
	spinner.Done("")
	return nil
}

func msiArchive(src, dest, extractDir, name string) error {
	spinner := progress.NewSpinner(fmt.Sprintf("Extracting %s", name))
	spinner.Start()

	outDir := dest
	os.MkdirAll(outDir, 0755)

	cmd := exec.Command("msiexec", "/a", src, "/qn", fmt.Sprintf("TARGETDIR=%s", outDir))
	if output, err := cmd.CombinedOutput(); err != nil {
		spinner.Fail(err.Error())
		return fmt.Errorf("msi extract: %w\n%s", err, string(output))
	}

	moveUpAfterExtract(dest, extractDir)
	extractNestedArchives(dest, spinner)
	spinner.Done("")
	return nil
}

func InnoSetup(src, dest, extractDir string) error {
	spinner := progress.NewSpinner(fmt.Sprintf("Extracting %s (innosetup)", filepath.Base(src)))
	spinner.Start()

	// Try innounp first (installed via Scoop)
	innounpPath := findInnounp()
	if innounpPath != "" {
		cmd := exec.Command(innounpPath, "-x", "-d"+dest, "-a", src, "-y")
		if output, err := cmd.CombinedOutput(); err != nil {
			spinner.Fail(string(output))
			// fall through to 7z
		} else {
			moveUpAfterExtract(dest, extractDir)
			extractNestedArchives(dest, spinner)
			spinner.Done(fmt.Sprintf("(%d files)", countFiles(dest)))
			return nil
		}
	}

	// Fallback: try 7z
	cmd := exec.Command("7z", "x", src, fmt.Sprintf("-o%s", dest), "-y")
	if output, err := cmd.CombinedOutput(); err != nil {
		spinner.Fail(string(output))
		return fmt.Errorf("innosetup extract (7z fallback): %w\n%s", err, string(output))
	}

	moveUpAfterExtract(dest, extractDir)
	extractNestedArchives(dest, spinner)
	spinner.Done(fmt.Sprintf("(%d files)", countFiles(dest)))
	return nil
}

func findInnounp() string {
	// Check common locations
	candidates := []string{
		filepath.Join(os.Getenv("USERPROFILE"), "scoop", "apps", "innounp-unicode", "current", "innounp.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "scoop", "apps", "innounp-unicode", "current", "innounp.exe"),
		"innounp.exe",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func copyAsIs(src, dest, filename string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(dest, filename)
	os.MkdirAll(filepath.Dir(dst), 0755)
	return os.WriteFile(dst, data, 0755)
}
