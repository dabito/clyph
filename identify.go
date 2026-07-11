// identify.go: reverse glyph lookup (glyph char -> record).
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type identifyResponse struct {
	Query   string   `json:"query"`
	Matches []Record `json:"matches"`
}

func glyphFamily(name string) string {
	s := strings.TrimPrefix(name, "nf-")
	if s == name {
		return ""
	}
	idx := strings.Index(s, "-")
	if idx < 0 {
		return ""
	}
	return s[:idx]
}

func findRecordsByGlyph(records []Record, glyph string) []Record {
	var out []Record
	for _, rec := range records {
		if rec.Glyph == glyph {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func parseIdentifyArgs(args []string) (glyphs []string, jsonOut bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOut = true
		default:
			if strings.HasPrefix(arg, "--") {
				return nil, false, cliError{"unknown flag: " + arg}
			}
			glyphs = append(glyphs, arg)
		}
	}
	return glyphs, jsonOut, nil
}

func cmdIdentify(args []string) int {
	glyphs, jsonOut, err := parseIdentifyArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	var rawInput string
	if len(glyphs) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		rawInput = string(data)
		for _, r := range rawInput {
			if r != '\n' && r != '\r' && r != ' ' && r != '\t' {
				glyphs = append(glyphs, string(r))
			}
		}
		if len(glyphs) == 0 {
			fmt.Fprintln(os.Stderr, cliError{"missing glyph input"})
			return 2
		}
	} else {
		rawInput = strings.Join(glyphs, "")
	}

	records, err := loadRecords(catalogPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if jsonOut {
		matches := []Record{}
		seen := map[string]bool{}
		for _, g := range glyphs {
			for _, rec := range findRecordsByGlyph(records, g) {
				if !seen[rec.Name] {
					seen[rec.Name] = true
					matches = append(matches, rec)
				}
			}
		}
		if err := printJSON(identifyResponse{Query: rawInput, Matches: matches}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	for _, g := range glyphs {
		for _, rec := range findRecordsByGlyph(records, g) {
			fmt.Printf("%s\t%s\t%s\t%s\n", rec.Name, rec.Codepoint, glyphFamily(rec.Name), rec.Glyph)
		}
	}
	return 0
}
