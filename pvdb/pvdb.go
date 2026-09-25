// Package pvdb implements parsing, merging, patching and rendering of
// MM+ mod_pv_db.txt files.
package pvdb

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// StringSet is a set of strings.
type StringSet map[string]struct{}

// Source is one parsed mod_pv_db.txt (or patch_pv_db.txt) file.
type Source struct {
	// Name is the mod folder name this source was loaded from.
	Name string
	// Path is the full path of the file.
	Path string
	// PVs maps pv id (e.g. "pv_1133") to field key (e.g. "bpm") to value.
	PVs map[string]map[string]string
}

// PV holds the merged state of one pv id.
type PV struct {
	// Base holds all field keys that are not another_song entries.
	Base map[string]string
	// Song0 is the shared index-0 another_song entry, nil if absent.
	Song0 map[string]string
	// Songs is the ordered list of appended another_song entries
	// (source indices >= 1), in priority order.
	Songs []map[string]string

	// ownerHasSong0Name reports whether the owner source (the
	// highest-priority source containing this pv) defined
	// another_song.0.name.
	ownerHasSong0Name bool
	// ownerSongName is the owner source's song_name value.
	ownerSongName string
	// ownerHasSongName reports whether the owner source defined song_name.
	ownerHasSongName bool
}

// song0NameKey is the field key of the shared index-0 entry's name.
const song0NameKey = songPrefix + "0.name"

// song0NameValue returns the value to use for the shared index-0 entry's
// "name" field when a source provides it. When the owner source did not
// define another_song.0.name but did define song_name, the owner's
// song_name is used instead, so that a later source adding entry 0
// inherits the owner's title.
func (p *PV) song0NameValue(value string) string {
	if !p.ownerHasSong0Name && p.ownerHasSongName {
		return p.ownerSongName
	}
	return value
}

const songPrefix = "another_song."

// songLengthKey is the recomputed length field.
const songLengthKey = songPrefix + "length"

// songKey splits a field key like "another_song.1.name" into the entry
// index and the remaining field name. ok is false for non-entry keys
// such as "another_song.length" and for keys outside another_song.
func songKey(field string) (index int, rest string, ok bool) {
	if !strings.HasPrefix(field, songPrefix) {
		return 0, "", false
	}
	num, after, found := strings.Cut(field[len(songPrefix):], ".")
	if !found {
		return 0, "", false
	}
	idx, err := strconv.Atoi(num)
	if err != nil || idx < 0 {
		return 0, "", false
	}
	return idx, after, true
}

// songEntries returns the ordered list of another_song entries:
// the shared index-0 entry first (when present), then the appended ones.
func (p *PV) songEntries() []map[string]string {
	var entries []map[string]string
	if p.Song0 != nil {
		entries = append(entries, p.Song0)
	}
	return append(entries, p.Songs...)
}

// setSong0 sets a field of the shared index-0 entry, creating it if needed.
func (p *PV) setSong0(rest, value string) {
	if p.Song0 == nil {
		p.Song0 = map[string]string{}
	}
	p.Song0[rest] = value
}

// setSong0IfAbsent sets a field of the shared index-0 entry only when the
// field is not present yet, creating the entry if needed.
func (p *PV) setSong0IfAbsent(rest, value string) {
	if p.Song0 == nil {
		p.Song0 = map[string]string{}
	}
	if _, exists := p.Song0[rest]; !exists {
		p.Song0[rest] = value
	}
}

// appendSong adds a song entry (idx >= 1) to Songs. If an existing entry
// has the same song_file_name, the new fields are merged in with existing
// values winning (first-wins). Otherwise the entry is appended.
func (p *PV) appendSong(fields map[string]string) {
	sfname, ok := fields["song_file_name"]
	if ok {
		for _, existing := range p.Songs {
			if existing["song_file_name"] == sfname {
				for k, v := range fields {
					if _, exists := existing[k]; !exists {
						existing[k] = v
					}
				}
				return
			}
		}
	}
	p.Songs = append(p.Songs, fields)
}

// songEntry is one another_song entry (a set of fields sharing the same
// entry index) parsed from a source.
type songEntry struct {
	idx    int
	fields map[string]string
}

// splitFields separates the field map of one pv into regular fields and
// another_song entries grouped by entry index, sorted by index.
func splitFields(fields map[string]string) (regular map[string]string, songs []songEntry) {
	regular = map[string]string{}
	byIdx := map[int]map[string]string{}
	var idxs []int
	for field, value := range fields {
		if idx, rest, ok := songKey(field); ok {
			m, exists := byIdx[idx]
			if !exists {
				m = map[string]string{}
				byIdx[idx] = m
				idxs = append(idxs, idx)
			}
			m[rest] = value
		} else {
			regular[field] = value
		}
	}
	sort.Ints(idxs)
	for _, idx := range idxs {
		songs = append(songs, songEntry{idx: idx, fields: byIdx[idx]})
	}
	return regular, songs
}

// renderKeys returns the final field map of a PV for output: base fields,
// renumbered another_song entries and the recomputed length.
func (p *PV) renderKeys(date string) map[string]string {
	keys := map[string]string{}
	for k, v := range p.Base {
		if k == songLengthKey {
			continue // recomputed below
		}
		keys[k] = v
	}
	entries := p.songEntries()
	for idx, entry := range entries {
		for rest, v := range entry {
			keys[fmt.Sprintf("%s%d.%s", songPrefix, idx, rest)] = v
		}
	}
	if len(entries) > 0 {
		keys[songLengthKey] = strconv.Itoa(len(entries))
	}
	keys["date"] = date
	return keys
}
