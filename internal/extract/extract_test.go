package extract

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ss/internal/progress"
)

func createZip(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, "test.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	w.Close()
	return path
}

func createTarGz(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, "test.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0644,
		}
		if strings.HasSuffix(name, "/") {
			hdr.Typeflag = tar.TypeDir
			hdr.Size = 0
		}
		tw.WriteHeader(hdr)
		if !strings.HasSuffix(name, "/") {
			tw.Write([]byte(content))
		}
	}
	tw.Close()
	gw.Close()
	return path
}

func TestCopyAsIs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	content := []byte("hello world")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out")
	if err := copyAsIs(src, dest, "result.bin"); err != nil {
		t.Fatalf("copyAsIs failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "result.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %q, want %q", data, content)
	}
}

func TestCopyAsIs_NestedDest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	os.WriteFile(src, []byte("data"), 0644)
	dest := filepath.Join(dir, "a", "b")
	if err := copyAsIs(src, dest, "out.bin"); err != nil {
		t.Fatalf("copyAsIs to nested dir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "out.bin")); os.IsNotExist(err) {
		t.Error("copyAsIs did not create nested directory")
	}
}

func TestMoveUpAfterExtract(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("a"), 0644)
	os.MkdirAll(filepath.Join(subDir, "nested"), 0755)
	os.WriteFile(filepath.Join(subDir, "nested", "file2.txt"), []byte("b"), 0644)

	moveUpAfterExtract(dir, "sub")

	if _, err := os.Stat(filepath.Join(dir, "file1.txt")); os.IsNotExist(err) {
		t.Error("file1.txt was not moved up")
	}
	if _, err := os.Stat(filepath.Join(dir, "nested", "file2.txt")); os.IsNotExist(err) {
		t.Error("nested/file2.txt was not moved up")
	}
	if _, err := os.Stat(filepath.Join(dir, "sub")); !os.IsNotExist(err) {
		t.Error("sub dir was not removed")
	}
}

func TestMoveUpAfterExtract_NoExtractDir(t *testing.T) {
	dir := t.TempDir()
	moveUpAfterExtract(dir, "") // should not panic
}

func TestMoveUpAfterExtract_Nonexistent(t *testing.T) {
	dir := t.TempDir()
	moveUpAfterExtract(dir, "nonexistent") // should not panic
}

func TestCountFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "c.txt"), []byte("c"), 0644)

	if n := countFiles(dir); n != 3 {
		t.Errorf("countFiles = %d, want 3", n)
	}
}

func TestCountFiles_Empty(t *testing.T) {
	if n := countFiles(t.TempDir()); n != 0 {
		t.Errorf("countFiles empty = %d, want 0", n)
	}
}

func TestZipArchive(t *testing.T) {
	dir := t.TempDir()
	src := createZip(t, dir, map[string]string{
		"file1.txt": "hello",
		"sub/file2.txt": "world",
	})
	dest := filepath.Join(dir, "output")
	if err := zipArchive(src, dest, "", "test.zip"); err != nil {
		t.Fatalf("zipArchive failed: %v", err)
	}
	data1, _ := os.ReadFile(filepath.Join(dest, "file1.txt"))
	if string(data1) != "hello" {
		t.Errorf("file1.txt = %q, want %q", data1, "hello")
	}
	data2, _ := os.ReadFile(filepath.Join(dest, "sub", "file2.txt"))
	if string(data2) != "world" {
		t.Errorf("sub/file2.txt = %q, want %q", data2, "world")
	}
}

func TestZipArchive_WithExtractDir(t *testing.T) {
	dir := t.TempDir()
	src := createZip(t, dir, map[string]string{
		"Chrome-bin/chrome.exe": "binary",
		"Chrome-bin/version":    "150",
	})
	dest := filepath.Join(dir, "output")
	if err := zipArchive(src, dest, "Chrome-bin", "chrome.zip"); err != nil {
		t.Fatalf("zipArchive with extractDir failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dest, "chrome.exe"))
	if string(data) != "binary" {
		t.Errorf("chrome.exe = %q, want %q", string(data), "binary")
	}
	// Ensure extractDir prefix is stripped
	if _, err := os.Stat(filepath.Join(dest, "Chrome-bin", "chrome.exe")); !os.IsNotExist(err) {
		t.Error("Chrome-bin prefix should be stripped")
	}
}

func TestZipArchive_InvalidZip(t *testing.T) {
	dir := t.TempDir()
	badZip := filepath.Join(dir, "bad.zip")
	os.WriteFile(badZip, []byte("not a zip"), 0644)
	err := zipArchive(badZip, filepath.Join(dir, "out"), "", "bad.zip")
	if err == nil {
		t.Error("expected error for invalid zip")
	}
}

func TestArchive_Routing_Zip(t *testing.T) {
	dir := t.TempDir()
	src := createZip(t, dir, map[string]string{"test.txt": "content"})
	dest := filepath.Join(dir, "out")
	if err := Archive(src, dest, ""); err != nil {
		t.Fatalf("Archive(zip) failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "test.txt")); os.IsNotExist(err) {
		t.Error("zip archive extraction did not produce file")
	}
}

func TestArchive_Routing_CopyAsIs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "script.exe")
	os.WriteFile(src, []byte("exe data"), 0644)
	dest := filepath.Join(dir, "out")
	if err := Archive(src, dest, ""); err != nil {
		t.Fatalf("Archive(exe) failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dest, "script.exe"))
	if string(data) != "exe data" {
		t.Errorf("copyAsIs content = %q", data)
	}
}

func TestArchive_Routing_TarGz(t *testing.T) {
	dir := t.TempDir()
	src := createTarGz(t, dir, map[string]string{
		"app.bin": "tarcontent",
	})
	dest := filepath.Join(dir, "out")
	// This will try '7z x', which may not be installed — expect error, not panic
	err := Archive(src, dest, "")
	if err != nil {
		t.Logf("tar.gz extraction (requires 7z): %v (expected if 7z not installed)", err)
	}
}

func TestFindInnounp_Default(t *testing.T) {
	path := findInnounp()
	// Should always return "" in test env (no innounp installed via scoop)
	t.Logf("findInnounp returned: %q", path)
}

func TestArchive_NonexistentSrc(t *testing.T) {
	err := Archive(filepath.Join(t.TempDir(), "nothing.zip"), t.TempDir(), "")
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestExtractNestedArchives_Noop(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	s := progress.NewSpinner("test")
	extractNestedArchives(dir, s)
	s.Done("")
}

func TestMoveUpAfterExtract_SrcDirHasNoFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)
	moveUpAfterExtract(dir, "empty")
	// Should remove empty dir
	if _, err := os.Stat(filepath.Join(dir, "empty")); !os.IsNotExist(err) {
		t.Error("empty extract dir should be removed")
	}
}
