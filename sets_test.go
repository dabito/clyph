package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSets(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedSetGlyphsExist(t *testing.T) {
	sets, err := parseSets(embeddedSets)
	if err != nil {
		t.Fatal(err)
	}
	records, err := loadRecordsJSON(filepath.Join("data", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	idx := buildIndex(records)
	for setName, set := range sets {
		for key, glyphName := range set.Glyphs {
			if _, ok := idx[glyphName]; !ok {
				t.Fatalf("embedded set %s/%s references missing glyph %s", setName, key, glyphName)
			}
		}
	}
}

func setTestEnv(t *testing.T) map[string]string {
	t.Helper()
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	setsPath := filepath.Join(tmp, "sets.json")
	writeCatalog(t, catalog, fixtureRecords)
	writeSets(t, setsPath, `{
  "sets": {
    "prompt": {
      "description": "Prompt states",
      "glyphs": {
        "ok": "nf-md-check",
        "dirty": "nf-fa-circle"
      }
    },
    "status": {
      "glyphs": {
        "success": "nf-fa-check"
      }
    }
  }
}
`)
	return map[string]string{catalogPathEnv: catalog, setsPathEnv: setsPath}
}

func TestDefaultSetsPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	want := filepath.Join(tmp, ".clyph", "config", "sets.json")
	if got := defaultSetsPath(); got != want {
		t.Fatalf("defaultSetsPath() = %q want %q", got, want)
	}
}

func TestLoadSetsMergesUserConfig(t *testing.T) {
	env := setTestEnv(t)
	for k, v := range env {
		t.Setenv(k, v)
	}
	sets, err := loadSets()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sets["git"]; !ok {
		t.Fatalf("expected embedded git set")
	}
	if sets["prompt"].Description != "Prompt states" || sets["prompt"].Glyphs["ok"] != "nf-md-check" {
		t.Fatalf("expected user prompt set, got %+v", sets["prompt"])
	}
	if sets["status"].Glyphs["success"] != "nf-fa-check" {
		t.Fatalf("expected user status override, got %+v", sets["status"].Glyphs)
	}
}

func TestSetListShowGlyph(t *testing.T) {
	env := setTestEnv(t)
	code, out, errOut := captureCmd(t, cmdSet, []string{"list"}, env)
	if code != 0 {
		t.Fatalf("set list failed: %s", errOut)
	}
	if !strings.Contains(out, "prompt\t2\tPrompt states") {
		t.Fatalf("set list missing prompt:\n%s", out)
	}
	code, out, errOut = captureCmd(t, cmdSet, []string{"show", "prompt"}, env)
	if code != 0 {
		t.Fatalf("set show failed: %s", errOut)
	}
	if !strings.Contains(out, "dirty\tnf-fa-circle\tf111") || !strings.Contains(out, "ok\tnf-md-check\tf012c") {
		t.Fatalf("set show output wrong:\n%s", out)
	}
	code, out, errOut = captureCmd(t, cmdSet, []string{"glyph", "prompt", "ok"}, env)
	if code != 0 {
		t.Fatalf("set glyph failed: %s", errOut)
	}
	if strings.TrimSpace(out) != "\U000f012c" {
		t.Fatalf("set glyph = %q", out)
	}
}

func TestSetJSON(t *testing.T) {
	env := setTestEnv(t)
	code, out, errOut := captureCmd(t, cmdSet, []string{"show", "prompt", "--json"}, env)
	if code != 0 {
		t.Fatalf("set show json failed: %s", errOut)
	}
	var resp setShowResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Name != "prompt" || len(resp.Glyphs) != 2 || resp.Glyphs[0].Key != "dirty" {
		t.Fatalf("unexpected show json: %+v", resp)
	}
	code, out, errOut = captureCmd(t, cmdSet, []string{"glyph", "prompt", "ok", "--json"}, env)
	if code != 0 {
		t.Fatalf("set glyph json failed: %s", errOut)
	}
	var glyphResp setGlyphResponse
	if err := json.Unmarshal([]byte(out), &glyphResp); err != nil {
		t.Fatal(err)
	}
	if glyphResp.Set != "prompt" || glyphResp.Key != "ok" || glyphResp.Record.Name != "nf-md-check" {
		t.Fatalf("unexpected glyph json: %+v", glyphResp)
	}
}

func TestSetErrors(t *testing.T) {
	env := setTestEnv(t)
	if code, _, errOut := captureCmd(t, cmdSet, []string{}, env); code != 2 || !strings.Contains(errOut, "usage: clyph set") {
		t.Fatalf("expected usage, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := captureCmd(t, cmdSet, []string{"show", "missing"}, env); code != 1 || !strings.Contains(errOut, "set not found") {
		t.Fatalf("expected missing set, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := captureCmd(t, cmdSet, []string{"glyph", "prompt", "missing"}, env); code != 1 || !strings.Contains(errOut, "set key not found") {
		t.Fatalf("expected missing key, code=%d err=%q", code, errOut)
	}
	if code, _, errOut := captureCmd(t, cmdSet, []string{"wat"}, env); code != 2 || !strings.Contains(errOut, "unknown set command") {
		t.Fatalf("expected unknown set command, code=%d err=%q", code, errOut)
	}
}

func TestRunSetHelp(t *testing.T) {
	code, out, errOut := runAndCapture(t, []string{"set", "--help"}, nil)
	if code != 0 || errOut != "" || !strings.Contains(out, "usage: clyph set list") {
		t.Fatalf("expected set help, code=%d out=%q err=%q", code, out, errOut)
	}
}
