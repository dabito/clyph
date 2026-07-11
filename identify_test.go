package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// identifyAndCapture runs cmdIdentify directly, capturing stdout/stderr.
// If stdinData is non-empty, os.Stdin is replaced with a reader over that data.
func identifyAndCapture(t *testing.T, args []string, stdinData string, env map[string]string) (int, string, string) {
	t.Helper()
	oldOut, oldErr, oldIn := os.Stdout, os.Stderr, os.Stdin
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
		os.Stdin = oldIn
	}()

	if stdinData != "" {
		inR, inW, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := inW.WriteString(stdinData); err != nil {
			t.Fatal(err)
		}
		inW.Close()
		os.Stdin = inR
	}

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
		codeCh <- cmdIdentify(args)
		_ = outW.Close()
		_ = errW.Close()
	}()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(outR)
	_, _ = errBuf.ReadFrom(errR)
	return <-codeCh, outBuf.String(), errBuf.String()
}

func TestGlyphFamily(t *testing.T) {
	cases := []struct{ in, want string }{
		{"nf-md-check", "md"},
		{"nf-cod-account", "cod"},
		{"nf-fa-circle", "fa"},
		{"nf-fa-circle_o", "fa"},
		{"nope", ""},
		{"nf-", ""},
		{"nf-x", ""},
	}
	for _, tc := range cases {
		if got := glyphFamily(tc.in); got != tc.want {
			t.Errorf("glyphFamily(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFindRecordsByGlyph(t *testing.T) {
	got := findRecordsByGlyph(fixtureRecords, "")
	if len(got) != 1 || got[0].Name != "nf-fa-circle" {
		t.Fatalf("unexpected matches: %#v", got)
	}

	got = findRecordsByGlyph(fixtureRecords, "notexist")
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %#v", got)
	}
}

func TestIdentifyExactMatch(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	// nf-fa-circle has glyph 
	code, out, errOut := identifyAndCapture(t, []string{""}, "", env)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, errOut)
	}
	line := strings.TrimSpace(out)
	// expected: name TAB codepoint TAB family TAB glyph
	want := "nf-fa-circle\tf111\tfa\t"
	if line != want {
		t.Fatalf("output = %q, want %q", line, want)
	}
}

func TestIdentifyNoMatch(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	code, out, errOut := identifyAndCapture(t, []string{"X"}, "", env)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, errOut)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected empty output for no match, got %q", out)
	}
}

func TestIdentifyMissingCatalog(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.json")
	env := map[string]string{catalogPathEnv: missing}

	code, _, errOut := identifyAndCapture(t, []string{""}, "", env)
	if code != 1 {
		t.Fatalf("expected exit 1 for missing catalog, got %d (err=%q)", code, errOut)
	}
	if !strings.Contains(errOut, "run 'clyph update' first") {
		t.Fatalf("expected actionable error, got %q", errOut)
	}
}

func TestIdentifyUnknownFlag(t *testing.T) {
	code, _, errOut := identifyAndCapture(t, []string{"--bogus"}, "", nil)
	if code != 2 {
		t.Fatalf("expected exit 2 for unknown flag, got %d", code)
	}
	if !strings.Contains(errOut, "unknown flag: --bogus") {
		t.Fatalf("expected unknown flag message, got %q", errOut)
	}
}

func TestIdentifyEmptyInput(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	// No args and empty stdin (stdinData="" means stdin is not replaced; we
	// need to supply actual empty stdin to avoid blocking on a real terminal).
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	inW.Close()
	oldIn := os.Stdin
	os.Stdin = inR
	defer func() { os.Stdin = oldIn }()

	code, _, errOut := identifyAndCapture(t, []string{}, "", env)
	if code != 2 {
		t.Fatalf("expected exit 2 for empty input, got %d (err=%q)", code, errOut)
	}
	if !strings.Contains(errOut, "missing glyph input") {
		t.Fatalf("expected missing input error, got %q", errOut)
	}
}

func TestIdentifyJSON(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	code, out, errOut := identifyAndCapture(t, []string{"", "--json"}, "", env)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, errOut)
	}
	var resp identifyResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v (output=%q)", err, out)
	}
	if resp.Query != "" {
		t.Fatalf("query = %q, want %q", resp.Query, "")
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Name != "nf-fa-circle" {
		t.Fatalf("unexpected matches: %#v", resp.Matches)
	}
}

func TestIdentifyJSONNoMatch(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	code, out, errOut := identifyAndCapture(t, []string{"X", "--json"}, "", env)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, errOut)
	}
	var resp identifyResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if len(resp.Matches) != 0 {
		t.Fatalf("expected empty matches array, got %#v", resp.Matches)
	}
	// Ensure matches is [] not null in JSON.
	if !strings.Contains(out, `"matches": []`) {
		t.Fatalf("expected empty JSON array for matches, got %q", out)
	}
}

func TestIdentifyStdin(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	env := map[string]string{catalogPathEnv: catalog}

	// Pipe the glyph character via stdin.
	code, out, errOut := identifyAndCapture(t, []string{}, "\n", env)
	if code != 0 {
		t.Fatalf("unexpected exit %d: %s", code, errOut)
	}
	line := strings.TrimSpace(out)
	want := "nf-fa-circle\tf111\tfa\t"
	if line != want {
		t.Fatalf("stdin output = %q, want %q", line, want)
	}
}
