package bucket

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	Version       string                  `json:"version"`
	Description   string                  `json:"description"`
	Homepage      string                  `json:"homepage"`
	License       json.RawMessage         `json:"license"`
	URL           StringOrArray           `json:"url"`
	Hash          StringOrArray           `json:"hash"`
	Bin           BinList                 `json:"bin"`
	Depends       StringOrArray           `json:"depends"`
	ExtractDir    string                  `json:"extract_dir"`
	InnoSetup     bool                    `json:"innosetup"`
	PreInstall    DynamicStringArray      `json:"pre_install"`
	PostInstall   DynamicStringArray      `json:"post_install"`
	Notes         StringOrArray           `json:"notes"`
	Suggestions   StringOrArray           `json:"suggestions"`
	Persist       PersistList             `json:"persist"`
	Shortcuts     ShortcutList            `json:"shortcuts"`
	Architecture  *map[string]ArchManifest `json:"architecture,omitempty"`
	Checkver      json.RawMessage         `json:"checkver"`
	Autoupdate    json.RawMessage         `json:"autoupdate"`
}

type ArchManifest struct {
	URL        StringOrArray `json:"url"`
	Hash       StringOrArray `json:"hash"`
	Bin        BinList       `json:"bin"`
	ExtractDir string        `json:"extract_dir"`
	InnoSetup  *bool         `json:"innosetup,omitempty"`
	Shortcuts  ShortcutList  `json:"shortcuts"`
}

type StringOrArray []string

func (sa *StringOrArray) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*sa = []string{}
		return nil
	}
	if data[0] == '[' {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*sa = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*sa = []string{s}
	return nil
}

type DynamicStringArray []string

func (dsa *DynamicStringArray) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*dsa = []string{}
		return nil
	}
	if data[0] == '[' {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*dsa = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*dsa = []string{s}
	return nil
}

type PersistEntry struct {
	Source string
	Target string
}

type PersistList []PersistEntry

func (pl *PersistList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*pl = []PersistEntry{}
		return nil
	}
	if data[0] == '[' {
		// Could be ["file"] or [["source", "target"]] or ["file1", ["src1", "tgt1"]]
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		for _, item := range raw {
			itemStr := string(item)
			if itemStr[0] == '[' {
				var pair []string
				if err := json.Unmarshal(item, &pair); err != nil {
					continue
				}
				if len(pair) >= 2 {
					*pl = append(*pl, PersistEntry{Source: pair[0], Target: pair[1]})
				}
			} else {
				var s string
				if err := json.Unmarshal(item, &s); err != nil {
					continue
				}
				*pl = append(*pl, PersistEntry{Source: s, Target: s})
			}
		}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*pl = append(*pl, PersistEntry{Source: s, Target: s})
	return nil
}

type BinEntry struct {
	Path string
	Name string // alias (optional)
}

type BinList []BinEntry

func (bl *BinList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*bl = []BinEntry{}
		return nil
	}
	if data[0] == '[' {
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		if len(raw) == 0 {
			return nil
		}
		// Check if first element is a nested array or a string
		first := string(raw[0])
		if first[0] == '[' {
			// Nested entries: [["chrome.exe", "alias"], ["tool2.exe"]]
			for _, item := range raw {
				var parts []string
				if err := json.Unmarshal(item, &parts); err != nil {
					continue
				}
				entry := BinEntry{Path: parts[0]}
				if len(parts) >= 2 {
					entry.Name = parts[1]
				}
				*bl = append(*bl, entry)
			}
		} else {
			// Flat array: each string is a separate binary
			for _, item := range raw {
				var s string
				if err := json.Unmarshal(item, &s); err != nil {
					continue
				}
				*bl = append(*bl, BinEntry{Path: s})
			}
		}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*bl = append(*bl, BinEntry{Path: s})
	return nil
}

func (bl BinList) Paths() []string {
	var paths []string
	for _, e := range bl {
		paths = append(paths, e.Path)
	}
	return paths
}

type ShortcutEntry struct {
	Target    string
	Name      string
	Arguments string
	Icon      string
}

type ShortcutList []ShortcutEntry

func (sl *ShortcutList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, item := range raw {
		var parts []string
		if err := json.Unmarshal(item, &parts); err != nil {
			continue
		}
		var entry ShortcutEntry
		if len(parts) >= 1 {
			entry.Target = parts[0]
		}
		if len(parts) >= 2 {
			entry.Name = parts[1]
		}
		if len(parts) >= 3 {
			entry.Arguments = parts[2]
		}
		if len(parts) >= 4 {
			entry.Icon = parts[3]
		}
		*sl = append(*sl, entry)
	}
	return nil
}

type SearchIndexEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Bucket  string `json:"bucket"`
	Desc    string `json:"desc"`
	NameLow string `json:"nl,omitempty"`
	DescLow string `json:"dl,omitempty"`
}

func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func collectAllManifests(bucketsDir string) []SearchIndexEntry {
	var entries []SearchIndexEntry
	dirEntries, _ := os.ReadDir(bucketsDir)
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}
		bucketName := entry.Name()
		manifestDir := filepath.Join(bucketsDir, bucketName, "bucket")
		if _, err := os.Stat(manifestDir); os.IsNotExist(err) {
			manifestDir = filepath.Join(bucketsDir, bucketName)
		}
		files, _ := filepath.Glob(filepath.Join(manifestDir, "*.json"))
		for _, f := range files {
			name := strings.TrimSuffix(filepath.Base(f), ".json")
			man, err := ReadManifest(f)
			if err != nil {
				continue
			}
			entries = append(entries, SearchIndexEntry{
				Name:    name,
				Version: man.Version,
				Bucket:  bucketName,
				Desc:    man.Description,
				NameLow: strings.ToLower(name),
				DescLow: strings.ToLower(man.Description),
			})
		}
	}
	return entries
}

func BuildSearchIndex(bucketsDir, cachePath string) error {
	entries := collectAllManifests(bucketsDir)
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0644)
}

func LoadSearchIndex(cachePath string) ([]SearchIndexEntry, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	var entries []SearchIndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func SearchIndex(entries []SearchIndexEntry, query string) []SearchIndexEntry {
	if query == "" {
		return entries
	}
	query = strings.ToLower(query)
	results := make([]SearchIndexEntry, 0)
	for _, e := range entries {
		if strings.Contains(e.NameLow, query) ||
			strings.Contains(e.DescLow, query) {
			results = append(results, e)
		}
	}
	return results
}

func SearchBuckets(bucketsDir, query string) []SearchIndexEntry {
	return SearchIndex(collectAllManifests(bucketsDir), query)
}
