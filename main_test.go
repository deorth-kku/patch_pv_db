package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"patch_pv_db/pvdb"
)

func TestUpToDate(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "mod_pv_db.txt")
	stamps := []pvdb.SourceStamp{
		{Path: filepath.Join(dir, "a", "mod_pv_db.txt"), Mtime: "2025-01-01T00:00:00Z"},
		{Path: filepath.Join(dir, "b", "patch_pv_db.txt"), Mtime: "2025-01-02T00:00:00Z"},
	}

	// no existing output file
	if upToDate(out, "1", stamps) {
		t.Error("upToDate must be false when the output file does not exist")
	}

	var buf bytes.Buffer
	if err := pvdb.BuildHeader(&buf, "1", stamps); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// empty version (CLI mode): always false
	if upToDate(out, "", stamps) {
		t.Error("upToDate must be false for an empty version (CLI mode)")
	}

	// matching version and stamps
	if !upToDate(out, "1", stamps) {
		t.Error("upToDate must be true for matching version and stamps")
	}

	// version changed
	if upToDate(out, "2", stamps) {
		t.Error("upToDate must be false when the version changed")
	}

	// order changed
	reversed := []pvdb.SourceStamp{stamps[1], stamps[0]}
	if upToDate(out, "1", reversed) {
		t.Error("upToDate must be false when the source order changed")
	}

	// mtime changed
	changed := []pvdb.SourceStamp{stamps[0], {Path: stamps[1].Path, Mtime: "2025-01-03T00:00:00Z"}}
	if upToDate(out, "1", changed) {
		t.Error("upToDate must be false when a source mtime changed")
	}

	// file count changed
	if upToDate(out, "1", stamps[:1]) {
		t.Error("upToDate must be false when the source count changed")
	}
}

func TestRunWritesFieldFile(t *testing.T) {
	modsRoot := t.TempDir()

	writeMod := func(name, db, field string) {
		rom := filepath.Join(modsRoot, name, "rom")
		if err := os.MkdirAll(rom, 0o755); err != nil {
			t.Fatal(err)
		}
		if db != "" {
			if err := os.WriteFile(filepath.Join(rom, "mod_pv_db.txt"), []byte(db), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if field != "" {
			if err := os.WriteFile(filepath.Join(rom, "mod_pv_field.txt"), []byte(field), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// pv_100 and pv_200 are in both mods (rendered); pv_300 is only in B
	// (single-source, dropped from the db and therefore from the field file).
	writeMod("A",
		"pv_100.bpm=100\r\npv_200.bpm=200\r\n",
		"pv_100.field.01.stage=STGA\r\npv_100.field.length=1\r\n")
	writeMod("B",
		"pv_100.bpm=999\r\npv_200.bpm=200\r\npv_300.bpm=300\r\n",
		"pv_100.field.01.light=LTA\r\npv_200.field.01.stage=STGB\r\n")

	outDir := filepath.Join(modsRoot, "OutMod", "rom")
	out := filepath.Join(outDir, "mod_pv_db.txt")

	s, err := run(modsRoot, []string{"A", "B"}, out, "20260101", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if s.skipped {
		t.Fatal("run must not skip in CLI (empty version) mode")
	}
	if s.rendered != 2 {
		t.Fatalf("rendered %d pv, want 2", s.rendered)
	}
	if s.fields != 2 {
		t.Fatalf("fields %d, want 2", s.fields)
	}

	// DB output: pv_100 and pv_200 only.
	dbData, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dbData), "pv_100.bpm=100") {
		t.Errorf("db missing pv_100.bpm=100:\n%s", dbData)
	}
	if !strings.Contains(string(dbData), "pv_200.bpm=200") {
		t.Errorf("db missing pv_200.bpm=200:\n%s", dbData)
	}
	if strings.Contains(string(dbData), "pv_300") {
		t.Errorf("db must not contain single-source pv_300:\n%s", dbData)
	}

	// Field output: exactly the field entries for the rendered pv set,
	// merged first-wins per key in priority order (A before B).
	fieldOut := filepath.Join(outDir, "mod_pv_field.txt")
	fieldData, err := os.ReadFile(fieldOut)
	if err != nil {
		t.Fatalf("field file not written: %v", err)
	}
	wantField := "pv_100.field.01.light=LTA\r\n" +
		"pv_100.field.01.stage=STGA\r\n" +
		"pv_100.field.length=1\r\n" +
		"\r\n" +
		"pv_200.field.01.stage=STGB\r\n"
	if string(fieldData) != wantField {
		t.Errorf("field file = %q, want %q", string(fieldData), wantField)
	}
}
