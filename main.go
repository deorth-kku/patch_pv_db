// Package main implements patch_pv_db: it merges the rom/mod_pv_db.txt
// files of all mods listed in the MM+ config.toml priority list into a
// single file, so the result can override every other mod's mod_pv_db.
//
// It builds two ways:
//
//	go build                                  # CLI: patch_pv_db.exe
//	go build -buildmode=c-shared -tags dll    # DLL: patch_pv_db.dll (DML PreInit)
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"patch_pv_db/pvdb"
)

type gameConfig struct {
	Enabled  bool     `toml:"enabled"`
	Mods     string   `toml:"mods"`
	Priority []string `toml:"priority"`
}

// summary holds the counts of one run.
type summary struct {
	sources  int
	patches  int
	merged   int
	patched  int
	rendered int
	skipped  bool
}

// run performs the full merge of the game's mods and writes the result to
// out. The mod whose tree contains out is not scanned as a source.
//
// When version is non-empty and the existing out file carries a header
// generated from the same version and the same source files (same paths,
// modification times and order), regeneration is skipped.
func run(game, out, date, version string, verbose bool) (summary, error) {
	var s summary

	cfg, err := loadConfig(filepath.Join(game, "config.toml"))
	if err != nil {
		return s, fmt.Errorf("reading config: %w", err)
	}
	modsRoot := filepath.Join(game, cfg.Mods)
	excluded := excludedMod(out, modsRoot)

	srcFiles, patchFiles := collect(modsRoot, cfg.Priority, excluded, verbose)
	s.sources, s.patches = len(srcFiles), len(patchFiles)
	if verbose {
		log.Printf("scanned %d mod_pv_db files, %d patch_pv_db files", s.sources, s.patches)
	}

	stamps := stampsOf(srcFiles, patchFiles)
	if upToDate(out, version, stamps) {
		s.skipped = true
		return s, nil
	}

	sources, patches := parseSources(srcFiles, patchFiles, verbose)

	db, sourceCount := pvdb.Merge(sources)
	s.merged = len(sourceCount)
	if verbose {
		srcOf := map[string][]string{}
		for _, src := range sources {
			for id := range src.PVs {
				srcOf[id] = append(srcOf[id], src.Name)
			}
		}
		ids := make([]string, 0, len(srcOf))
		for id := range srcOf {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			log.Printf("%s <- %s", id, strings.Join(srcOf[id], ", "))
		}
	}

	patched := pvdb.ApplyPatches(db, patches)
	s.patched = len(patched)

	data, rendered := pvdb.Render(db, sourceCount, patched, date)
	s.rendered = rendered

	var outData bytes.Buffer
	outData.Write(pvdb.BuildHeader(version, stamps))
	outData.Write(data)

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return s, fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(out, outData.Bytes(), 0o644); err != nil {
		return s, fmt.Errorf("writing output: %w", err)
	}
	if verbose {
		for id := range patched {
			log.Printf("patched: %s", id)
		}
	}
	return s, nil
}

// loadConfig reads the game config.toml.
func loadConfig(path string) (gameConfig, error) {
	var cfg gameConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Mods == "" {
		cfg.Mods = "mods"
	}
	return cfg, nil
}

// excludedMod returns the mod folder name whose tree contains out,
// or "" when out is not under the mods root.
func excludedMod(out, modsRoot string) string {
	outAbs, err := filepath.Abs(out)
	if err != nil {
		return ""
	}
	rootAbs, err := filepath.Abs(modsRoot)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(rootAbs, outAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return ""
	}
	parts := strings.SplitN(rel, string(os.PathSeparator), 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// sourceFile is one mod_pv_db.txt or patch_pv_db.txt found by collect.
type sourceFile struct {
	name  string
	path  string
	mtime string
}

// collect lists the mod_pv_db.txt and patch_pv_db.txt files of every mod
// in the priority list (deduplicated, first occurrence wins), skipping the
// excluded mod, and records each file's modification time.
func collect(modsRoot string, priority []string, excluded string, verbose bool) (sources, patches []sourceFile) {
	seen := pvdb.StringSet{}
	for _, name := range priority {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if name == excluded {
			if verbose {
				log.Printf("skip mod %s (contains output file)", name)
			}
			continue
		}
		dir := filepath.Join(modsRoot, name)

		if f, ok := scanFile(dir, "mod_pv_db.txt"); ok {
			f.name = name
			sources = append(sources, f)
		} else if verbose {
			log.Printf("skip mod %s: no rom/mod_pv_db.txt", name)
		}

		if f, ok := scanFile(dir, "patch_pv_db.txt"); ok {
			f.name = name
			patches = append(patches, f)
		}
	}
	return sources, patches
}

// scanFile stats dir/rom/file and returns it with its modification time.
func scanFile(dir, file string) (sourceFile, bool) {
	path := filepath.Join(dir, "rom", file)
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return sourceFile{}, false
	}
	return sourceFile{path: path, mtime: fi.ModTime().Format(time.RFC3339Nano)}, true
}

// parseSources parses the scanned files, skipping any file that cannot be
// parsed.
func parseSources(srcFiles, patchFiles []sourceFile, verbose bool) (sources, patches []pvdb.Source) {
	for _, f := range srcFiles {
		src, err := pvdb.ParseFile(f.path)
		if err != nil {
			if verbose {
				log.Printf("skip mod %s: %v", f.name, err)
			}
			continue
		}
		src.Name = f.name
		sources = append(sources, src)
		if verbose {
			log.Printf("mod %-40s %d pv", f.name, len(src.PVs))
		}
	}
	for _, f := range patchFiles {
		psrc, err := pvdb.ParseFile(f.path)
		if err != nil {
			continue
		}
		psrc.Name = f.name
		patches = append(patches, psrc)
		if verbose {
			log.Printf("patch %-39s %d pv", f.name, len(psrc.PVs))
		}
	}
	return sources, patches
}

// stampsOf records the scanned files (sources first, then patches) in
// input order for the output header and the up-to-date check.
func stampsOf(srcFiles, patchFiles []sourceFile) []pvdb.SourceStamp {
	stamps := make([]pvdb.SourceStamp, 0, len(srcFiles)+len(patchFiles))
	for _, f := range srcFiles {
		stamps = append(stamps, pvdb.SourceStamp{Path: f.path, Mtime: f.mtime})
	}
	for _, f := range patchFiles {
		stamps = append(stamps, pvdb.SourceStamp{Path: f.path, Mtime: f.mtime})
	}
	return stamps
}

// upToDate reports whether the existing file at out was generated from the
// same version and the same source files (same paths, modification times
// and order), in which case regeneration can be skipped. When version is
// empty (CLI mode) it always reports false.
func upToDate(out, version string, stamps []pvdb.SourceStamp) bool {
	if version == "" {
		return false
	}
	f, err := os.Open(out)
	if err != nil {
		return false
	}
	defer f.Close()
	v, files, ok := pvdb.ParseHeaderStream(f)
	if !ok || v != version || len(files) != len(stamps) {
		return false
	}
	for i := range stamps {
		if files[i].Path != stamps[i].Path || files[i].Mtime != stamps[i].Mtime {
			return false
		}
	}
	return true
}
