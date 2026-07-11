package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureCmd runs fn against args with the given catalog env and returns
// (code, stdout, stderr). It does NOT go through run() dispatch, so commands
// not yet wired into the router can still be tested.
func captureCmd(t *testing.T, fn func([]string) int, args []string, env map[string]string) (int, string, string) {
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
			if e := os.Setenv(k, v); e != nil {
				t.Fatal(e)
			}
		}
		defer func() {
			for k, v := range oldEnv {
				if wasSet[k] {
					_ = os.Setenv(k, v)
				} else {
					_ = os.Unsetenv(k)
				}
			}
		}()
	}
	codeCh := make(chan int, 1)
	go func() {
		codeCh <- fn(args)
		_ = outW.Close()
		_ = errW.Close()
	}()
	var outBuf, errBuf bytes.Buffer
	_, _ = outBuf.ReadFrom(outR)
	_, _ = errBuf.ReadFrom(errR)
	return <-codeCh, outBuf.String(), errBuf.String()
}

func familiesEnv(t *testing.T) (map[string]string, string) {
	t.Helper()
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	return map[string]string{catalogPathEnv: catalog}, catalog
}

func TestFamilyCountsAndList(t *testing.T) {
	env, _ := familiesEnv(t)

	// fixture: nf-fa-* x3, nf-md-* x2 -> fa=3, md=2; sorted count desc.
	code, out, errOut := captureCmd(t, cmdFamilies, nil, env)
	if code != 0 {
		t.Fatalf("families failed: %s", errOut)
	}
	got := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{"fa\t3", "md\t2"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("families plain = %q want %q", got, want)
	}

	// --limit 1 keeps only the top family.
	code, out, errOut = captureCmd(t, cmdFamilies, []string{"--limit", "1"}, env)
	if code != 0 {
		t.Fatalf("families --limit failed: %s", errOut)
	}
	if strings.TrimSpace(out) != "fa\t3" {
		t.Fatalf("families --limit 1 = %q want %q", out, "fa\t3")
	}

	// --json round-trips into the expected slice.
	code, out, errOut = captureCmd(t, cmdFamilies, []string{"--json"}, env)
	if code != 0 {
		t.Fatalf("families --json failed: %s", errOut)
	}
	var fams []familyStat
	if err := json.Unmarshal([]byte(out), &fams); err != nil {
		t.Fatal(err)
	}
	if len(fams) != 2 || fams[0].Family != "fa" || fams[0].Count != 3 || fams[1].Count != 2 {
		t.Fatalf("families json = %#v", fams)
	}

	// negative limit rejected.
	code, _, errOut = captureCmd(t, cmdFamilies, []string{"--limit", "-1"}, env)
	if code == 0 || !strings.Contains(errOut, "must be non-negative") {
		t.Fatalf("expected negative-limit rejection, code=%d err=%q", code, errOut)
	}

	// unknown flag rejected.
	code, _, errOut = captureCmd(t, cmdFamilies, []string{"--wat"}, env)
	if code == 0 || !strings.Contains(errOut, "unknown flag") {
		t.Fatalf("expected unknown flag rejection, code=%d err=%q", code, errOut)
	}
}

func TestStats(t *testing.T) {
	env, _ := familiesEnv(t)

	// fixture: total 5, families 2, all labeled (5), aliased 4 (nf-md-check has none).
	code, out, errOut := captureCmd(t, cmdStats, nil, env)
	if code != 0 {
		t.Fatalf("stats failed: %s", errOut)
	}
	wantLines := []string{"total\t5", "families\t2", "labeled\t5", "aliased\t4"}
	got := strings.Split(strings.TrimSpace(out), "\n")
	if len(got) != len(wantLines) {
		t.Fatalf("stats plain = %q want %q", got, wantLines)
	}
	for i := range wantLines {
		if got[i] != wantLines[i] {
			t.Fatalf("stats[%d] = %q want %q", i, got[i], wantLines[i])
		}
	}

	// --json round-trip.
	code, out, errOut = captureCmd(t, cmdStats, []string{"--json"}, env)
	if code != 0 {
		t.Fatalf("stats --json failed: %s", errOut)
	}
	var s catalogStats
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatal(err)
	}
	if s != (catalogStats{Total: 5, Families: 2, Labeled: 5, Aliased: 4}) {
		t.Fatalf("stats json = %#v", s)
	}

	// missing catalog -> exit 1.
	code, _, errOut = captureCmd(t, cmdStats, nil, map[string]string{catalogPathEnv: filepath.Join(t.TempDir(), "missing.json")})
	if code == 0 {
		t.Fatalf("expected missing-catalog failure, got code 0, err=%q", errOut)
	}
}
