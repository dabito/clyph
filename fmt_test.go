package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fmtCapture runs cmdFmt directly and captures stdout/stderr.
func fmtCapture(t *testing.T, args []string, env map[string]string) (code int, stdout, stderr string) {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
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
	codeCh := make(chan int, 1)
	go func() {
		codeCh <- cmdFmt(args)
		_ = outW.Close()
		_ = errW.Close()
	}()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(outR)
	_, _ = errBuf.ReadFrom(errR)
	return <-codeCh, outBuf.String(), errBuf.String()
}

func setupFmtCatalog(t *testing.T) map[string]string {
	t.Helper()
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	return map[string]string{catalogPathEnv: catalog}
}

func TestFormatCodepointValues(t *testing.T) {
	// nf-fa-check: codepoint f00c (≤ 0xFFFF)
	faCheck := Record{Name: "nf-fa-check", Codepoint: "f00c"}
	faCases := []struct{ which, want string }{
		{"html", "&#xf00c;"},
		{"css", "content: \"\\f00c\";"},
		{"unicode", "\\uf00c"},
		{"js", "\\uf00c"},
		{"hex", "0xf00c"},
		{"octal", "170014"},
	}
	for _, tc := range faCases {
		if got := formatCodepoint(faCheck, tc.which); got != tc.want {
			t.Errorf("nf-fa-check %s: got %q want %q", tc.which, got, tc.want)
		}
	}

	// nf-md-check: codepoint f012c (> 0xFFFF)
	mdCheck := Record{Name: "nf-md-check", Codepoint: "f012c"}
	mdCases := []struct{ which, want string }{
		{"html", "&#xf012c;"},
		{"css", "content: \"\\f012c\";"},
		{"unicode", "\\U000f012c"},
		{"js", "\\U000f012c"},
		{"hex", "0xf012c"},
		{"octal", "3600454"},
	}
	for _, tc := range mdCases {
		if got := formatCodepoint(mdCheck, tc.which); got != tc.want {
			t.Errorf("nf-md-check %s: got %q want %q", tc.which, got, tc.want)
		}
	}
}

func TestCmdFmtAllDefault(t *testing.T) {
	env := setupFmtCatalog(t)
	code, out, errOut := fmtCapture(t, []string{"nf-fa-check"}, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, errOut)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 6 {
		t.Fatalf("expected 6 labeled lines, got %d: %q", len(lines), lines)
	}
	wantLines := []string{
		"html\t&#xf00c;",
		"css\tcontent: \"\\f00c\";",
		"unicode\t\\uf00c",
		"js\t\\uf00c",
		"hex\t0xf00c",
		"octal\t170014",
	}
	for i, want := range wantLines {
		if lines[i] != want {
			t.Errorf("line[%d]: got %q want %q", i, lines[i], want)
		}
	}
}

func TestCmdFmtSingleFormat(t *testing.T) {
	env := setupFmtCatalog(t)
	singleCases := []struct {
		format, name, want string
	}{
		{"html", "nf-fa-check", "&#xf00c;"},
		{"css", "nf-fa-check", "content: \"\\f00c\";"},
		{"unicode", "nf-fa-check", "\\uf00c"},
		{"js", "nf-fa-check", "\\uf00c"},
		{"hex", "nf-fa-check", "0xf00c"},
		{"octal", "nf-fa-check", "170014"},
		{"html", "nf-md-check", "&#xf012c;"},
		{"css", "nf-md-check", "content: \"\\f012c\";"},
		{"unicode", "nf-md-check", "\\U000f012c"},
		{"js", "nf-md-check", "\\U000f012c"},
		{"hex", "nf-md-check", "0xf012c"},
		{"octal", "nf-md-check", "3600454"},
	}
	for _, tc := range singleCases {
		code, out, errOut := fmtCapture(t, []string{"--format", tc.format, tc.name}, env)
		if code != 0 {
			t.Errorf("%s %s: exit %d: %s", tc.name, tc.format, code, errOut)
			continue
		}
		if got := strings.TrimSpace(out); got != tc.want {
			t.Errorf("%s %s: got %q want %q", tc.name, tc.format, got, tc.want)
		}
	}
}

func TestCmdFmtJSON(t *testing.T) {
	env := setupFmtCatalog(t)

	// nf-fa-check exact values
	code, out, errOut := fmtCapture(t, []string{"--json", "nf-fa-check"}, env)
	if code != 0 {
		t.Fatalf("json nf-fa-check: exit %d: %s", code, errOut)
	}
	var resp fmtResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("nf-fa-check: JSON parse error: %v — output: %q", err, out)
	}
	if resp.Name != "nf-fa-check" || resp.Codepoint != "f00c" {
		t.Errorf("nf-fa-check: unexpected name/codepoint: %q %q", resp.Name, resp.Codepoint)
	}
	wantFa := map[string]string{
		"html":    "&#xf00c;",
		"css":     "content: \"\\f00c\";",
		"unicode": "\\uf00c",
		"js":      "\\uf00c",
		"hex":     "0xf00c",
		"octal":   "170014",
	}
	for k, want := range wantFa {
		if resp.Formats[k] != want {
			t.Errorf("JSON nf-fa-check %s: got %q want %q", k, resp.Formats[k], want)
		}
	}

	// nf-md-check exact values
	code, out, errOut = fmtCapture(t, []string{"--json", "nf-md-check"}, env)
	if code != 0 {
		t.Fatalf("json nf-md-check: exit %d: %s", code, errOut)
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("nf-md-check: JSON parse error: %v — output: %q", err, out)
	}
	if resp.Name != "nf-md-check" || resp.Codepoint != "f012c" {
		t.Errorf("nf-md-check: unexpected name/codepoint: %q %q", resp.Name, resp.Codepoint)
	}
	wantMd := map[string]string{
		"html":    "&#xf012c;",
		"css":     "content: \"\\f012c\";",
		"unicode": "\\U000f012c",
		"js":      "\\U000f012c",
		"hex":     "0xf012c",
		"octal":   "3600454",
	}
	for k, want := range wantMd {
		if resp.Formats[k] != want {
			t.Errorf("JSON nf-md-check %s: got %q want %q", k, resp.Formats[k], want)
		}
	}

	// Confirm all 6 format keys are present regardless of --format flag
	for _, f := range formatOrder {
		if _, ok := resp.Formats[f]; !ok {
			t.Errorf("JSON nf-md-check missing format key %q", f)
		}
	}
}

func TestCmdFmtErrors(t *testing.T) {
	env := setupFmtCatalog(t)

	// not-found exits 1
	code, _, errOut := fmtCapture(t, []string{"nf-does-not-exist"}, env)
	if code != 1 {
		t.Errorf("not-found: expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut, "not found: nf-does-not-exist") {
		t.Errorf("not-found: unexpected stderr %q", errOut)
	}

	// unknown flag exits 2
	code, _, errOut = fmtCapture(t, []string{"nf-fa-check", "--wat"}, env)
	if code != 2 {
		t.Errorf("unknown flag: expected exit 2, got %d", code)
	}
	if !strings.Contains(errOut, "unknown flag: --wat") {
		t.Errorf("unknown flag: unexpected stderr %q", errOut)
	}

	// missing name exits 2
	code, _, errOut = fmtCapture(t, []string{}, env)
	if code != 2 {
		t.Errorf("missing name: expected exit 2, got %d", code)
	}
	if !strings.Contains(errOut, "missing name") {
		t.Errorf("missing name: unexpected stderr %q", errOut)
	}
}
