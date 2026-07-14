package cmd

import (
	"testing"

	"ss/internal/bucket"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0.0 KB"},
		{1023, "1.0 KB"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0", 0},
		{"1.0", "2.0", -1},
		{"2.0", "1.0", 1},
		{"v1.0", "1.0", 0},
		{"V1.0", "1.0", 0},
		{"1.0.0", "1.0", 0},
		{"1.0", "1.0.0", 0},
		{"150.0.7871.115", "149.0.0.0", 1},
		{"10.0", "9.0", 1},
		{"9", "10", -1},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestResolveBins(t *testing.T) {
	bin := bucket.BinList{{Path: "app.exe"}, {Path: "tool.exe", Name: "tool"}}
	fallback := bucket.BinList{{Path: "fallback.exe"}}
	paths := resolveBins(bin, fallback)
	if len(paths) != 2 || paths[0] != "app.exe" || paths[1] != "tool.exe" {
		t.Errorf("resolveBins with primary = %v, want [app.exe tool.exe]", paths)
	}
	paths = resolveBins(bucket.BinList{}, fallback)
	if len(paths) != 1 || paths[0] != "fallback.exe" {
		t.Errorf("resolveBins with empty = %v, want [fallback.exe]", paths)
	}
	paths = resolveBins(bucket.BinList{}, bucket.BinList{})
	if len(paths) != 0 {
		t.Errorf("resolveBins both empty = %v, want []", paths)
	}
}

func TestBuildURLs(t *testing.T) {
	urls := bucket.StringOrArray{"https://example.com/file.zip#dl.7z"}
	hashes := bucket.StringOrArray{"abc123"}
	infos := buildURLs(urls, hashes, nil, false)
	if len(infos) != 1 {
		t.Fatalf("got %d urls", len(infos))
	}
	if infos[0].URL != "https://example.com/file.zip" {
		t.Errorf("URL = %q, want stripped URL", infos[0].URL)
	}
	if infos[0].Hash != "abc123" {
		t.Errorf("Hash = %q", infos[0].Hash)
	}
	if infos[0].IsInno {
		t.Error("IsInno should be false")
	}
}

func TestBuildURLs_FallbackHash(t *testing.T) {
	urls := bucket.StringOrArray{"https://example.com/file.zip"}
	infos := buildURLs(urls, nil, bucket.StringOrArray{"fallback"}, false)
	if len(infos) != 1 || infos[0].Hash != "fallback" {
		t.Errorf("fallback hash not used: %+v", infos[0])
	}
}

func TestBuildURLs_Empty(t *testing.T) {
	infos := buildURLs(bucket.StringOrArray{}, nil, nil, false)
	if len(infos) != 1 || infos[0].URL != "" {
		t.Errorf("expected single empty urlInfo, got %+v", infos)
	}
}

func TestGetShortcuts_Arch(t *testing.T) {
	arch := map[string]bucket.ArchManifest{
		"64bit": {Shortcuts: bucket.ShortcutList{{Target: "app.exe", Name: "App64"}}},
	}
	man := &bucket.Manifest{
		Architecture: &arch,
		Shortcuts:    bucket.ShortcutList{{Target: "fallback.exe", Name: "Fallback"}},
	}
	sc := getShortcuts(man)
	if len(sc) != 1 || sc[0].Name != "App64" {
		t.Errorf("expected arch shortcut, got %+v", sc)
	}
}

func TestGetShortcuts_TopLevel(t *testing.T) {
	man := &bucket.Manifest{
		Shortcuts: bucket.ShortcutList{{Target: "app.exe", Name: "App"}},
	}
	sc := getShortcuts(man)
	if len(sc) != 1 || sc[0].Name != "App" {
		t.Errorf("expected top-level shortcut, got %+v", sc)
	}
}

func TestGetShortcuts_Empty(t *testing.T) {
	sc := getShortcuts(&bucket.Manifest{})
	if len(sc) != 0 {
		t.Errorf("expected empty, got %+v", sc)
	}
}

func TestIndexOf(t *testing.T) {
	infos := []urlInfo{
		{URL: "https://a.com/file.zip", Hash: "hash1"},
		{URL: "https://b.com/file.zip", Hash: "hash2"},
	}
	idx := indexOf(&infos, &infos[1])
	if idx != 1 {
		t.Errorf("indexOf = %d, want 1", idx)
	}
	notFound := indexOf(&infos, &urlInfo{URL: "https://x.com"})
	if notFound != -1 {
		t.Errorf("indexOf nonexistent = %d, want -1", notFound)
	}
}

func TestExtractAppName(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"googlechrome#150.0.7871.115#dl.7z_cab_b58f672a9fc9aae09c8a1b93d4b006db04e4a4d2.7z",
			"googlechrome"},
		{"firefox-1.0.zip", "firefox"},
		{"node-v18.0.0-win-x64.zip", "node-v18.0.0-win"},
		{"simple.tar.gz", "simple.tar"},
	}
	for _, tt := range tests {
		got := extractAppName(tt.filename)
		if got != tt.want {
			t.Errorf("extractAppName(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestDirSize_Empty(t *testing.T) {
	dir := t.TempDir()
	s := dirSize(dir)
	if s != 0 {
		t.Errorf("empty dir size = %d, want 0", s)
	}
}
