// semantic.go: concept -> glyph resolution from a curated seed, with label/alias fallback.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

//go:embed data/semantic.json
var embeddedSemanticSeed []byte

type semanticSeed struct {
	Concepts map[string]string `json:"concepts"`
}

// loadSemanticSeed returns the concept->canonical-name map. A custom seed can
// be supplied via CLYPH_SEMANTIC_PATH; otherwise the embedded shipped seed is used.
func loadSemanticSeed() (map[string]string, error) {
	raw := embeddedSemanticSeed
	if p := os.Getenv("CLYPH_SEMANTIC_PATH"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	var s semanticSeed
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return s.Concepts, nil
}

// semanticMatch is one ranked resolution result.
type semanticMatch struct {
	Record    Record `json:"record"`
	Canonical bool   `json:"canonical"`
	Source    string `json:"source"` // "seed" | "label" | "alias" | "search"
}

// resolveSemantic resolves a concept into a ranked list of matches.
// Order: seed canonical (if present), then exact label/alias matches, then (only
// when no seed+exact hit) a substring search fallback.
func resolveSemantic(seed map[string]string, records []Record, concept string) []semanticMatch {
	idx := buildIndex(records)
	var out []semanticMatch
	seen := map[string]bool{}

	if name, ok := seed[concept]; ok {
		if r, ok := idx[name]; ok {
			out = append(out, semanticMatch{Record: r, Canonical: true, Source: "seed"})
			seen[r.Name] = true
		}
	}
	for _, r := range records {
		if seen[r.Name] {
			continue
		}
		if r.Label == concept {
			out = append(out, semanticMatch{Record: r, Source: "label"})
			seen[r.Name] = true
			continue
		}
		for _, a := range r.Aliases {
			if a == concept {
				out = append(out, semanticMatch{Record: r, Source: "alias"})
				seen[r.Name] = true
				break
			}
		}
	}
	if len(out) == 0 {
		// Degrade to a substring search so unknown concepts are still useful.
		fallback, _ := searchRecords(records, concept, -1, 0)
		for _, r := range fallback {
			out = append(out, semanticMatch{Record: r, Source: "search"})
		}
	}
	return out
}

type semanticResponse struct {
	Concept   string          `json:"concept"`
	Canonical *Record         `json:"canonical,omitempty"`
	Matches   []semanticMatch `json:"matches"`
}

func parseSemanticArgs(args []string) (concept string, all, jsonOut bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, _, _ := strings.Cut(arg, "=")
		switch name {
		case "--json":
			jsonOut = true
		case "--all":
			all = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, false, cliError{"unknown flag: " + arg}
			}
			if concept != "" {
				return "", false, false, cliError{"too many arguments"}
			}
			concept = arg
		}
	}
	if concept == "" {
		return "", false, false, cliError{"missing concept"}
	}
	return strings.ToLower(strings.TrimSpace(concept)), all, jsonOut, nil
}

func cmdSemantic(args []string) int {
	concept, all, jsonOut, err := parseSemanticArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	seed, serr := loadSemanticSeed()
	if serr != nil {
		printError(jsonOut, serr.Error())
		return 1
	}
	records, lerr := loadRecords(catalogPath())
	if lerr != nil {
		printError(jsonOut, lerr.Error())
		return 1
	}
	matches := resolveSemantic(seed, records, concept)

	if jsonOut {
		resp := semanticResponse{Concept: concept, Matches: matches}
		for i := range matches {
			if matches[i].Canonical {
				c := matches[i].Record
				resp.Canonical = &c
				break
			}
		}
		if perr := printJSON(resp); perr != nil {
			printError(true, perr.Error())
			return 1
		}
		return 0
	}

	if len(matches) == 0 {
		printError(false, fmt.Sprintf("no semantic match for: %s", concept))
		return 1
	}
	if !all {
		// Default: crisp single canonical row.
		matches = matches[:1]
	}
	for _, m := range matches {
		fmt.Printf("%s\t%s\t%s\t%s\n", m.Record.Name, m.Record.Codepoint, m.Record.Glyph, m.Source)
	}
	return 0
}
