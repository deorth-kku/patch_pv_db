//go:build dll

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// modName is this mod's folder name under the game's mods directory.
const modName = "Patch PV DB"

// main is required by the c-shared build mode; the real entry point is
// PreInit, which DivaModLoader calls.
func main() {}

// PreInit is called by DivaModLoader before the game's global variables
// are initialized, i.e. before the game loads any mod_pv_db.txt. It
// regenerates this mod's rom/mod_pv_db.txt from all the other mods.
//
//export PreInit
func PreInit() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	log.SetPrefix("[" + modName + "] ")

	gameRoot, err := findGameRoot()
	if err != nil {
		log.Printf("%v", err)
		return
	}
	gameCfg, err := loadConfig(filepath.Join(gameRoot, "config.toml"))
	if err != nil {
		log.Printf("reading config: %v", err)
		return
	}
	modsRoot := filepath.Join(gameRoot, gameCfg.Mods)
	modDir := filepath.Join(modsRoot, modName)
	out := filepath.Join(modDir, "rom", "mod_pv_db.txt")

	modCfg := loadModConfig(filepath.Join(modDir, "config.toml"))
	s, err := run(modsRoot, gameCfg.Priority, out, time.Now().Format("20060102"), modCfg.Version, modCfg.Verbose)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	if s.skipped {
		log.Printf("up to date, skipped -> %s", out)
		return
	}
	log.Printf("merged %d pv from %d mods, patched %d, rendered %d -> %s",
		s.merged, s.sources, s.patched, s.rendered, out)
}

// findGameRoot walks up from the game executable's directory and returns
// the first directory that contains a config.toml file.
func findGameRoot() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	for {
		if fi, err := os.Stat(filepath.Join(dir, "config.toml")); err == nil && !fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("game config.toml not found above %s", exe)
		}
		dir = parent
	}
}

type modConfig struct {
	Verbose bool   `toml:"verbose"`
	Version string `toml:"version"`
}

// loadModConfig reads the config flags from this mod's config.toml.
// It returns zero value when the file cannot be read.
func loadModConfig(path string) modConfig {
	var cfg modConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return modConfig{}
	}
	return cfg
}
