package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	version        = "0.1.0-beta.4"
	defaultLimit   = 10
	defaultSource  = "https://www.nerdfonts.com/assets/css/webfont.css"
	catalogPathEnv = "CLYPH_CATALOG_PATH"
)

type searchResponse struct {
	Query   string   `json:"query"`
	Matches []Record `json:"matches"`
}

type glyphResponse struct {
	Name  string `json:"name"`
	Glyph string `json:"glyph"`
}

type codepointResponse struct {
	Name      string `json:"name"`
	Codepoint string `json:"codepoint"`
}

type updateResponse struct {
	Status  string `json:"status"`
	Records int    `json:"records"`
	Catalog string `json:"catalog"`
}

type cliError struct{ msg string }

func (e cliError) Error() string { return e.msg }

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(data))
	return err
}

func cmdSearch(args []string) int {
	query, limit, jsonOut, err := parseSearchArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	matches := searchRecords(records, query, limit)
	if jsonOut {
		resp := searchResponse{Query: query, Matches: matches}
		if err := printJSON(resp); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	for _, rec := range matches {
		fmt.Println(formatRow(rec))
	}
	return 0
}

func cmdGet(args []string) int {
	name, jsonOut, err := parseSingleNameArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "not found: %s\n", name)
		return 1
	}
	if jsonOut {
		if err := printJSON(rec); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Println(formatRow(rec))
	return 0
}

func cmdGlyph(args []string) int {
	name, jsonOut, err := parseSingleNameArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "not found: %s\n", name)
		return 1
	}
	if jsonOut {
		if err := printJSON(glyphResponse{Name: rec.Name, Glyph: rec.Glyph}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Println(rec.Glyph)
	return 0
}

func cmdCodepoint(args []string) int {
	name, jsonOut, err := parseSingleNameArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "not found: %s\n", name)
		return 1
	}
	if jsonOut {
		if err := printJSON(codepointResponse{Name: rec.Name, Codepoint: rec.Codepoint}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Println(rec.Codepoint)
	return 0
}

func cmdUpdate(args []string) int {
	source, jsonOut, err := parseUpdateArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	text, err := loadSource(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
		return 1
	}
	fresh, err := parseCSSCatalog(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
		return 1
	}
	if len(fresh) == 0 {
		fmt.Fprintln(os.Stderr, "update failed: no glyph records parsed from source")
		return 1
	}
	existing, err := loadRecords(catalogPath())
	if err != nil {
		existing = nil
	}
	merged := mergeCatalog(existing, fresh)
	if err := saveRecords(merged, catalogPath()); err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
		return 1
	}
	if jsonOut {
		if err := printJSON(updateResponse{Status: "updated", Records: len(merged), Catalog: catalogPath()}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Printf("updated %d records\n", len(merged))
	return 0
}

func parseSearchArgs(args []string) (query string, limit int, jsonOut bool, err error) {
	limit = defaultLimit
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOut = true
		case "--limit":
			i++
			if i >= len(args) {
				return "", 0, false, cliError{"missing value for --limit"}
			}
			limit, err = strconv.Atoi(args[i])
			if err != nil {
				return "", 0, false, cliError{"invalid --limit"}
			}
			if limit < 0 {
				return "", 0, false, cliError{"--limit must be non-negative"}
			}
		default:
			if strings.HasPrefix(arg, "--") {
				return "", 0, false, cliError{"unknown flag: " + arg}
			}
			if query != "" {
				return "", 0, false, cliError{"too many arguments"}
			}
			query = arg
		}
	}
	if query == "" {
		return "", 0, false, cliError{"missing query"}
	}
	return query, limit, jsonOut, nil
}

func parseSingleNameArgs(args []string) (name string, jsonOut bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOut = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, cliError{"unknown flag: " + arg}
			}
			if name != "" {
				return "", false, cliError{"too many arguments"}
			}
			name = arg
		}
	}
	if name == "" {
		return "", false, cliError{"missing name"}
	}
	return name, jsonOut, nil
}

func parseUpdateArgs(args []string) (source string, jsonOut bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOut = true
		case "--source":
			i++
			if i >= len(args) {
				return "", false, cliError{"missing value for --source"}
			}
			source = args[i]
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, cliError{"unknown flag: " + arg}
			}
			return "", false, cliError{"too many arguments"}
		}
	}
	if source == "" {
		source = defaultSource
	}
	return source, jsonOut, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: clyph <search|get|glyph|codepoint|update|version> ...")
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "search":
		return cmdSearch(rest)
	case "get":
		return cmdGet(rest)
	case "glyph":
		return cmdGlyph(rest)
	case "codepoint":
		return cmdCodepoint(rest)
	case "update":
		return cmdUpdate(rest)
	case "version", "--version", "-v":
		fmt.Printf("clyph %s\n", version)
		return 0
	case "-h", "--help", "help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		return 2
	}
}
