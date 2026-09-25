package pvdb

import "testing"

func TestApplyPatches(t *testing.T) {
	db, _ := Merge([]Source{
		src(map[string]map[string]string{"pv_1": {"bpm": "100", "song_name": "X"}}),
	})

	p1 := src(map[string]map[string]string{
		"pv_1": {"bpm": "200"},
		"pv_9": {"bpm": "1"}, // pv_9 not in db: ignored
	})
	p2 := src(map[string]map[string]string{
		"pv_1": {"bpm": "300"}, // loses to p1
	})

	patched := ApplyPatches(db, []Source{p1, p2})

	if got := db["pv_1"].Base["bpm"]; got != "200" {
		t.Errorf("pv_1.bpm = %q, want %q (first patch wins)", got, "200")
	}
	if got := db["pv_1"].Base["song_name"]; got != "X" {
		t.Errorf("pv_1.song_name = %q, want unchanged %q", got, "X")
	}
	if _, ok := db["pv_9"]; ok {
		t.Error("pv_9 must not be created by patch")
	}
	if _, ok := patched["pv_1"]; !ok {
		t.Error("pv_1 must be marked patched")
	}
	if len(patched) != 1 {
		t.Errorf("patched = %v, want only pv_1", patched)
	}
}

func TestApplyPatchesAnotherSong(t *testing.T) {
	db, _ := Merge([]Source{
		src(map[string]map[string]string{"pv_1": {"bpm": "100"}}),
	})

	p := src(map[string]map[string]string{
		"pv_1": {
			"another_song.length": "2",
			"another_song.0.name": "P0",
			"another_song.1.name": "P1",
		},
	})

	patched := ApplyPatches(db, []Source{p})
	keys := db["pv_1"].renderKeys("20260101")

	want := map[string]string{
		"bpm":                 "100",
		"another_song.length": "2",
		"another_song.0.name": "P0",
		"another_song.1.name": "P1",
	}
	for k, v := range want {
		if keys[k] != v {
			t.Errorf("%s = %q, want %q", k, keys[k], v)
		}
	}
	if _, ok := patched["pv_1"]; !ok {
		t.Error("pv_1 must be marked patched")
	}
}

func TestApplyPatchesSong0NameFromOwnerSongName(t *testing.T) {
	db, _ := Merge([]Source{
		src(map[string]map[string]string{"pv_1": {"song_name": "Owner Title"}}),
	})

	p := src(map[string]map[string]string{
		"pv_1": {
			"another_song.0.name":           "P0",
			"another_song.0.song_file_name": "rom/sound/song/pv_1.ogg",
		},
	})

	ApplyPatches(db, []Source{p})
	keys := db["pv_1"].renderKeys("20260101")

	if got := keys["another_song.0.name"]; got != "Owner Title" {
		t.Errorf("another_song.0.name = %q, want %q (owner song_name)", got, "Owner Title")
	}
	if got := keys["another_song.0.song_file_name"]; got != "rom/sound/song/pv_1.ogg" {
		t.Errorf("another_song.0.song_file_name = %q, want unchanged", got)
	}
}

func TestApplyPatchesAnotherSongAppends(t *testing.T) {
	db, _ := Merge([]Source{
		src(map[string]map[string]string{
			"pv_1": {
				"another_song.length": "2",
				"another_song.0.name": "A0",
				"another_song.1.name": "A1",
			},
		}),
	})

	p := src(map[string]map[string]string{
		"pv_1": {
			"another_song.length": "2",
			"another_song.0.name": "P0", // overrides A0
			"another_song.1.name": "P1", // appended after A1
		},
	})

	ApplyPatches(db, []Source{p})
	keys := db["pv_1"].renderKeys("20260101")

	want := map[string]string{
		"another_song.length": "3",
		"another_song.0.name": "P0",
		"another_song.1.name": "A1",
		"another_song.2.name": "P1",
	}
	for k, v := range want {
		if keys[k] != v {
			t.Errorf("%s = %q, want %q", k, keys[k], v)
		}
	}
}

func TestApplyPatchesSongFileDedup(t *testing.T) {
	db, _ := Merge([]Source{
		src(map[string]map[string]string{
			"pv_1": {
				"another_song.0.name":            "A0",
				"another_song.1.name":            "Base",
				"another_song.1.song_file_name":  "rom/sound/song/pv_1.ogg",
				"another_song.1.vocal_chara_num": "MIK",
			},
		}),
	})

	// Patch provides same song_file_name → merges into existing entry,
	// patch values overwrite.
	p := src(map[string]map[string]string{
		"pv_1": {
			"another_song.1.name":            "Patched",
			"another_song.1.song_file_name":  "rom/sound/song/pv_1.ogg",
			"another_song.1.vocal_disp_name": "Miku",
		},
	})

	patched := ApplyPatches(db, []Source{p})
	keys := db["pv_1"].renderKeys("20260101")

	// Should be 2 entries (song0 + 1 deduped), not 3.
	if got := keys["another_song.length"]; got != "2" {
		t.Errorf("another_song.length = %q, want %q (dedup)", got, "2")
	}
	// Patch overwrites name.
	if got := keys["another_song.1.name"]; got != "Patched" {
		t.Errorf("another_song.1.name = %q, want %q (patch wins)", got, "Patched")
	}
	// Existing field preserved.
	if got := keys["another_song.1.vocal_chara_num"]; got != "MIK" {
		t.Errorf("another_song.1.vocal_chara_num = %q, want %q", got, "MIK")
	}
	// New field added.
	if got := keys["another_song.1.vocal_disp_name"]; got != "Miku" {
		t.Errorf("another_song.1.vocal_disp_name = %q, want %q", got, "Miku")
	}
	if _, ok := patched["pv_1"]; !ok {
		t.Error("pv_1 must be marked patched")
	}
}

func TestApplyPatchesSongFileDedupNewFile(t *testing.T) {
	db, _ := Merge([]Source{
		src(map[string]map[string]string{
			"pv_1": {
				"another_song.1.name":           "A",
				"another_song.1.song_file_name": "rom/sound/song/pv_1a.ogg",
			},
		}),
	})

	// Different song_file_name → appended as new entry.
	p := src(map[string]map[string]string{
		"pv_1": {
			"another_song.1.name":           "B",
			"another_song.1.song_file_name": "rom/sound/song/pv_1b.ogg",
		},
	})

	ApplyPatches(db, []Source{p})
	keys := db["pv_1"].renderKeys("20260101")

	if got := keys["another_song.length"]; got != "2" {
		t.Errorf("another_song.length = %q, want %q (different files)", got, "2")
	}
}
