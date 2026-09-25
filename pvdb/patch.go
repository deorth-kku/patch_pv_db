package pvdb

// ApplyPatches applies patch sources in priority order on top of the
// first-round merged db.
//
// Only pv ids that already exist in db are patched. For regular keys the
// first patch (in priority order) that sets a key wins; patch values
// override first-round values. another_song keys follow the same
// accumulation rules as the first round: index-0 fields are merged into
// the shared entry (first patch wins per field), entries with index >= 1
// are appended after the existing entries. The owner song_name rule for
// the entry-0 name (see PV.song0NameValue) applies to patches as well.
//
// The returned set contains the pv ids that were touched by at least one
// patch key.
func ApplyPatches(db map[string]*PV, patches []Source) map[string]bool {
	patched := map[string]bool{}
	patchedBase := map[string]map[string]bool{}
	patchedSong0 := map[string]map[string]bool{}
	patchedSongs := map[string]map[string]map[string]bool{}

	for _, src := range patches {
		for pvID, fields := range src.PVs {
			state, ok := db[pvID]
			if !ok {
				continue
			}
			if patchedBase[pvID] == nil {
				patchedBase[pvID] = map[string]bool{}
			}
			if patchedSong0[pvID] == nil {
				patchedSong0[pvID] = map[string]bool{}
			}
			if patchedSongs[pvID] == nil {
				patchedSongs[pvID] = map[string]map[string]bool{}
			}

			regular, songs := splitFields(fields)
			touched := false
			for field, value := range regular {
				if !patchedBase[pvID][field] {
					state.Base[field] = value
					patchedBase[pvID][field] = true
					touched = true
				}
			}
			for _, entry := range songs {
				if entry.idx == 0 {
					for rest, value := range entry.fields {
						if rest == "name" {
							value = state.song0NameValue(value)
						}
						if !patchedSong0[pvID][rest] {
							state.setSong0(rest, value)
							patchedSong0[pvID][rest] = true
							touched = true
						}
					}
				} else {
					sfname, hasSfname := entry.fields["song_file_name"]
					if hasSfname {
						found := false
						for _, existing := range state.Songs {
							if existing["song_file_name"] == sfname {
								if patchedSongs[pvID][sfname] == nil {
									patchedSongs[pvID][sfname] = map[string]bool{}
								}
								for k, v := range entry.fields {
									if !patchedSongs[pvID][sfname][k] {
										existing[k] = v
										patchedSongs[pvID][sfname][k] = true
										touched = true
									}
								}
								found = true
								break
							}
						}
						if !found {
							state.Songs = append(state.Songs, entry.fields)
							touched = true
						}
					} else {
						state.Songs = append(state.Songs, entry.fields)
						touched = true
					}
				}
			}
			if touched {
				patched[pvID] = true
			}
		}
	}
	return patched
}
