package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCatalogCachePath(t *testing.T) {
	got := catalogCachePath(filepath.Join("tmp", "catalog.json"))
	want := filepath.Join("tmp", "catalog.cache.gob")
	if got != want {
		t.Fatalf("catalogCachePath() = %q want %q", got, want)
	}
	got = catalogCachePath(filepath.Join("tmp", "custom"))
	want = filepath.Join("tmp", "custom.cache.gob")
	if got != want {
		t.Fatalf("catalogCachePath(no ext) = %q want %q", got, want)
	}
}

func TestLoadRecordsRefreshesGobCache(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	cache := catalogCachePath(catalog)
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("expected no cache before load, stat err=%v", err)
	}
	recs, err := loadRecords(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != len(fixtureRecords) {
		t.Fatalf("record count = %d want %d", len(recs), len(fixtureRecords))
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("expected cache after load: %v", err)
	}
	cached, err := loadRecordsCache(cache)
	if err != nil {
		t.Fatalf("cache decode failed: %v", err)
	}
	if len(cached) != len(fixtureRecords) {
		t.Fatalf("cache record count = %d want %d", len(cached), len(fixtureRecords))
	}
}

func TestLoadRecordsFallsBackFromCorruptGobCache(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	writeCatalog(t, catalog, fixtureRecords)
	cache := catalogCachePath(catalog)
	if err := os.WriteFile(cache, []byte("not gob"), 0o644); err != nil {
		t.Fatal(err)
	}
	recs, err := loadRecords(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != len(fixtureRecords) {
		t.Fatalf("record count = %d want %d", len(recs), len(fixtureRecords))
	}
	cached, err := loadRecordsCache(cache)
	if err != nil {
		t.Fatalf("expected corrupt cache to be refreshed: %v", err)
	}
	if len(cached) != len(fixtureRecords) {
		t.Fatalf("refreshed cache record count = %d want %d", len(cached), len(fixtureRecords))
	}
}

func TestSaveRecordsWritesJsonAndGobCache(t *testing.T) {
	tmp := t.TempDir()
	catalog := filepath.Join(tmp, "catalog.json")
	if err := saveRecords(fixtureRecords, catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(catalog); err != nil {
		t.Fatalf("expected json catalog: %v", err)
	}
	cache := catalogCachePath(catalog)
	if _, err := os.Stat(cache); err != nil {
		t.Fatalf("expected cache catalog: %v", err)
	}
	recs, err := loadRecords(cache)
	if err == nil || recs != nil {
		t.Fatalf("cache path should not be treated as canonical catalog; recs=%v err=%v", recs, err)
	}
	cached, err := loadRecordsCache(cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(cached) != len(fixtureRecords) {
		t.Fatalf("cached record count = %d want %d", len(cached), len(fixtureRecords))
	}
}
