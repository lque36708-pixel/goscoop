package download

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"ss/internal/progress"
)

func TestCachePath(t *testing.T) {
	cacheDir := filepath.Join("cache")
	url := "https://example.com/path/to/file.zip"
	got := CachePath(cacheDir, url)
	want := filepath.Join(cacheDir, "file.zip")
	if got != want {
		t.Errorf("CachePath(%q, %q) = %q, want %q", cacheDir, url, got, want)
	}
}

func TestProbeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", "1000")
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	total, supportsRange, err := probeURL(server.URL)
	if err != nil {
		t.Fatalf("probeURL failed: %v", err)
	}
	if total != 1000 {
		t.Errorf("total = %d, want 1000", total)
	}
	if !supportsRange {
		t.Error("supportsRange should be true")
	}
}

func TestProbeURL_NoRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "500")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	total, supportsRange, err := probeURL(server.URL)
	if err != nil {
		t.Fatalf("probeURL failed: %v", err)
	}
	if total != 500 {
		t.Errorf("total = %d, want 500", total)
	}
	if supportsRange {
		t.Error("supportsRange should be false for server without Accept-Ranges")
	}
}

func TestProbeURL_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, _, err := probeURL(server.URL)
	if err == nil {
		t.Error("expected error for 500")
	}
}

func TestProbeURL_NetworkError(t *testing.T) {
	_, _, err := probeURL("http://127.0.0.1:1")
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestDownloadSingle(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.Header().Set("Content-Length", "13")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "output.bin")
	if err := downloadSingle(server.URL, dest, "output.bin", 13); err != nil {
		t.Fatalf("downloadSingle failed: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Hello, World!" {
		t.Errorf("content = %q, want %q", data, "Hello, World!")
	}
}

func TestDownloadSingle_WithProgress(t *testing.T) {
	content := make([]byte, 1024*100) // 100 KB
	for i := range content {
		content[i] = byte(i % 256)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "large.bin")
	if err := downloadSingle(server.URL, dest, "large.bin", int64(len(content))); err != nil {
		t.Fatalf("downloadSingle large failed: %v", err)
	}
	data, _ := os.ReadFile(dest)
	if len(data) != len(content) {
		t.Errorf("downloaded %d bytes, want %d", len(data), len(content))
	}
}

func TestDownloadSingle_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := downloadSingle(server.URL, filepath.Join(t.TempDir(), "out"), "out", 0)
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestDownloadRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("raw data"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "raw.bin")
	if err := downloadRaw(server.URL, dest); err != nil {
		t.Fatalf("downloadRaw failed: %v", err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "raw data" {
		t.Errorf("content = %q", data)
	}
}

func TestDownloadRaw_Error(t *testing.T) {
	err := downloadRaw("http://127.0.0.1:1/", filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestFile_RawGithub(t *testing.T) {
	url := "https://raw.githubusercontent.com/test/repo/main/file.txt"
	dest := filepath.Join(t.TempDir(), "file.txt")
	// Can't actually reach github in test — expect network error, not routing error
	err := File(url, dest)
	if err != nil {
		t.Logf("File(raw github) returned: %v (expected if offline)", err)
	}
}

func TestFile_UnknownHost(t *testing.T) {
	err := File("http://127.0.0.1:1/nonexistent", filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestDownloadPart(t *testing.T) {
	fullContent := []byte("Hello, World! This is a test.")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.WriteHeader(http.StatusOK)
			return
		}
		var start, end int
		fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(fullContent)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(fullContent[start : end+1])
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "part.tmp")
	os.WriteFile(dest, make([]byte, len(fullContent)), 0644)
	bar := progress.New("test", int64(len(fullContent)))

	var mu sync.Mutex
	err := downloadPart(server.URL, dest, "test", 0, int64(len(fullContent)-1), 0, &mu, bar)
	if err != nil {
		t.Fatalf("downloadPart failed: %v", err)
	}
	bar.Done()

	data, _ := os.ReadFile(dest)
	if string(data) != string(fullContent) {
		t.Errorf("part content mismatch: got %q", data)
	}
}

func TestDownloadPart_NoRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("full content"))
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "part.tmp")
	os.WriteFile(dest, []byte(""), 0644)
	bar := progress.New("test", 100)
	var mu sync.Mutex

	err := downloadPart(server.URL, dest, "test", 0, 10, 0, &mu, bar)
	if err == nil {
		t.Error("expected error when server returns 200 instead of 206")
	}
}

func TestDownloadMulti(t *testing.T) {
	// Create content >= 10MB to trigger multi-threaded
	content := make([]byte, minMultiSize+1)
	for i := range content {
		content[i] = byte(i % 256)
	}
	callCount := 0
	var muCall sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		muCall.Lock()
		callCount++
		muCall.Unlock()
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			w.WriteHeader(http.StatusOK)
			w.Write(content)
			return
		}
		var start, end int
		fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(content[start : end+1])
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "multi.bin")
	if err := downloadMulti(server.URL, dest, "multi.bin", int64(len(content))); err != nil {
		t.Fatalf("downloadMulti failed: %v", err)
	}
	data, _ := os.ReadFile(dest)
	if len(data) != len(content) {
		t.Errorf("downloaded %d bytes, want %d", len(data), len(content))
	}
}

func TestDownloadSmart_SmallFile(t *testing.T) {
	content := []byte("small file")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "small.bin")
	if err := downloadSmart(server.URL, dest, "small.bin"); err != nil {
		t.Fatalf("downloadSmart failed: %v", err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "small file" {
		t.Errorf("content = %q", data)
	}
}
