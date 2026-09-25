package pvdb

// Merge merges sources in priority order: earlier sources win every
// key conflict (the original date fields are ignored).
//
// another_song fields are accumulated instead of overridden: the shared
// index-0 entry is merged field by field (earlier source wins), and
// entries with index >= 1 are appended in priority order and renumbered
// on render.
//
// Exception: when the owner source (the highest-priority source
// containing a pv) does not define another_song.0.name, a later source
// that provides it gets the owner's song_name as the entry-0 name
// instead (see PV.song0NameValue).
//
// The second return value counts, per pv id, how many sources contain it.
func Merge(sources []Source) (map[string]*PV, map[string]int) {
	db := map[string]*PV{}
	sourceCount := map[string]int{}
	for _, src := range sources {
		for pvID, fields := range src.PVs {
			state, ok := db[pvID]
			if !ok {
				state = &PV{Base: map[string]string{}}
				db[pvID] = state
				if _, exists := fields[song0NameKey]; exists {
					state.ownerHasSong0Name = true
				}
				if value, exists := fields["song_name"]; exists {
					state.ownerSongName = value
					state.ownerHasSongName = true
				}
			}
			sourceCount[pvID]++

			regular, songs := splitFields(fields)
			for field, value := range regular {
				if _, exists := state.Base[field]; !exists {
					state.Base[field] = value
				}
			}
			for _, entry := range songs {
				if entry.idx == 0 {
					for rest, value := range entry.fields {
						if rest == "name" {
							value = state.song0NameValue(value)
						}
						state.setSong0IfAbsent(rest, value)
					}
				} else {
					state.appendSong(entry.fields)
				}
			}
		}
	}
	return db, sourceCount
}
