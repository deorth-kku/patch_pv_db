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
func ApplyPatches(db map[string]*PV, patches []Source) StringSet {
	patched := StringSet{}
	patchedBase := map[string]StringSet{}
	patchedSong0 := map[string]StringSet{}
	patchedSongs := map[string]map[string]StringSet{}

	for _, src := range patches {
		for pvID, fields := range src.PVs {
			state, ok := db[pvID]
			if !ok {
				continue
			}
			if patchedBase[pvID] == nil {
				patchedBase[pvID] = StringSet{}
			}
			if patchedSong0[pvID] == nil {
				patchedSong0[pvID] = StringSet{}
			}
			if patchedSongs[pvID] == nil {
				patchedSongs[pvID] = map[string]StringSet{}
			}

			regular, songs := splitFields(fields)
			touched := false
			for field, value := range regular {
				if _, ok := patchedBase[pvID][field]; !ok {
					state.Base[field] = value
					patchedBase[pvID][field] = struct{}{}
					touched = true
				}
			}
			for _, entry := range songs {
				if entry.idx == 0 {
					for rest, value := range entry.fields {
						if rest == "name" {
							value = state.song0NameValue(value)
						}
						if _, ok := patchedSong0[pvID][rest]; !ok {
							state.setSong0(rest, value)
							patchedSong0[pvID][rest] = struct{}{}
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
									patchedSongs[pvID][sfname] = StringSet{}
								}
								for k, v := range entry.fields {
									if _, ok := patchedSongs[pvID][sfname][k]; !ok {
										existing[k] = v
										patchedSongs[pvID][sfname][k] = struct{}{}
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
				patched[pvID] = struct{}{}
			}
		}
	}
	return patched
}
