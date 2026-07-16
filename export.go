package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strings"
)

type exportOptions struct {
	Format    string
	Names     []string
	Family    string
	Semantics []string
	Sets      []string
	Output    string
}

type exportResponse struct {
	Format  string   `json:"format"`
	Count   int      `json:"count"`
	Records []Record `json:"records"`
}

func parseExportArgs(args []string) (exportOptions, error) {
	opts := exportOptions{Format: "json"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, inline, hasInline := strings.Cut(arg, "=")
		switch name {
		case "--format":
			val, err := flagValue(args, &i, "--format", inline, hasInline)
			if err != nil {
				return opts, err
			}
			opts.Format = strings.ToLower(strings.TrimSpace(val))
		case "--names":
			val, err := flagValue(args, &i, "--names", inline, hasInline)
			if err != nil {
				return opts, err
			}
			opts.Names = splitCSV(val)
		case "--family":
			val, err := flagValue(args, &i, "--family", inline, hasInline)
			if err != nil {
				return opts, err
			}
			opts.Family = strings.TrimSpace(strings.TrimPrefix(val, "nf-"))
		case "--semantic":
			val, err := flagValue(args, &i, "--semantic", inline, hasInline)
			if err != nil {
				return opts, err
			}
			opts.Semantics = splitCSV(val)
		case "--set":
			val, err := flagValue(args, &i, "--set", inline, hasInline)
			if err != nil {
				return opts, err
			}
			opts.Sets = splitCSV(val)
		case "--output", "-o":
			val, err := flagValue(args, &i, name, inline, hasInline)
			if err != nil {
				return opts, err
			}
			opts.Output = strings.TrimSpace(val)
		default:
			if strings.HasPrefix(arg, "--") {
				return opts, cliError{"unknown flag: " + arg}
			}
			return opts, cliError{"too many arguments"}
		}
	}
	if opts.Format == "" {
		opts.Format = "json"
	}
	switch opts.Format {
	case "json", "css", "ts", "go":
		return opts, nil
	default:
		return opts, cliError{"invalid --format: " + opts.Format}
	}
}

func splitCSV(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func selectExportRecords(records []Record, opts exportOptions) ([]Record, error) {
	idx := buildIndex(records)
	selected := map[string]Record{}
	add := func(rec Record) { selected[rec.Name] = rec }
	selecting := len(opts.Names) > 0 || opts.Family != "" || len(opts.Semantics) > 0 || len(opts.Sets) > 0
	if !selecting {
		for _, rec := range records {
			add(rec)
		}
	}
	for _, name := range opts.Names {
		rec, ok := idx[name]
		if !ok {
			return nil, cliError{"not found: " + name}
		}
		add(rec)
	}
	if opts.Family != "" {
		family := strings.TrimSpace(opts.Family)
		for _, rec := range records {
			if glyphFamily(rec.Name) == family {
				add(rec)
			}
		}
	}
	if len(opts.Semantics) > 0 {
		seed, err := loadSemanticSeed()
		if err != nil {
			return nil, err
		}
		for _, concept := range opts.Semantics {
			matches := resolveSemantic(seed, records, strings.ToLower(strings.TrimSpace(concept)))
			if len(matches) == 0 {
				return nil, cliError{"no semantic match for: " + concept}
			}
			add(matches[0].Record)
		}
	}
	if len(opts.Sets) > 0 {
		sets, err := loadSets()
		if err != nil {
			return nil, err
		}
		for _, setName := range opts.Sets {
			set, ok := sets[normalizeSetName(setName)]
			if !ok {
				return nil, cliError{"set not found: " + setName}
			}
			for _, glyphName := range set.Glyphs {
				rec, ok := idx[glyphName]
				if !ok {
					return nil, cliError{"set references missing glyph: " + glyphName}
				}
				add(rec)
			}
		}
	}
	out := make([]Record, 0, len(selected))
	for _, rec := range selected {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func renderExport(records []Record, formatName string) ([]byte, error) {
	switch formatName {
	case "json":
		data, err := json.MarshalIndent(exportResponse{Format: "json", Count: len(records), Records: records}, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	case "css":
		return []byte(renderExportCSS(records)), nil
	case "ts":
		return []byte(renderExportTS(records)), nil
	case "go":
		return renderExportGo(records)
	default:
		return nil, cliError{"invalid --format: " + formatName}
	}
}

func renderExportCSS(records []Record) string {
	var b strings.Builder
	b.WriteString(":root {\n")
	for _, rec := range records {
		fmt.Fprintf(&b, "  --clyph-%s: \"\\%s\";\n", cssIdent(rec.Name), rec.Codepoint)
	}
	b.WriteString("}\n")
	for _, rec := range records {
		fmt.Fprintf(&b, ".clyph-%s::before { content: var(--clyph-%s); }\n", cssIdent(rec.Name), cssIdent(rec.Name))
	}
	return b.String()
}

func cssIdent(name string) string {
	return strings.NewReplacer("_", "-", ":", "-").Replace(strings.ToLower(name))
}

func renderExportTS(records []Record) string {
	var b strings.Builder
	b.WriteString("export const glyphs = {\n")
	for _, rec := range records {
		fmt.Fprintf(&b, "  %s: { glyph: %s, codepoint: %s, unicode: %s },\n", tsString(rec.Name), tsString(rec.Glyph), tsString(rec.Codepoint), tsString(rec.Unicode))
	}
	b.WriteString("} as const;\n\n")
	b.WriteString("export type GlyphName = keyof typeof glyphs;\n")
	return b.String()
}

func tsString(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func renderExportGo(records []Record) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("package clyphicons\n\n")
	b.WriteString("type Glyph struct {\n\tGlyph string\n\tCodepoint string\n\tUnicode string\n}\n\n")
	b.WriteString("var Glyphs = map[string]Glyph{\n")
	for _, rec := range records {
		fmt.Fprintf(&b, "\t%q: {Glyph: %q, Codepoint: %q, Unicode: %q},\n", rec.Name, rec.Glyph, rec.Codepoint, rec.Unicode)
	}
	b.WriteString("}\n")
	out, err := format.Source(b.Bytes())
	if err != nil {
		return nil, err
	}
	return out, nil
}

func writeExport(data []byte, output string) error {
	if output == "" || output == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return saveBytesAtomic(output, data)
}

func cmdExport(args []string) int {
	opts, err := parseExportArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, err := loadRecords(catalogPath())
	if err != nil {
		printError(false, err.Error())
		return 1
	}
	selected, err := selectExportRecords(records, opts)
	if err != nil {
		printError(false, err.Error())
		return 1
	}
	data, err := renderExport(selected, opts.Format)
	if err != nil {
		printError(false, err.Error())
		return 1
	}
	if err := writeExport(data, opts.Output); err != nil {
		printError(false, err.Error())
		return 1
	}
	return 0
}
