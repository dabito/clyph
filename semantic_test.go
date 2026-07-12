package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func semanticEnv(t *testing.T, seed map[string]string) (map[string]string, string, string) {
	t.Helper()
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	seedPath := filepath.Join(tmp, "semantic.json")
	b, err := json.Marshal(semanticSeed{Concepts: seed})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(seedPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return map[string]string{catalogPathEnv: catalog, "CLYPH_SEMANTIC_PATH": seedPath}, catalog, seedPath
}

func TestSemanticSeedCanonical(t *testing.T) {
	env, _, _ := semanticEnv(t, map[string]string{"success": "nf-md-check"})

	// Default: crisp single canonical row, source=seed.
	code, out, errOut := captureCmd(t, cmdSemantic, []string{"success"}, env)
	if code != 0 {
		t.Fatalf("semantic failed: %s", errOut)
	}
	rows := strings.Split(strings.TrimSpace(out), "\n")
	if len(rows) != 1 || !strings.Contains(rows[0], "nf-md-check") || !strings.HasSuffix(rows[0], "\tseed") {
		t.Fatalf("semantic default row = %q", rows)
	}
}

func TestSemanticLabelAliasSources(t *testing.T) {
	env, _, _ := semanticEnv(t, map[string]string{})

	// "check" is nf-md-check's Label -> source=label.
	code, out, _ := captureCmd(t, cmdSemantic, []string{"check"}, env)
	if code != 0 || !strings.Contains(out, "nf-md-check") || !strings.HasSuffix(strings.TrimSpace(out), "\tlabel") {
		t.Fatalf("semantic check (label) = %q", out)
	}

	// "offline-outline" is nf-fa-circle_o's alias only -> source=alias.
	code, out, _ = captureCmd(t, cmdSemantic, []string{"offline-outline"}, env)
	if code != 0 || !strings.Contains(out, "nf-fa-circle_o") || !strings.HasSuffix(strings.TrimSpace(out), "\talias") {
		t.Fatalf("semantic offline-outline (alias) = %q", out)
	}

	// "done" matches nf-fa-check (label+alias). --all should surface it.
	code, out, _ = captureCmd(t, cmdSemantic, []string{"done", "--all"}, env)
	if code != 0 || !strings.Contains(out, "nf-fa-check") {
		t.Fatalf("semantic done --all = %q", out)
	}
}

func TestSemanticFallbackAndMiss(t *testing.T) {
	env, _, _ := semanticEnv(t, map[string]string{"success": "nf-md-check"})

	// Unknown concept that is a substring of names -> search fallback.
	code, out, _ := captureCmd(t, cmdSemantic, []string{"circle", "--all"}, env)
	if code != 0 || !strings.Contains(out, "nf-fa-circle") {
		t.Fatalf("semantic circle --all fallback = %q", out)
	}
	if !strings.HasSuffix(strings.Split(strings.TrimSpace(out), "\n")[0], "\tsearch") {
		t.Fatalf("fallback source should be search, got %q", out)
	}

	// Truly unknown -> default exits 1.
	code, _, errOut := captureCmd(t, cmdSemantic, []string{"zzznotathing"}, env)
	if code == 0 || !strings.Contains(errOut, "no semantic match") {
		t.Fatalf("expected miss failure, code=%d err=%q", code, errOut)
	}
}

func TestSemanticJSON(t *testing.T) {
	env, _, _ := semanticEnv(t, map[string]string{"success": "nf-md-check"})
	code, out, errOut := captureCmd(t, cmdSemantic, []string{"success", "--json"}, env)
	if code != 0 {
		t.Fatalf("semantic --json failed: %s", errOut)
	}
	var resp semanticResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Concept != "success" || resp.Canonical == nil || resp.Canonical.Name != "nf-md-check" {
		t.Fatalf("semantic json = %#v", resp)
	}
	if len(resp.Matches) == 0 || !resp.Matches[0].Canonical || resp.Matches[0].Source != "seed" {
		t.Fatalf("semantic json matches = %#v", resp.Matches)
	}
}

func TestSemanticGuards(t *testing.T) {
	env, _, _ := semanticEnv(t, map[string]string{"success": "nf-md-check"})
	if code, _, _ := captureCmd(t, cmdSemantic, []string{}, env); code == 0 {
		t.Fatal("expected missing-concept failure")
	}
	if code, _, errOut := captureCmd(t, cmdSemantic, []string{"success", "--wat"}, env); code == 0 || !strings.Contains(errOut, "unknown flag") {
		t.Fatalf("expected unknown flag, code=%d err=%q", code, errOut)
	}
	// uppercase input is normalized to lowercase and still resolves.
	if code, out, _ := captureCmd(t, cmdSemantic, []string{"SUCCESS"}, env); code != 0 || !strings.Contains(out, "nf-md-check") {
		t.Fatalf("uppercase concept should normalize: code=%d out=%q", code, out)
	}
}
