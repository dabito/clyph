package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	version        = "0.4.0"
	defaultLimit   = 100
	defaultSource  = "https://www.nerdfonts.com/assets/css/webfont.css"
	catalogPathEnv = "CLYPH_CATALOG_PATH"
)

type searchResponse struct {
	Query   string   `json:"query"`
	Total   int      `json:"total"`
	Offset  int      `json:"offset"`
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

type errorResponse struct {
	Error string `json:"error"`
}

type cliError struct{ msg string }

func (e cliError) Error() string { return e.msg }

// flagValue resolves a flag's value, supporting both "--flag value" and
// "--flag=value" forms. i is advanced past the value when the space form is
// used. inline/hasInline come from splitting the raw arg on "=" up front.
func flagValue(args []string, i *int, flagName, inline string, hasInline bool) (string, error) {
	if hasInline {
		return inline, nil
	}
	*i++
	if *i >= len(args) {
		return "", cliError{"missing value for " + flagName}
	}
	return args[*i], nil
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(data))
	return err
}

// printError reports a runtime failure on stderr. When jsonOut is set, the
// message is wrapped as JSON so scripts parsing --json output don't have to
// fall back to plain text on the failure path.
func printError(jsonOut bool, msg string) {
	if jsonOut {
		if data, err := json.MarshalIndent(errorResponse{Error: msg}, "", "  "); err == nil {
			fmt.Fprintln(os.Stderr, string(data))
			return
		}
	}
	fmt.Fprintln(os.Stderr, msg)
}

func cmdSearch(args []string) int {
	query, limit, offset, jsonOut, pretty, err := parseSearchArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	matches, total := searchRecords(records, query, limit, offset)
	if jsonOut {
		resp := searchResponse{Query: query, Total: total, Offset: offset, Matches: matches}
		if err := printJSON(resp); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if pretty {
		for _, row := range formatRowsPretty(matches) {
			fmt.Println(row)
		}
	} else {
		for _, rec := range matches {
			fmt.Println(formatRow(rec))
		}
	}
	if offset > 0 || offset+len(matches) < total {
		start, end := offset, offset
		if len(matches) > 0 {
			start, end = offset+1, offset+len(matches)
		}
		fmt.Fprintf(os.Stderr, "showing %d-%d of %d matches; use --offset/--limit to see more\n", start, end, total)
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
		printError(jsonOut, err.Error())
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		printError(jsonOut, fmt.Sprintf("not found: %s", name))
		return 1
	}
	if jsonOut {
		if err := printJSON(rec); err != nil {
			printError(jsonOut, err.Error())
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
		printError(jsonOut, err.Error())
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		printError(jsonOut, fmt.Sprintf("not found: %s", name))
		return 1
	}
	if jsonOut {
		if err := printJSON(glyphResponse{Name: rec.Name, Glyph: rec.Glyph}); err != nil {
			printError(jsonOut, err.Error())
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
		printError(jsonOut, err.Error())
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		printError(jsonOut, fmt.Sprintf("not found: %s", name))
		return 1
	}
	if jsonOut {
		if err := printJSON(codepointResponse{Name: rec.Name, Codepoint: rec.Codepoint}); err != nil {
			printError(jsonOut, err.Error())
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
		printError(jsonOut, fmt.Sprintf("update failed: %v", err))
		return 1
	}
	fresh, err := parseCSSCatalog(text)
	if err != nil {
		printError(jsonOut, fmt.Sprintf("update failed: %v", err))
		return 1
	}
	if len(fresh) == 0 {
		printError(jsonOut, "update failed: no glyph records parsed from source")
		return 1
	}
	existing, err := loadRecords(catalogPath())
	if err != nil {
		existing = nil
	}
	merged := mergeCatalog(existing, fresh)
	if err := saveRecords(merged, catalogPath()); err != nil {
		printError(jsonOut, fmt.Sprintf("update failed: %v", err))
		return 1
	}
	if jsonOut {
		if err := printJSON(updateResponse{Status: "updated", Records: len(merged), Catalog: catalogPath()}); err != nil {
			printError(jsonOut, err.Error())
			return 1
		}
		return 0
	}
	fmt.Printf("updated %d records\n", len(merged))
	return 0
}

func cmdLabel(args []string) int {
	name, text, clear, jsonOut, err := parseLabelArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		printError(jsonOut, fmt.Sprintf("not found: %s", name))
		return 1
	}
	if clear {
		rec.Label = ""
	} else {
		rec.Label = text
	}
	if err := saveRecords(replaceRecord(records, rec), catalogPath()); err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	if jsonOut {
		if err := printJSON(rec); err != nil {
			printError(jsonOut, err.Error())
			return 1
		}
		return 0
	}
	fmt.Println(formatRow(rec))
	return 0
}

func cmdAlias(args []string) int {
	name, op, value, jsonOut, err := parseAliasArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		printError(jsonOut, fmt.Sprintf("not found: %s", name))
		return 1
	}
	switch op {
	case "add":
		rec.Aliases = normalizeAliases(append(rec.Aliases, value))
	case "rm":
		rec.Aliases = removeAlias(rec.Aliases, value)
	}
	if err := saveRecords(replaceRecord(records, rec), catalogPath()); err != nil {
		printError(jsonOut, err.Error())
		return 1
	}
	if jsonOut {
		if err := printJSON(rec); err != nil {
			printError(jsonOut, err.Error())
			return 1
		}
		return 0
	}
	fmt.Println(formatRow(rec))
	return 0
}

func parseSearchArgs(args []string) (query string, limit, offset int, jsonOut, pretty bool, err error) {
	limit = defaultLimit
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, inline, hasInline := strings.Cut(arg, "=")
		switch name {
		case "--json":
			jsonOut = true
		case "--pretty":
			pretty = true
		case "--limit":
			val, verr := flagValue(args, &i, "--limit", inline, hasInline)
			if verr != nil {
				return "", 0, 0, false, false, verr
			}
			limit, err = strconv.Atoi(val)
			if err != nil {
				return "", 0, 0, false, false, cliError{"invalid --limit"}
			}
			if limit < 0 {
				return "", 0, 0, false, false, cliError{"--limit must be non-negative"}
			}
		case "--offset":
			val, verr := flagValue(args, &i, "--offset", inline, hasInline)
			if verr != nil {
				return "", 0, 0, false, false, verr
			}
			offset, err = strconv.Atoi(val)
			if err != nil {
				return "", 0, 0, false, false, cliError{"invalid --offset"}
			}
			if offset < 0 {
				return "", 0, 0, false, false, cliError{"--offset must be non-negative"}
			}
		default:
			if strings.HasPrefix(arg, "--") {
				return "", 0, 0, false, false, cliError{"unknown flag: " + arg}
			}
			if query != "" {
				return "", 0, 0, false, false, cliError{"too many arguments"}
			}
			query = arg
		}
	}
	if query == "" {
		return "", 0, 0, false, false, cliError{"missing query"}
	}
	return query, limit, offset, jsonOut, pretty, nil
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
		name, inline, hasInline := strings.Cut(arg, "=")
		switch name {
		case "--json":
			jsonOut = true
		case "--source":
			val, verr := flagValue(args, &i, "--source", inline, hasInline)
			if verr != nil {
				return "", false, verr
			}
			source = val
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

func parseLabelArgs(args []string) (name, text string, clear, jsonOut bool, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOut = true
		case "--clear":
			clear = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", false, false, cliError{"unknown flag: " + arg}
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) == 0 {
		return "", "", false, false, cliError{"missing name"}
	}
	name = positional[0]
	rest := positional[1:]
	if clear {
		if len(rest) != 0 {
			return "", "", false, false, cliError{"too many arguments"}
		}
		return name, "", true, jsonOut, nil
	}
	switch len(rest) {
	case 0:
		return "", "", false, false, cliError{"missing label text"}
	case 1:
		return name, rest[0], false, jsonOut, nil
	default:
		return "", "", false, false, cliError{"too many arguments"}
	}
}

func parseAliasArgs(args []string) (name, op, value string, jsonOut bool, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json":
			jsonOut = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", "", false, cliError{"unknown flag: " + arg}
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) != 3 {
		return "", "", "", false, cliError{"usage: clyph alias <name> <add|rm> <value>"}
	}
	name, op, value = positional[0], positional[1], positional[2]
	if op != "add" && op != "rm" {
		return "", "", "", false, cliError{"alias op must be 'add' or 'rm'"}
	}
	return name, op, value, jsonOut, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: clyph <search|get|glyph|codepoint|identify|fmt|semantic|families|stats|update|label|alias|version> ...")
}

var commandUsage = map[string]string{
	"search":    "usage: clyph search <query> [--limit N] [--offset N] [--json] [--pretty]",
	"get":       "usage: clyph get <name> [--json]",
	"glyph":     "usage: clyph glyph <name> [--json]",
	"codepoint": "usage: clyph codepoint <name> [--json]",
	"update":    "usage: clyph update [--source <file-or-url>] [--json]",
	"label":     "usage: clyph label <name> <text> [--json]\n       clyph label <name> --clear [--json]",
	"alias":     "usage: clyph alias <name> <add|rm> <value> [--json]",
	"identify":  "usage: clyph identify <glyph...> [--json]   (reads glyphs from stdin if none given)",
	"fmt":       "usage: clyph fmt <name> [--format html|css|unicode|js|hex|octal|all] [--json]",
	"semantic":  "usage: clyph semantic <concept> [--all] [--json]",
	"families":  "usage: clyph families [--limit N] [--json]",
	"stats":     "usage: clyph stats [--json]",
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
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
	if help, ok := commandUsage[cmd]; ok && hasHelpFlag(rest) {
		fmt.Println(help)
		return 0
	}
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
	case "label":
		return cmdLabel(rest)
	case "alias":
		return cmdAlias(rest)
	case "identify":
		return cmdIdentify(rest)
	case "fmt":
		return cmdFmt(rest)
	case "semantic":
		return cmdSemantic(rest)
	case "families":
		return cmdFamilies(rest)
	case "stats":
		return cmdStats(rest)
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
