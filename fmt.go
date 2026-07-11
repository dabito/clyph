// fmt.go: format a glyph codepoint into html/css/unicode/js/hex/octal.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type fmtResponse struct {
	Name      string            `json:"name"`
	Codepoint string            `json:"codepoint"`
	Formats   map[string]string `json:"formats"`
}

var validFormats = map[string]bool{
	"html": true, "css": true, "unicode": true,
	"js": true, "hex": true, "octal": true, "all": true,
}

var formatOrder = []string{"html", "css", "unicode", "js", "hex", "octal"}

// formatCodepoint formats rec's codepoint into the named representation.
// which must be one of: html, css, unicode, js, hex, octal.
func formatCodepoint(rec Record, which string) string {
	v, _ := strconv.ParseInt(rec.Codepoint, 16, 32)
	r := rune(v)
	switch which {
	case "html":
		return fmt.Sprintf("&#x%s;", rec.Codepoint)
	case "css":
		return fmt.Sprintf(`content: "\%s";`, rec.Codepoint)
	case "unicode", "js":
		return unicodeEscape(r)
	case "hex":
		return "0x" + rec.Codepoint
	case "octal":
		return fmt.Sprintf("%o", r)
	}
	return ""
}

func allFormatMap(rec Record) map[string]string {
	m := make(map[string]string, len(formatOrder))
	for _, f := range formatOrder {
		m[f] = formatCodepoint(rec, f)
	}
	return m
}

func parseFmtArgs(args []string) (name, format string, jsonOut bool, err error) {
	format = "all"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		argName, inline, hasInline := strings.Cut(arg, "=")
		switch argName {
		case "--json":
			jsonOut = true
		case "--format":
			val, verr := flagValue(args, &i, "--format", inline, hasInline)
			if verr != nil {
				return "", "", false, verr
			}
			if !validFormats[val] {
				return "", "", false, cliError{"unknown format: " + val}
			}
			format = val
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", false, cliError{"unknown flag: " + arg}
			}
			if name != "" {
				return "", "", false, cliError{"too many arguments"}
			}
			name = arg
		}
	}
	if name == "" {
		return "", "", false, cliError{"missing name"}
	}
	return name, format, jsonOut, nil
}

func cmdFmt(args []string) int {
	name, format, jsonOut, err := parseFmtArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, lerr := loadRecords(catalogPath())
	if lerr != nil {
		printError(jsonOut, lerr.Error())
		return 1
	}
	rec, ok := buildIndex(records)[name]
	if !ok {
		printError(jsonOut, fmt.Sprintf("not found: %s", name))
		return 1
	}
	if jsonOut {
		resp := fmtResponse{
			Name:      rec.Name,
			Codepoint: rec.Codepoint,
			Formats:   allFormatMap(rec),
		}
		if perr := printJSON(resp); perr != nil {
			printError(true, perr.Error())
			return 1
		}
		return 0
	}
	if format == "all" {
		for _, f := range formatOrder {
			fmt.Printf("%s\t%s\n", f, formatCodepoint(rec, f))
		}
		return 0
	}
	fmt.Println(formatCodepoint(rec, format))
	return 0
}
