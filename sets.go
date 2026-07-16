// sets.go: curated glyph sets from embedded data plus user config.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const setsPathEnv = "CLYPH_SETS_PATH"

//go:embed data/sets.json
var embeddedSets []byte

type glyphSet struct {
	Description string            `json:"description,omitempty"`
	Glyphs      map[string]string `json:"glyphs"`
}

type setsFile struct {
	Sets map[string]glyphSet `json:"sets"`
}

type setListItem struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Count       int    `json:"count"`
}

type setGlyphItem struct {
	Key    string `json:"key"`
	Record Record `json:"record"`
}

type setShowResponse struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Glyphs      []setGlyphItem `json:"glyphs"`
}

type setGlyphResponse struct {
	Set    string `json:"set"`
	Key    string `json:"key"`
	Record Record `json:"record"`
}

func userConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".clyph", "config")
	}
	return filepath.Join(home, ".clyph", "config")
}

func defaultSetsPath() string {
	return filepath.Join(userConfigDir(), "sets.json")
}

func loadSets() (map[string]glyphSet, error) {
	merged, err := parseSets(embeddedSets)
	if err != nil {
		return nil, err
	}
	if envPath := os.Getenv(setsPathEnv); envPath != "" {
		return mergeSetsPath(merged, envPath, false)
	}
	return mergeSetsPath(merged, defaultSetsPath(), true)
}

func parseSets(data []byte) (map[string]glyphSet, error) {
	var file setsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	out := map[string]glyphSet{}
	for name, set := range file.Sets {
		name = normalizeSetName(name)
		glyphs := map[string]string{}
		for key, glyphName := range set.Glyphs {
			key = normalizeSetName(key)
			glyphName = strings.TrimSpace(glyphName)
			if key == "" || glyphName == "" {
				continue
			}
			glyphs[key] = glyphName
		}
		if name == "" || len(glyphs) == 0 {
			continue
		}
		out[name] = glyphSet{Description: strings.TrimSpace(set.Description), Glyphs: glyphs}
	}
	return out, nil
}

func mergeSetsPath(base map[string]glyphSet, path string, ignoreMissing bool) (map[string]glyphSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if ignoreMissing && os.IsNotExist(err) {
			return base, nil
		}
		return nil, err
	}
	overlay, err := parseSets(data)
	if err != nil {
		return nil, err
	}
	return mergeSets(base, overlay), nil
}

func mergeSets(base, overlay map[string]glyphSet) map[string]glyphSet {
	out := map[string]glyphSet{}
	for name, set := range base {
		out[name] = cloneSet(set)
	}
	for name, set := range overlay {
		merged := cloneSet(out[name])
		if set.Description != "" {
			merged.Description = set.Description
		}
		if merged.Glyphs == nil {
			merged.Glyphs = map[string]string{}
		}
		for key, glyphName := range set.Glyphs {
			merged.Glyphs[key] = glyphName
		}
		out[name] = merged
	}
	return out
}

func cloneSet(set glyphSet) glyphSet {
	out := glyphSet{Description: set.Description, Glyphs: map[string]string{}}
	for key, glyphName := range set.Glyphs {
		out.Glyphs[key] = glyphName
	}
	return out
}

func normalizeSetName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func listSets(sets map[string]glyphSet) []setListItem {
	out := make([]setListItem, 0, len(sets))
	for name, set := range sets {
		out = append(out, setListItem{Name: name, Description: set.Description, Count: len(set.Glyphs)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func resolveSetRecords(set glyphSet, records []Record) ([]setGlyphItem, error) {
	idx := buildIndex(records)
	keys := make([]string, 0, len(set.Glyphs))
	for key := range set.Glyphs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setGlyphItem, 0, len(keys))
	for _, key := range keys {
		name := set.Glyphs[key]
		rec, ok := idx[name]
		if !ok {
			return nil, cliError{"set references missing glyph: " + name}
		}
		out = append(out, setGlyphItem{Key: key, Record: rec})
	}
	return out, nil
}

func parseSetArgs(args []string) (op string, positional []string, jsonOut bool, err error) {
	if len(args) == 0 {
		return "", nil, false, cliError{"usage: clyph set <list|show|glyph> ..."}
	}
	op = args[0]
	for _, arg := range args[1:] {
		switch arg {
		case "--json":
			jsonOut = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", nil, false, cliError{"unknown flag: " + arg}
			}
			positional = append(positional, arg)
		}
	}
	return op, positional, jsonOut, nil
}

func cmdSet(args []string) int {
	op, positional, jsonOut, err := parseSetArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	switch op {
	case "list":
		if len(positional) != 0 {
			fmt.Fprintln(os.Stderr, "usage: clyph set list [--json]")
			return 2
		}
		sets, err := loadSets()
		if err != nil {
			printError(jsonOut, err.Error())
			return 1
		}
		items := listSets(sets)
		if jsonOut {
			if err := printJSON(items); err != nil {
				printError(true, err.Error())
				return 1
			}
			return 0
		}
		for _, item := range items {
			fmt.Printf("%s\t%d\t%s\n", item.Name, item.Count, item.Description)
		}
		return 0
	case "show":
		if len(positional) != 1 {
			fmt.Fprintln(os.Stderr, "usage: clyph set show <name> [--json]")
			return 2
		}
		return cmdSetShow(positional[0], jsonOut)
	case "glyph":
		if len(positional) != 2 {
			fmt.Fprintln(os.Stderr, "usage: clyph set glyph <set> <key> [--json]")
			return 2
		}
		return cmdSetGlyph(positional[0], positional[1], jsonOut)
	default:
		fmt.Fprintln(os.Stderr, "unknown set command: "+op)
		return 2
	}
}

func cmdSetShow(name string, jsonOut bool) int {
	sets, err := loadSets()
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	set, ok := sets[normalizeSetName(name)]
	if !ok {
		printError(jsonOut, "set not found: "+name)
		return 1
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	items, err := resolveSetRecords(set, records)
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	if jsonOut {
		if err := printJSON(setShowResponse{Name: normalizeSetName(name), Description: set.Description, Glyphs: items}); err != nil {
			printError(true, err.Error())
			return 1
		}
		return 0
	}
	for _, item := range items {
		fmt.Printf("%s\t%s\t%s\t%s\n", item.Key, item.Record.Name, item.Record.Codepoint, item.Record.Glyph)
	}
	return 0
}

func cmdSetGlyph(setName, key string, jsonOut bool) int {
	sets, err := loadSets()
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	set, ok := sets[normalizeSetName(setName)]
	if !ok {
		printError(jsonOut, "set not found: "+setName)
		return 1
	}
	glyphName, ok := set.Glyphs[normalizeSetName(key)]
	if !ok {
		printError(jsonOut, fmt.Sprintf("set key not found: %s/%s", setName, key))
		return 1
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	rec, ok := buildIndex(records)[glyphName]
	if !ok {
		printError(jsonOut, "set references missing glyph: "+glyphName)
		return 1
	}
	if jsonOut {
		if err := printJSON(setGlyphResponse{Set: normalizeSetName(setName), Key: normalizeSetName(key), Record: rec}); err != nil {
			printError(true, err.Error())
			return 1
		}
		return 0
	}
	fmt.Println(rec.Glyph)
	return 0
}
