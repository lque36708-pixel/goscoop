package bucket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStringOrArrayUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"single string", `"hello"`, []string{"hello"}, false},
		{"string array", `["a","b"]`, []string{"a", "b"}, false},
		{"null", `null`, []string{}, false},
		{"empty", `""`, []string{""}, false},
		{"empty array", `[]`, []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sa StringOrArray
			err := json.Unmarshal([]byte(tt.input), &sa)
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if len(sa) != len(tt.want) {
				t.Errorf("got len %d, want %d", len(sa), len(tt.want))
			}
			for i := range sa {
				if sa[i] != tt.want[i] {
					t.Errorf("got [%d]=%q, want %q", i, sa[i], tt.want[i])
				}
			}
		})
	}
}

func TestDynamicStringArrayUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"single string", `"hello"`, []string{"hello"}, false},
		{"string array", `["a","b"]`, []string{"a", "b"}, false},
		{"null", `null`, []string{}, false},
		{"empty array", `[]`, []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dsa DynamicStringArray
			err := json.Unmarshal([]byte(tt.input), &dsa)
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if len(dsa) != len(tt.want) {
				t.Errorf("got len %d, want %d", len(dsa), len(tt.want))
			}
			for i := range dsa {
				if dsa[i] != tt.want[i] {
					t.Errorf("got [%d]=%q, want %q", i, dsa[i], tt.want[i])
				}
			}
		})
	}
}

func TestPersistListUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []PersistEntry
		wantErr bool
	}{
		{"single string", `"data"`, []PersistEntry{{Source: "data", Target: "data"}}, false},
		{"string array", `["a","b"]`, []PersistEntry{{Source: "a", Target: "a"}, {Source: "b", Target: "b"}}, false},
		{"pair array", `[["src","tgt"]]`, []PersistEntry{{Source: "src", Target: "tgt"}}, false},
		{"mixed", `["a",["b","c"]]`, []PersistEntry{{Source: "a", Target: "a"}, {Source: "b", Target: "c"}}, false},
		{"null", `null`, []PersistEntry{}, false},
		{"empty array", `[]`, []PersistEntry{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pl PersistList
			err := json.Unmarshal([]byte(tt.input), &pl)
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if len(pl) != len(tt.want) {
				t.Errorf("got len %d, want %d", len(pl), len(tt.want))
			}
			for i := range pl {
				if pl[i].Source != tt.want[i].Source || pl[i].Target != tt.want[i].Target {
					t.Errorf("got [%d]={%q,%q}, want {%q,%q}",
						i, pl[i].Source, pl[i].Target, tt.want[i].Source, tt.want[i].Target)
				}
			}
		})
	}
}

func TestBinListUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []BinEntry
		wantErr bool
	}{
		{"single string", `"tool.exe"`, []BinEntry{{Path: "tool.exe"}}, false},
		{"flat array", `["7z.exe","7zG.exe"]`, []BinEntry{{Path: "7z.exe"}, {Path: "7zG.exe"}}, false},
		{"nested", `[["chrome.exe","chrome"]]`, []BinEntry{{Path: "chrome.exe", Name: "chrome"}}, false},
		{"mixed nested", `[["chrome.exe","chrome"],["tool.exe"]]`,
			[]BinEntry{{Path: "chrome.exe", Name: "chrome"}, {Path: "tool.exe"}}, false},
		{"null", `null`, []BinEntry{}, false},
		{"empty array", `[]`, []BinEntry{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bl BinList
			err := json.Unmarshal([]byte(tt.input), &bl)
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if len(bl) != len(tt.want) {
				t.Errorf("got len %d, want %d", len(bl), len(tt.want))
			}
			for i := range bl {
				if bl[i].Path != tt.want[i].Path || bl[i].Name != tt.want[i].Name {
					t.Errorf("got [%d]={%q,%q}, want {%q,%q}",
						i, bl[i].Path, bl[i].Name, tt.want[i].Path, tt.want[i].Name)
				}
			}
		})
	}
}

func TestBinListPaths(t *testing.T) {
	bl := BinList{{"a.exe", ""}, {"b.exe", "b"}}
	paths := bl.Paths()
	if len(paths) != 2 || paths[0] != "a.exe" || paths[1] != "b.exe" {
		t.Errorf("got %v, want [a.exe b.exe]", paths)
	}
}

func TestShortcutListUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []ShortcutEntry
		wantErr bool
	}{
		{"basic", `[["chrome.exe","Google Chrome"]]`,
			[]ShortcutEntry{{Target: "chrome.exe", Name: "Google Chrome"}}, false},
		{"with args", `[["chrome.exe","Chrome","--incognito"]]`,
			[]ShortcutEntry{{Target: "chrome.exe", Name: "Chrome", Arguments: "--incognito"}}, false},
		{"with icon", `[["app.exe","App","","icon.ico"]]`,
			[]ShortcutEntry{{Target: "app.exe", Name: "App", Arguments: "", Icon: "icon.ico"}}, false},
		{"multiple", `[["a","A"],["b","B"]]`,
			[]ShortcutEntry{{Target: "a", Name: "A"}, {Target: "b", Name: "B"}}, false},
		{"null", `null`, nil, false},
		{"empty array", `[]`, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sl ShortcutList
			err := json.Unmarshal([]byte(tt.input), &sl)
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
			if len(sl) != len(tt.want) {
				t.Errorf("got len %d, want %d", len(sl), len(tt.want))
			}
			for i := range sl {
				if sl[i].Target != tt.want[i].Target || sl[i].Name != tt.want[i].Name {
					t.Errorf("got [%d]={%q,%q}, want {%q,%q}",
						i, sl[i].Target, sl[i].Name, tt.want[i].Target, tt.want[i].Name)
				}
			}
		})
	}
}

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	content := `{
		"version": "1.0",
		"description": "test app",
		"homepage": "https://example.com",
		"url": "https://example.com/app.zip",
		"bin": "app.exe",
		"shortcuts": [["app.exe", "My App"]]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	man, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	if man.Version != "1.0" {
		t.Errorf("Version = %q, want %q", man.Version, "1.0")
	}
	if man.Description != "test app" {
		t.Errorf("Description = %q, want %q", man.Description, "test app")
	}
	if len(man.URL) != 1 || man.URL[0] != "https://example.com/app.zip" {
		t.Errorf("URL = %v, want [https://example.com/app.zip]", man.URL)
	}
	if len(man.Bin) != 1 || man.Bin[0].Path != "app.exe" {
		t.Errorf("Bin = %v, want [{app.exe}]", man.Bin)
	}
	if len(man.Shortcuts) != 1 || man.Shortcuts[0].Name != "My App" {
		t.Errorf("Shortcuts = %v, want [{Name: My App}]", man.Shortcuts)
	}
}

func TestReadManifest_NotFound(t *testing.T) {
	_, err := ReadManifest(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadManifest_Architecture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	content := `{
		"version": "2.0",
		"architecture": {
			"64bit": {
				"url": "https://example.com/app64.zip",
				"hash": "abc123",
				"bin": [["app64.exe", "app"]],
				"shortcuts": [["app64.exe", "My App 64"]],
				"innosetup": true
			}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	man, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	if man.Architecture == nil {
		t.Fatal("Architecture is nil")
	}
	arch, ok := (*man.Architecture)["64bit"]
	if !ok {
		t.Fatal("64bit arch not found")
	}
	if len(arch.URL) != 1 || arch.URL[0] != "https://example.com/app64.zip" {
		t.Errorf("arch URL = %v", arch.URL)
	}
	if len(arch.Bin) != 1 || arch.Bin[0].Path != "app64.exe" || arch.Bin[0].Name != "app" {
		t.Errorf("arch Bin = %v", arch.Bin)
	}
	if arch.InnoSetup == nil || !*arch.InnoSetup {
		t.Error("arch InnoSetup should be true")
	}
}

func TestReadManifest_FullGoogleChrome(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "googlechrome.json")
	content := `{
		"version": "150.0.7871.115",
		"description": "Fast, secure, and free web browser",
		"homepage": "https://www.google.com/chrome/",
		"license": "Freeware",
		"architecture": {
			"64bit": {
				"url": "https://dl.google.com/dl/chrome/install/150.0.7871.115_chrome_installer.exe#/dl.7z",
				"hash": "a5792a4fd2757251184bfd49868785c061b125a40c1b09b87f5e44218904b5b4"
			}
		},
		"extract_dir": "Chrome-bin",
		"bin": [["chrome.exe", "chrome"]],
		"shortcuts": [["chrome.exe", "Google Chrome"]],
		"env_set": {
			"CHROME_EXECUTABLE": "$dir\\chrome.exe"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	man, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	if man.Version != "150.0.7871.115" {
		t.Errorf("Version = %q", man.Version)
	}
	if man.ExtractDir != "Chrome-bin" {
		t.Errorf("ExtractDir = %q", man.ExtractDir)
	}
	if len(man.Bin) != 1 || man.Bin[0].Path != "chrome.exe" || man.Bin[0].Name != "chrome" {
		t.Errorf("Bin = %+v", man.Bin)
	}
	if len(man.Shortcuts) != 1 || man.Shortcuts[0].Target != "chrome.exe" {
		t.Errorf("Shortcuts = %+v", man.Shortcuts)
	}
}

func TestReadManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadManifest(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSearchBuckets(t *testing.T) {
	dir := t.TempDir()
	bucketDir := filepath.Join(dir, "main", "bucket")
	os.MkdirAll(bucketDir, 0755)
	app1 := `{"version":"1.0","description":"alpha app"}`
	app2 := `{"version":"2.0","description":"beta app"}`
	os.WriteFile(filepath.Join(bucketDir, "alpha.json"), []byte(app1), 0644)
	os.WriteFile(filepath.Join(bucketDir, "beta.json"), []byte(app2), 0644)
	tests := []struct {
		query    string
		wantN    int
		wantName string
	}{
		{"", 2, ""},
		{"alpha", 1, "alpha"},
		{"beta", 1, "beta"},
		{"gamma", 0, ""},
	}
	for _, tt := range tests {
		results := SearchBuckets(dir, tt.query)
		if len(results) != tt.wantN {
			t.Errorf("SearchBuckets(%q) returned %d results, want %d", tt.query, len(results), tt.wantN)
		}
		if tt.wantN == 1 && len(results) == 1 && results[0].Name != tt.wantName {
			t.Errorf("got name %q, want %q", results[0].Name, tt.wantName)
		}
	}
}
