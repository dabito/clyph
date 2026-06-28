package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var fixtureRecords = []Record{
	{Name: "nf-fa-check", Codepoint: "f00c", Unicode: "\\uf00c", Glyph: "\uf00c", Label: "done", Aliases: []string{"done"}},
	{Name: "nf-fa-circle", Codepoint: "f111", Unicode: "\\uf111", Glyph: "\uf111", Label: "offline", Aliases: []string{"offline"}},
	{Name: "nf-fa-circle_o", Codepoint: "f10c", Unicode: "\\uf10c", Glyph: "\uf10c", Label: "offline outline", Aliases: []string{"offline-outline"}},
	{Name: "nf-md-check", Codepoint: "f012c", Unicode: "\\U000f012c", Glyph: "\U000f012c", Label: "check", Aliases: []string{}},
	{Name: "nf-md-circle_half", Codepoint: "f1395", Unicode: "\\U000f1395", Glyph: "\U000f1395", Label: "progress", Aliases: []string{"progress"}},
}

func writeCatalog(t *testing.T, path string, records []Record) {
	t.Helper()
	payload := catalogFile{Records: records}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runAndCapture(t *testing.T, args []string, env map[string]string) (int, string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = outW
	os.Stderr = errW
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()
	if env != nil {
		oldEnv := map[string]string{}
		wasSet := map[string]bool{}
		for k, v := range env {
			oldEnv[k], wasSet[k] = os.LookupEnv(k)
			if err := os.Setenv(k, v); err != nil {
				t.Fatal(err)
			}
		}
		defer func() {
			for k, v := range oldEnv {
				var err error
				if wasSet[k] {
					err = os.Setenv(k, v)
				} else {
					err = os.Unsetenv(k)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}
	codeCh := make(chan int, 1)
	go func() {
		codeCh <- run(args)
		_ = outW.Close()
		_ = errW.Close()
	}()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(outR)
	_, _ = errBuf.ReadFrom(errR)
	return <-codeCh, outBuf.String(), errBuf.String()
}

func TestCatalogMetadata(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("data", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var payload catalogFile
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Records) <= 10000 {
		t.Fatalf("expected large catalog, got %d records", len(payload.Records))
	}
	lookup := buildIndex(payload.Records)
	if lookup["nf-md-check"].Codepoint != "f012c" {
		t.Fatalf("unexpected nf-md-check codepoint: %s", lookup["nf-md-check"].Codepoint)
	}
	if lookup["nf-fa-circle"].Codepoint != "f111" {
		t.Fatalf("unexpected nf-fa-circle codepoint: %s", lookup["nf-fa-circle"].Codepoint)
	}
}

func TestCatalogPathDefaultsAndEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv(catalogPathEnv, "")
	t.Setenv(dataDirEnv, "")

	wantDefault := filepath.Join(tmp, ".clyph", "data", "catalog.json")
	if got := catalogPath(); got != wantDefault {
		t.Fatalf("default catalogPath() = %q want %q", got, wantDefault)
	}

	customData := filepath.Join(tmp, "custom-data")
	t.Setenv(dataDirEnv, customData)
	wantData := filepath.Join(customData, "catalog.json")
	if got := catalogPath(); got != wantData {
		t.Fatalf("data-dir catalogPath() = %q want %q", got, wantData)
	}

	exact := filepath.Join(tmp, "exact.json")
	t.Setenv(catalogPathEnv, exact)
	if got := catalogPath(); got != exact {
		t.Fatalf("catalog override catalogPath() = %q want %q", got, exact)
	}
}
func TestSearchAndScalarCommands(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	code, out, errOut := runAndCapture(t, []string{"search", "circle"}, env)
	if code != 0 {
		t.Fatalf("search failed: %s", errOut)
	}
	got := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{
		"nf-fa-circle\tf111\t\uf111\toffline",
		"nf-fa-circle_o\tf10c\t\uf10c\toffline outline",
		"nf-md-circle_half\tf1395\t\U000f1395\tprogress",
	}
	if len(got) != len(want) {
		t.Fatalf("search results mismatch: %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("search[%d] = %q want %q", i, got[i], want[i])
		}
	}

	code, out, errOut = runAndCapture(t, []string{"search", "check", "--json"}, env)
	if code != 0 {
		t.Fatalf("search json failed: %s", errOut)
	}
	var searchPayload searchResponse
	if err := json.Unmarshal([]byte(out), &searchPayload); err != nil {
		t.Fatal(err)
	}
	if searchPayload.Query != "check" || len(searchPayload.Matches) != 2 {
		t.Fatalf("unexpected search payload: %#v", searchPayload)
	}
	if searchPayload.Matches[1].Unicode != "\\U000f012c" {
		t.Fatalf("unexpected unicode: %s", searchPayload.Matches[1].Unicode)
	}

	code, out, errOut = runAndCapture(t, []string{"get", "nf-md-check"}, env)
	if code != 0 || strings.TrimSpace(out) != "nf-md-check\tf012c\t\U000f012c\tcheck" {
		t.Fatalf("get failed: %q %s", out, errOut)
	}

	code, out, errOut = runAndCapture(t, []string{"glyph", "nf-md-check"}, env)
	if code != 0 || strings.TrimSpace(out) != "\U000f012c" {
		t.Fatalf("glyph failed: %q %s", out, errOut)
	}

	code, out, errOut = runAndCapture(t, []string{"glyph", "nf-md-check", "--json"}, env)
	if code != 0 {
		t.Fatalf("glyph json failed: %s", errOut)
	}
	var glyphPayload glyphResponse
	if err := json.Unmarshal([]byte(out), &glyphPayload); err != nil {
		t.Fatal(err)
	}
	if glyphPayload != (glyphResponse{Name: "nf-md-check", Glyph: "\U000f012c"}) {
		t.Fatalf("unexpected glyph payload: %#v", glyphPayload)
	}

	code, out, errOut = runAndCapture(t, []string{"codepoint", "nf-md-check"}, env)
	if code != 0 || strings.TrimSpace(out) != "f012c" {
		t.Fatalf("codepoint failed: %q %s", out, errOut)
	}

	code, out, errOut = runAndCapture(t, []string{"codepoint", "nf-md-check", "--json"}, env)
	if code != 0 {
		t.Fatalf("codepoint json failed: %s", errOut)
	}
	var codePayload codepointResponse
	if err := json.Unmarshal([]byte(out), &codePayload); err != nil {
		t.Fatal(err)
	}
	if codePayload != (codepointResponse{Name: "nf-md-check", Codepoint: "f012c"}) {
		t.Fatalf("unexpected codepoint payload: %#v", codePayload)
	}
}

func TestCSSParserAndUpdate(t *testing.T) {
	css := `
/* comment */
.nf-a:before,
.nf-b::before { content: "\f111"; }
.nf-a:before { content : "\f222"; }
.nf-c:before,
.nf-d:before { content: "\f1395"; }
`
	recs, err := parseCSSCatalog(css)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(recs); got != 4 {
		t.Fatalf("unexpected record count: %d", got)
	}
	idx := buildIndex(recs)
	if idx["nf-a"].Codepoint != "f222" || idx["nf-c"].Glyph != "\U000f1395" {
		t.Fatalf("unexpected parse result: %#v %#v", idx["nf-a"], idx["nf-c"])
	}

	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, []Record{{Name: "nf-x", Codepoint: "f000", Unicode: "\\uf000", Glyph: "\uf000", Label: "keep", Aliases: []string{"alias"}}})
	cssPath := filepath.Join(tmp, "webfont.css")
	if err := os.WriteFile(cssPath, []byte(`.nf-x:before { content: "\f123"; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runAndCapture(t, []string{"update", "--source", cssPath, "--json"}, map[string]string{catalogPathEnv: catalog})
	if code != 0 {
		t.Fatalf("update failed: %s", errOut)
	}
	var payload updateResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Records != 1 || payload.Catalog != catalog {
		t.Fatalf("unexpected update response: %#v", payload)
	}
	updated, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var catalogPayload catalogFile
	if err := json.Unmarshal(updated, &catalogPayload); err != nil {
		t.Fatal(err)
	}
	if len(catalogPayload.Records) != 1 || catalogPayload.Records[0].Label != "keep" || len(catalogPayload.Records[0].Aliases) != 1 {
		t.Fatalf("metadata not preserved: %#v", catalogPayload.Records[0])
	}

	before := string(updated)
	bad := filepath.Join(tmp, "missing.css")
	code, _, errOut = runAndCapture(t, []string{"update", "--source", bad, "--json"}, map[string]string{catalogPathEnv: catalog})
	if code == 0 || !strings.Contains(errOut, "update failed") {
		t.Fatalf("expected failed update, got code=%d err=%q", code, errOut)
	}
	after, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatal("catalog changed after failed update")
	}
}

func TestArgumentAndUpdateGuards(t *testing.T) {
	if glyph, err := codepointToGlyph("0xf012c"); err != nil || glyph != "\U000f012c" {
		t.Fatalf("0x-prefixed codepoint failed: glyph=%q err=%v", glyph, err)
	}

	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	code, _, errOut := runAndCapture(t, []string{"search", "circle", "--wat"}, env)
	if code == 0 || !strings.Contains(errOut, "unknown flag: --wat") {
		t.Fatalf("expected unknown flag error, code=%d err=%q", code, errOut)
	}

	before, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	emptyCSS := filepath.Join(tmp, "empty.css")
	if err := os.WriteFile(emptyCSS, []byte("body { color: red; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut = runAndCapture(t, []string{"update", "--source", emptyCSS, "--json"}, env)
	if code == 0 || !strings.Contains(errOut, "no glyph records parsed") {
		t.Fatalf("expected empty update rejection, code=%d err=%q", code, errOut)
	}
	after, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("catalog changed after empty update")
	}
}
