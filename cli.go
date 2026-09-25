//go:build !dll && !sim

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	log.SetFlags(0)
	game := flag.String("game", "", "game root directory containing config.toml (required)")
	out := flag.String("out", "", "output mod_pv_db.txt path (required)")
	date := flag.String("date", time.Now().Format("20060102"), "date written into all pv entries (YYYYMMDD)")
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	if *game == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "both -game and -out are required")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := loadConfig(filepath.Join(*game, "config.toml"))
	if err != nil {
		log.Fatalf("reading config: %v", err)
	}
	modsRoot := filepath.Join(*game, cfg.Mods)

	// CLI mode passes an empty version, so run never skips.
	s, err := run(modsRoot, cfg.Priority, *out, *date, "", *verbose)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if s.skipped {
		fmt.Printf("up to date, skipped -> %s\n", *out)
		return
	}
	fmt.Printf("merged %d pv from %d mods, patched %d, rendered %d -> %s\n",
		s.merged, s.sources, s.patched, s.rendered, *out)
}
