package pvdb

import "testing"

func src(pvs map[string]map[string]string) Source {
	return Source{PVs: pvs}
}

func TestMergePriorityWins(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_2": {"bpm": "100", "song_name": "A"},
		"pv_1": {"bpm": "200"},
	})
	b := src(map[string]map[string]string{
		"pv_2": {"bpm": "999", "extra": "B"},
	})

	db, count := Merge([]Source{a, b})

	if got := db["pv_2"].Base["bpm"]; got != "100" {
		t.Errorf("pv_2.bpm = %q, want %q (earlier source wins)", got, "100")
	}
	if got := db["pv_2"].Base["song_name"]; got != "A" {
		t.Errorf("pv_2.song_name = %q, want %q", got, "A")
	}
	if got := db["pv_2"].Base["extra"]; got != "B" {
		t.Errorf("pv_2.extra = %q, want %q (union)", got, "B")
	}
	if count["pv_2"] != 2 || count["pv_1"] != 1 {
		t.Errorf("sourceCount = %v, want pv_2:2 pv_1:1", count)
	}
}

func TestMergeAnotherSongAccumulate(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {
			"bpm":                  "100",
			"another_song.length":  "2",
			"another_song.0.name":  "A0",
			"another_song.1.name":  "A1",
			"another_song.1.vocal": "A1v",
		},
	})
	b := src(map[string]map[string]string{
		"pv_1": {
			"bpm":                  "999",
			"another_song.length":  "2",
			"another_song.0.name":  "B0", // conflicts: A0 wins
			"another_song.0.vocal": "B0v",
			"another_song.1.name":  "B1",
		},
	})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	// shared index 0 + a's index 1 + b's index 1 = 3 entries
	want := map[string]string{
		"bpm":                  "100",
		"date":                 "20260101",
		"another_song.length":  "3",
		"another_song.0.name":  "A0",
		"another_song.0.vocal": "B0v",
		"another_song.1.name":  "A1",
		"another_song.1.vocal": "A1v",
		"another_song.2.name":  "B1",
	}
	for k, v := range want {
		if keys[k] != v {
			t.Errorf("%s = %q, want %q", k, keys[k], v)
		}
	}
	if len(keys) != len(want) {
		t.Errorf("rendered %d keys, want %d: %v", len(keys), len(want), keys)
	}
}

func TestMergeSingleSourceAnotherSongUntouched(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {
			"another_song.length": "2",
			"another_song.0.name": "A0",
			"another_song.1.name": "A1",
		},
	})
	db, _ := Merge([]Source{a})
	keys := db["pv_1"].renderKeys("20260101")
	if keys["another_song.length"] != "2" || keys["another_song.0.name"] != "A0" || keys["another_song.1.name"] != "A1" {
		t.Errorf("single source another_song changed: %v", keys)
	}
}

func TestMergeSong0NameFromOwnerSongName(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {"song_name": "Owner Title"},
	})
	b := src(map[string]map[string]string{
		"pv_1": {
			"another_song.0.name":           "B0",
			"another_song.0.song_file_name": "rom/sound/song/pv_1.ogg",
		},
	})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	if got := keys["another_song.0.name"]; got != "Owner Title" {
		t.Errorf("another_song.0.name = %q, want %q (owner song_name)", got, "Owner Title")
	}
	if got := keys["another_song.0.song_file_name"]; got != "rom/sound/song/pv_1.ogg" {
		t.Errorf("another_song.0.song_file_name = %q, want unchanged", got)
	}
}

func TestMergeSong0NameOwnerDefinedWins(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {
			"song_name":           "Owner Title",
			"another_song.0.name": "A0",
		},
	})
	b := src(map[string]map[string]string{
		"pv_1": {"another_song.0.name": "B0"},
	})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	if got := keys["another_song.0.name"]; got != "A0" {
		t.Errorf("another_song.0.name = %q, want %q (owner defined it)", got, "A0")
	}
}

func TestMergeSong0NameNoOwnerSongNameFallback(t *testing.T) {
	a := src(map[string]map[string]string{"pv_1": {"bpm": "100"}})
	b := src(map[string]map[string]string{"pv_1": {"another_song.0.name": "B0"}})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	if got := keys["another_song.0.name"]; got != "B0" {
		t.Errorf("another_song.0.name = %q, want %q (no owner song_name)", got, "B0")
	}
}

func TestMergeSong0NameOwnerEmptySongName(t *testing.T) {
	a := src(map[string]map[string]string{"pv_1": {"song_name": ""}})
	b := src(map[string]map[string]string{"pv_1": {"another_song.0.name": "B0"}})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	if got := keys["another_song.0.name"]; got != "" {
		t.Errorf("another_song.0.name = %q, want %q (owner song_name is empty)", got, "")
	}
}

func TestMergeSongFileDedup(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {
			"another_song.0.name":            "A0",
			"another_song.1.name":            "Base",
			"another_song.1.song_file_name":  "rom/sound/song/pv_1.ogg",
			"another_song.1.vocal_chara_num": "MIK",
		},
	})
	b := src(map[string]map[string]string{
		"pv_1": {
			"another_song.1.name":            "Base",
			"another_song.1.song_file_name":  "rom/sound/song/pv_1.ogg",
			"another_song.1.vocal_disp_name": "Miku",
		},
	})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	// Should be 2 entries (song0 + 1 deduped), not 3.
	if got := keys["another_song.length"]; got != "2" {
		t.Errorf("another_song.length = %q, want %q (dedup by song_file_name)", got, "2")
	}
	// First-wins: a's fields are kept, b's new field is added.
	if got := keys["another_song.1.name"]; got != "Base" {
		t.Errorf("another_song.1.name = %q, want %q", got, "Base")
	}
	if got := keys["another_song.1.vocal_chara_num"]; got != "MIK" {
		t.Errorf("another_song.1.vocal_chara_num = %q, want %q", got, "MIK")
	}
	if got := keys["another_song.1.vocal_disp_name"]; got != "Miku" {
		t.Errorf("another_song.1.vocal_disp_name = %q, want %q (union)", got, "Miku")
	}
}

func TestMergeSongFileDedupDifferentFiles(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {
			"another_song.1.name":           "A",
			"another_song.1.song_file_name": "rom/sound/song/pv_1a.ogg",
		},
	})
	b := src(map[string]map[string]string{
		"pv_1": {
			"another_song.1.name":           "B",
			"another_song.1.song_file_name": "rom/sound/song/pv_1b.ogg",
		},
	})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	// Different song_file_name → two entries.
	if got := keys["another_song.length"]; got != "2" {
		t.Errorf("another_song.length = %q, want %q (different files)", got, "2")
	}
}

func TestMergeSongFileNoSfnameAppends(t *testing.T) {
	a := src(map[string]map[string]string{
		"pv_1": {"another_song.1.name": "A"},
	})
	b := src(map[string]map[string]string{
		"pv_1": {"another_song.1.name": "B"},
	})

	db, _ := Merge([]Source{a, b})
	keys := db["pv_1"].renderKeys("20260101")

	// No song_file_name → cannot dedup, both appended.
	if got := keys["another_song.length"]; got != "2" {
		t.Errorf("another_song.length = %q, want %q (no sfname)", got, "2")
	}
}
