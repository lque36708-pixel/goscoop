package powershell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildScript(t *testing.T) {
	vars := Vars{
		Dir:         `C:\scoop\apps\myapp\1.0`,
		PersistDir:  `C:\scoop\persist\myapp`,
		App:         "myapp",
		Version:     "1.0",
		Bucket:      "main",
		BucketsDir:  `C:\scoop\buckets`,
		Architecture: "64bit",
		Global:      false,
		CleanVersion: "1.0",
		Fname:       "myapp-1.0.zip",
	}
	script := buildScript("Write-Host 'hello'", vars)
	if !strings.Contains(script, `$dir = 'C:\scoop\apps\myapp\1.0'`) {
		t.Error("script missing $dir variable")
	}
	if !strings.Contains(script, `$persist_dir = 'C:\scoop\persist\myapp'`) {
		t.Error("script missing $persist_dir variable")
	}
	if !strings.Contains(script, `$version = "1.0"`) {
		t.Error("script missing $version variable")
	}
	if !strings.Contains(script, "Write-Host 'hello'") {
		t.Error("script missing original content")
	}
}

func TestEscApos(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"it's", "it''s"},
		{"'quoted'", "''quoted''"},
	}
	for _, tt := range tests {
		got := escApos(tt.input)
		if got != tt.want {
			t.Errorf("escApos(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestQuotePath(t *testing.T) {
	got := quotePath(`C:\path\to\app`)
	want := "'C:\\path\\to\\app'"
	if got != want {
		t.Errorf("quotePath = %q, want %q", got, want)
	}
	got = quotePath(`C:\path with spaces\app`)
	want = "'C:\\path with spaces\\app'"
	if got != want {
		t.Errorf("quotePath with space = %q, want %q", got, want)
	}
	got = quotePath(`C:\it's\app`)
	want = "'C:\\it''s\\app'"
	if got != want {
		t.Errorf("quotePath with apostrophe = %q, want %q", got, want)
	}
}

func TestRunScript_Simple(t *testing.T) {
	vars := Vars{
		Dir:        t.TempDir(),
		PersistDir: t.TempDir(),
	}
	err := RunScript("Write-Host 'test ok'", vars)
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
}

func TestRunScript_Error(t *testing.T) {
	vars := Vars{
		Dir:        t.TempDir(),
		PersistDir: t.TempDir(),
	}
	err := RunScript("throw 'test error'", vars)
	if err == nil {
		t.Fatal("expected error from throw")
	}
}

func TestRunScript_FileExists(t *testing.T) {
	dir := t.TempDir()
	vars := Vars{
		Dir:        dir,
		PersistDir: dir,
	}
	err := RunScript("New-Item -Path (Join-Path $dir 'test.txt') -ItemType File -Force > $null", vars)
	if err != nil {
		t.Fatalf("RunScript failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "test.txt")); os.IsNotExist(err) {
		t.Error("script did not create file")
	}
}

func TestRunScripts_Multiple(t *testing.T) {
	vars := Vars{
		Dir:        t.TempDir(),
		PersistDir: t.TempDir(),
	}
	err := RunScripts([]string{
		"$x = 1",
		"$y = 2",
		"$z = $x + $y",
	}, vars)
	if err != nil {
		t.Fatalf("RunScripts failed: %v", err)
	}
}
