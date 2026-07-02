package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Record is a single glyph catalog entry.
type Record struct {
	Name      string   `json:"name"`
	Codepoint string   `json:"codepoint"`
	Unicode   string   `json:"unicode"`
	Glyph     string   `json:"glyph"`
	Label     string   `json:"label"`
	Aliases   []string `json:"aliases"`
}

type catalogFile struct {
	Records []Record `json:"records"`
}

func catalogPath() string {
	if v := os.Getenv(catalogPathEnv); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".clyph", "data", "catalog.json")
	}
	return filepath.Join(home, ".clyph", "data", "catalog.json")
}

func unicodeEscape(r rune) string {
	if r <= 0xFFFF {
		return fmt.Sprintf("\\u%04x", r)
	}
	return fmt.Sprintf("\\U%08x", r)
}

func codepointToGlyph(codepointHex string) (string, error) {
	v, err := strconv.ParseInt(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(codepointHex)), "0x"), 16, 32)
	if err != nil {
		return "", err
	}
	return string(rune(v)), nil
}

func normalizeAliases(aliases []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

func recordFromParts(name, codepointHex string, aliases []string) (Record, error) {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(codepointHex)), "0x")
	g, err := codepointToGlyph(normalized)
	if err != nil {
		return Record{}, err
	}
	v, _ := strconv.ParseInt(normalized, 16, 32)
	return Record{
		Name:      name,
		Codepoint: normalized,
		Unicode:   unicodeEscape(rune(v)),
		Glyph:     g,
		Label:     "",
		Aliases:   normalizeAliases(aliases),
	}, nil
}

func loadRecords(path string) ([]Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload catalogFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	recs := make([]Record, 0, len(payload.Records))
	for _, rec := range payload.Records {
		recs = append(recs, Record{
			Name:      rec.Name,
			Codepoint: strings.ToLower(rec.Codepoint),
			Unicode:   rec.Unicode,
			Glyph:     rec.Glyph,
			Label:     rec.Label,
			Aliases:   normalizeAliases(rec.Aliases),
		})
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Name < recs[j].Name })
	return recs, nil
}

func buildIndex(records []Record) map[string]Record {
	idx := make(map[string]Record, len(records))
	for _, rec := range records {
		idx[rec.Name] = rec
	}
	return idx
}

func saveRecords(records []Record, path string) error {
	sort.Slice(records, func(i, j int) bool { return records[i].Name < records[j].Name })
	payload := catalogFile{Records: records}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func mergeCatalog(existing, fresh []Record) []Record {
	meta := buildIndex(existing)
	merged := make([]Record, 0, len(fresh))
	for _, rec := range fresh {
		if old, ok := meta[rec.Name]; ok {
			rec.Label = old.Label
			rec.Aliases = normalizeAliases(old.Aliases)
		}
		merged = append(merged, rec)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Name < merged[j].Name })
	return merged
}

// searchRecords returns records matching needle in name, label, or aliases.
// limit >= 1 caps results; limit < 0 means unlimited; limit == 0 returns at most 1
// (the >= comparison fires immediately after the first append).
func searchRecords(records []Record, query string, limit int) []Record {
	needle := strings.ToLower(strings.TrimSpace(query))
	matches := make([]Record, 0)
	for _, rec := range records {
		haystacks := []string{strings.ToLower(rec.Name), strings.ToLower(rec.Label)}
		for _, alias := range rec.Aliases {
			haystacks = append(haystacks, strings.ToLower(alias))
		}
		if needle == "" || containsAny(haystacks, needle) {
			matches = append(matches, rec)
			if limit >= 0 && len(matches) >= limit {
				break
			}
		}
	}
	return matches
}

func containsAny(haystacks []string, needle string) bool {
	for _, hay := range haystacks {
		if strings.Contains(hay, needle) {
			return true
		}
	}
	return false
}

func formatRow(rec Record) string {
	label := rec.Label
	if label == "" {
		if len(rec.Aliases) > 0 {
			label = strings.Join(rec.Aliases, "/")
		} else {
			label = "-"
		}
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s", rec.Name, rec.Codepoint, rec.Glyph, label)
}
