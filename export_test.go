package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func exportEnv(t *testing.T) map[string]string {
	t.Helper()
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	return map[string]string{catalogPathEnv: catalog}
}

func TestExportJSONNames(t *testing.T) {
	code, out, errOut := captureCmd(t, cmdExport, []string{"--names", "nf-md-check,nf-fa-circle", "--format", "json"}, exportEnv(t))
	if code != 0 {
		t.Fatalf("export failed: %s", errOut)
	}
	var resp exportResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out)
	}
	if resp.Format != "json" || resp.Count != 2 || len(resp.Records) != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Records[0].Name != "nf-fa-circle" || resp.Records[1].Name != "nf-md-check" {
		t.Fatalf("records not sorted/deterministic: %+v", resp.Records)
	}
}

func TestExportCSSFamily(t *testing.T) {
	code, out, errOut := captureCmd(t, cmdExport, []string{"--family", "md", "--format", "css"}, exportEnv(t))
	if code != 0 {
		t.Fatalf("export css failed: %s", errOut)
	}
	if !strings.Contains(out, `--clyph-nf-md-check: "\f012c";`) {
		t.Fatalf("missing md check variable:\n%s", out)
	}
	if strings.Contains(out, "nf-fa-check") {
		t.Fatalf("family filter leaked fa record:\n%s", out)
	}
}

func TestExportTSAndGo(t *testing.T) {
	env := exportEnv(t)
	code, out, errOut := captureCmd(t, cmdExport, []string{"--names", "nf-md-check", "--format", "ts"}, env)
	if code != 0 {
		t.Fatalf("export ts failed: %s", errOut)
	}
	if !strings.Contains(out, `export const glyphs`) || !strings.Contains(out, `"nf-md-check"`) || !strings.Contains(out, `export type GlyphName`) {
		t.Fatalf("unexpected ts output:\n%s", out)
	}
	code, out, errOut = captureCmd(t, cmdExport, []string{"--names", "nf-md-check", "--format", "go"}, env)
	if code != 0 {
		t.Fatalf("export go failed: %s", errOut)
	}
	if !strings.Contains(out, `package clyphicons`) || !strings.Contains(out, `"nf-md-check"`) {
		t.Fatalf("unexpected go output:\n%s", out)
	}
}

func TestExportSemanticAndOutput(t *testing.T) {
	env := exportEnv(t)
	outPath := filepath.Join(t.TempDir(), "icons.json")
	code, out, errOut := captureCmd(t, cmdExport, []string{"--semantic", "success", "--output", outPath}, env)
	if code != 0 {
		t.Fatalf("export semantic failed: %s", errOut)
	}
	if out != "" {
		t.Fatalf("expected no stdout when --output used, got %q", out)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"nf-md-check"`) {
		t.Fatalf("semantic export did not contain canonical check:\n%s", data)
	}
}

func TestExportErrors(t *testing.T) {
	env := exportEnv(t)
	if code, _, errOut := captureCmd(t, cmdExport, []string{"--format", "yaml"}, env); code != 2 || !strings.Contains(errOut, "invalid --format") {
		t.Fatalf("expected invalid format usage, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := captureCmd(t, cmdExport, []string{"--names", "missing"}, env); code != 1 || !strings.Contains(errOut, "not found: missing") {
		t.Fatalf("expected missing name error, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := captureCmd(t, cmdExport, []string{"--wat"}, env); code != 2 || !strings.Contains(errOut, "unknown flag") {
		t.Fatalf("expected unknown flag, code=%d err=%q", code, errOut)
	}
}

func TestRunExportHelp(t *testing.T) {
	code, out, errOut := runAndCapture(t, []string{"export", "--help"}, nil)
	if code != 0 || errOut != "" || !strings.Contains(out, "usage: clyph export") {
		t.Fatalf("expected export help, code=%d out=%q err=%q", code, out, errOut)
	}
}
