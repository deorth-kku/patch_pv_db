package main

import (
	"os"
	"path/filepath"
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

	data := pvdb.BuildHeader("1", stamps)
	if err := os.WriteFile(out, data, 0o644); err != nil {
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
