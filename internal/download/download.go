package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ss/internal/progress"
)

const (
	minMultiSize = 10 * 1024 * 1024 // 10 MB threshold for multi-threaded
	numParts     = 4
)

func File(url, dest string) error {
	if strings.HasPrefix(url, "https://raw.githubusercontent.com") ||
		strings.HasPrefix(url, "http://raw.githubusercontent.com") {
		return downloadRaw(url, dest)
	}

	cacheDir := filepath.Dir(dest)
	_ = os.MkdirAll(cacheDir, 0755)

	name := filepath.Base(dest)
	return downloadSmart(url, dest, name)
}

func downloadSmart(url, dest, name string) error {
	_ = os.MkdirAll(filepath.Dir(dest), 0755)

	// Check server capabilities and file size
	total, supportsRange, err := probeURL(url)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}

	// Single-threaded for small files or no Range support
	if total < minMultiSize || !supportsRange {
		return downloadSingle(url, dest, name, total)
	}

	// Multi-threaded download
	return downloadMulti(url, dest, name, total)
}

func probeURL(url string) (total int64, supportsRange bool, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return 0, false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("HEAD %s returned %d", url, resp.StatusCode)
	}

	total = resp.ContentLength
	supportsRange = resp.Header.Get("Accept-Ranges") == "bytes"

	// Some servers don't advertise but still support Range
	// Try with a small range to detect
	if !supportsRange && total > 0 {
		testReq, _ := http.NewRequest("GET", url, nil)
		testReq.Header.Set("Range", "bytes=0-0")
		testResp, testErr := http.DefaultClient.Do(testReq)
		if testErr == nil {
			defer testResp.Body.Close()
			if testResp.StatusCode == 206 {
				supportsRange = true
			}
		}
	}

	return total, supportsRange, nil
}

func downloadSingle(url, dest, name string, total int64) error {
	out, err := os.Create(dest + ".tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		os.Remove(dest + ".tmp")
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(dest + ".tmp")
		return fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	if total <= 0 {
		total = resp.ContentLength
	}

	bar := progress.New(fmt.Sprintf("Downloading %s", name), total)
	if total <= 0 {
		bar = progress.New(fmt.Sprintf("Downloading %s", name), 100)
	}

	var written int64
	buf := make([]byte, 64*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				os.Remove(dest + ".tmp")
				return werr
			}
			written += int64(n)
			if total > 0 {
				bar.SetCurrent(written)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(dest + ".tmp")
			return fmt.Errorf("read body: %w", err)
		}
	}

	bar.Done()
	out.Close()
	return os.Rename(dest+".tmp", dest)
}

func downloadMulti(url, dest, name string, total int64) error {
	partSize := total / numParts
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, numParts)
	bar := progress.New(fmt.Sprintf("Downloading %s (x%d)", name, numParts), total)

	// Create temp file with correct size
	tmpFile := dest + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	f.Truncate(total)
	f.Close()

	for i := range numParts {
		wg.Add(1)
		go func(part int) {
			defer wg.Done()

			start := int64(part) * partSize
			end := start + partSize - 1
			if part == numParts-1 {
				end = total - 1
			}

			err := downloadPart(url, tmpFile, name, start, end, part, &mu, bar)
			mu.Lock()
			errs[part] = err
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("multi-download: %w", err)
		}
	}

	bar.Done()
	// Remove part tracking files
	return os.Rename(tmpFile, dest)
}

func downloadPart(url, tmpFile, name string, start, end int64, part int, mu *sync.Mutex, bar *progress.Bar) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 206 {
		return fmt.Errorf("part %d: server returned %d (expected 206)", part, resp.StatusCode)
	}

	// Open file for writing at the correct offset
	f, err := os.OpenFile(tmpFile, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return err
	}

	buf := make([]byte, 256*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			mu.Lock()
			bar.Add(int64(n))
			mu.Unlock()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadRaw(url, dest string) error {
	name := filepath.Base(dest)
	spinner := progress.NewSpinner(fmt.Sprintf("Downloading %s", name))
	spinner.Start()

	out, err := os.Create(dest)
	if err != nil {
		spinner.Fail(err.Error())
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		spinner.Fail(err.Error())
		os.Remove(dest)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		spinner.Fail(fmt.Sprintf("GET returned %d", resp.StatusCode))
		os.Remove(dest)
		return fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		spinner.Fail(err.Error())
		os.Remove(dest)
		return err
	}

	spinner.Done("")
	return nil
}

func CachePath(cacheDir, url string) string {
	name := filepath.Base(url)
	return filepath.Join(cacheDir, name)
}
