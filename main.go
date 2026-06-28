package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultLimit   = 10
	defaultSource  = "https://www.nerdfonts.com/assets/css/webfont.css"
	catalogPathEnv = "CLYPH_CATALOG_PATH"
	dataDirEnv     = "CLYPH_DATA_DIR"
)

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

type recordView struct {
	Name      string   `json:"name"`
	Codepoint string   `json:"codepoint"`
	Unicode   string   `json:"unicode"`
	Glyph     string   `json:"glyph"`
	Label     string   `json:"label"`
	Aliases   []string `json:"aliases"`
}

type searchResponse struct {
	Query   string       `json:"query"`
	Matches []recordView `json:"matches"`
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

func catalogPath() string {
	if v := os.Getenv(catalogPathEnv); v != "" {
		return v
	}
	if v := os.Getenv(dataDirEnv); v != "" {
		return filepath.Join(v, "catalog.json")
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

func recordPayload(rec Record) recordView {
	return recordView{
		Name:      rec.Name,
		Codepoint: rec.Codepoint,
		Unicode:   rec.Unicode,
		Glyph:     rec.Glyph,
		Label:     rec.Label,
		Aliases:   rec.Aliases,
	}
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

func loadSource(source string) (string, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source) // #nosec G107 - intentional fetch for update
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeCSSEscape(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); {
		if value[i] != '\\' {
			out.WriteByte(value[i])
			i++
			continue
		}
		i++
		if i >= len(value) {
			break
		}
		hexStart := i
		for i < len(value) && i-hexStart < 6 {
			c := value[i]
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
				i++
				continue
			}
			break
		}
		if i > hexStart {
			v, err := strconv.ParseInt(value[hexStart:i], 16, 32)
			if err == nil {
				out.WriteRune(rune(v))
				if i < len(value) && isSpace(value[i]) {
					i++
				}
				continue
			}
		}
		out.WriteByte(value[i])
		i++
	}
	return out.String()
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func parseCSSCatalog(text string) ([]Record, error) {
	text = stripComments(text)
	records := map[string]Record{}
	for _, block := range findBlocks(text) {
		content, ok := extractContent(block.body)
		if !ok {
			continue
		}
		decoded := decodeCSSEscape(content)
		if decoded == "" {
			continue
		}
		codepointHex := fmt.Sprintf("%x", []rune(decoded)[0])
		aliases := extractClasses(block.selector)
		if len(aliases) == 0 {
			continue
		}
		for _, name := range aliases {
			rec, err := recordFromParts(name, codepointHex, nil)
			if err != nil {
				return nil, err
			}
			records[name] = rec
		}
	}
	out := make([]Record, 0, len(records))
	for _, rec := range records {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type cssBlock struct {
	selector string
	body     string
}

func stripComments(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		if i+1 < len(text) && text[i] == '/' && text[i+1] == '*' {
			i += 2
			for i+1 < len(text) && !(text[i] == '*' && text[i+1] == '/') {
				i++
			}
			if i+1 < len(text) {
				i += 2
			}
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

func findBlocks(text string) []cssBlock {
	var blocks []cssBlock
	for {
		open := strings.IndexByte(text, '{')
		if open < 0 {
			break
		}
		close := strings.IndexByte(text[open+1:], '}')
		if close < 0 {
			break
		}
		close += open + 1
		selector := strings.TrimSpace(text[:open])
		body := text[open+1 : close]
		blocks = append(blocks, cssBlock{selector: selector, body: body})
		text = text[close+1:]
	}
	return blocks
}

func extractContent(body string) (string, bool) {
	idx := strings.Index(body, "content")
	if idx < 0 {
		return "", false
	}
	rest := body[idx+len("content"):]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return "", false
	}
	rest = rest[colon+1:]
	rest = strings.TrimSpace(rest)
	if len(rest) == 0 {
		return "", false
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return "", false
	}
	rest = rest[1:]
	end := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == quote && (i == 0 || rest[i-1] != '\\') {
			end = i
			break
		}
	}
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

func extractClasses(selector string) []string {
	parts := strings.Split(selector, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, ".") {
			part = part[1:]
		}
		for i := 0; i < len(part); i++ {
			if part[i] == ':' {
				part = part[:i]
				break
			}
		}
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

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
		resp := searchResponse{Query: query, Matches: recordsToPayload(matches)}
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
		if err := printJSON(recordPayload(rec)); err != nil {
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

func recordsToPayload(records []Record) []recordView {
	out := make([]recordView, 0, len(records))
	for _, rec := range records {
		out = append(out, recordPayload(rec))
	}
	return out
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
	fmt.Fprintln(os.Stderr, "usage: clyph <search|get|glyph|codepoint|update> ...")
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
	case "-h", "--help", "help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		return 2
	}
}
