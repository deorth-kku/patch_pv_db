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
	"bufio"
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
	fields   int
	skipped  bool
}

// run performs the full merge of the mods under modsRoot (scanned in
// priority order) and writes the result to out. The mod whose tree
// contains out is not scanned as a source.
//
// When version is non-empty and the existing out file carries a header
// generated from the same version and the same source files (same paths,
// modification times and order), regeneration is skipped.
func run(modsRoot string, priority []string, out, date, version string, verbose bool) (summary, error) {
	var s summary

	excluded := excludedMod(out, modsRoot)

	srcFiles, patchFiles, fieldFiles := collect(modsRoot, priority, excluded, verbose)
	s.sources, s.patches = len(srcFiles), len(patchFiles)
	if verbose {
		log.Printf("scanned %d mod_pv_db files, %d patch_pv_db files, %d mod_pv_field files",
			s.sources, s.patches, len(fieldFiles))
	}

	stamps := stampsOf(srcFiles, patchFiles, fieldFiles)
	if upToDate(out, version, stamps) {
		s.skipped = true
		return s, nil
	}

	sources, patches, fieldSources := parseSources(srcFiles, patchFiles, fieldFiles, verbose)

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

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return s, fmt.Errorf("creating output directory: %w", err)
	}
	// Write to a temp file in the same directory, then atomically move it
	// over out. If anything fails, the temp file is removed and the
	// existing out (old version) is left untouched.
	tmp, err := os.CreateTemp(filepath.Dir(out), filepath.Base(out)+".tmp-*")
	if err != nil {
		return s, fmt.Errorf("creating temp output: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			os.Remove(tmpName)
		}
	}()

	w := bufio.NewWriter(tmp)
	if err := pvdb.BuildHeader(w, version, stamps); err != nil {
		tmp.Close()
		return s, fmt.Errorf("writing output: %w", err)
	}
	if s.rendered, err = pvdb.Render(w, db, sourceCount, patched, date); err != nil {
		tmp.Close()
		return s, fmt.Errorf("writing output: %w", err)
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return s, fmt.Errorf("writing output: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return s, fmt.Errorf("writing output: %w", err)
	}
	if err := os.Rename(tmpName, out); err != nil {
		return s, fmt.Errorf("writing output: %w", err)
	}
	tmpName = ""

	// Write the companion mod_pv_field.txt alongside out: for every pv in
	// the rendered db, the field entries copied (first-wins per key, in
	// priority order) from the source mods' mod_pv_field.txt files. The
	// same temp-file/rename cleanup as above applies via tmpName.
	fieldOut := filepath.Join(filepath.Dir(out), "mod_pv_field.txt")
	fieldTmp, err := os.CreateTemp(filepath.Dir(out), filepath.Base(fieldOut)+".tmp-*")
	if err != nil {
		return s, fmt.Errorf("creating temp field output: %w", err)
	}
	tmpName = fieldTmp.Name()
	wf := bufio.NewWriter(fieldTmp)
	pvSet := pvdb.IncludedPVs(sourceCount, patched)
	fieldDB := pvdb.MergeFields(fieldSources)
	if s.fields, err = pvdb.RenderFields(wf, fieldDB, pvSet); err != nil {
		fieldTmp.Close()
		return s, fmt.Errorf("writing field output: %w", err)
	}
	if err := wf.Flush(); err != nil {
		fieldTmp.Close()
		return s, fmt.Errorf("writing field output: %w", err)
	}
	if err := fieldTmp.Close(); err != nil {
		return s, fmt.Errorf("writing field output: %w", err)
	}
	if err := os.Rename(tmpName, fieldOut); err != nil {
		return s, fmt.Errorf("writing field output: %w", err)
	}
	tmpName = ""

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

// collect lists the mod_pv_db.txt, patch_pv_db.txt and mod_pv_field.txt
// files of every mod in the priority list (deduplicated, first occurrence
// wins), skipping the excluded mod, and records each file's modification
// time.
func collect(modsRoot string, priority []string, excluded string, verbose bool) (sources, patches, fieldFiles []sourceFile) {
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

		if f, ok := scanFile(dir, "mod_pv_field.txt"); ok {
			f.name = name
			fieldFiles = append(fieldFiles, f)
		}
	}
	return sources, patches, fieldFiles
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
func parseSources(srcFiles, patchFiles, fieldFiles []sourceFile, verbose bool) (sources, patches, fieldSources []pvdb.Source) {
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
	for _, f := range fieldFiles {
		fsrc, err := pvdb.ParseFile(f.path)
		if err != nil {
			if verbose {
				log.Printf("skip mod %s field: %v", f.name, err)
			}
			continue
		}
		fsrc.Name = f.name
		fieldSources = append(fieldSources, fsrc)
		if verbose {
			log.Printf("field %-39s %d pv", f.name, len(fsrc.PVs))
		}
	}
	return sources, patches, fieldSources
}

// stampsOf records the scanned files (sources, then patches, then field
// files) in input order for the output header and the up-to-date check.
func stampsOf(srcFiles, patchFiles, fieldFiles []sourceFile) []pvdb.SourceStamp {
	stamps := make([]pvdb.SourceStamp, 0, len(srcFiles)+len(patchFiles)+len(fieldFiles))
	for _, f := range srcFiles {
		stamps = append(stamps, pvdb.SourceStamp{Path: f.path, Mtime: f.mtime})
	}
	for _, f := range patchFiles {
		stamps = append(stamps, pvdb.SourceStamp{Path: f.path, Mtime: f.mtime})
	}
	for _, f := range fieldFiles {
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
