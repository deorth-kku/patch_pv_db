package pvdb

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseFile(t *testing.T) {
	content := "\xEF\xBB\xBFpv_1.bpm=120\r\n\r\npv_1.songinfo.illustrator=\r\npv_2.x=a=b\r\nnokeyline\r\npv_3.song_name=曲名\r\n"
	src, err := ParseFile(writeTemp(t, "db.txt", content))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]map[string]string{
		"pv_1": {"bpm": "120", "songinfo.illustrator": ""},
		"pv_2": {"x": "a=b"},
		"pv_3": {"song_name": "曲名"},
	}
	if len(src.PVs) != len(want) {
		t.Fatalf("got %d pv, want %d: %v", len(src.PVs), len(want), src.PVs)
	}
	for id, fields := range want {
		for k, v := range fields {
			if got := src.PVs[id][k]; got != v {
				t.Errorf("%s.%s = %q, want %q", id, k, got, v)
			}
		}
	}
}

func TestParseFileKeepsEmptyLanguageValues(t *testing.T) {
	// The parser is raw: empty values (including empty language fields)
	// are kept byte-for-byte. Dropping empty language fields is a
	// merge-level rule (see splitFields / TestMergeEmptyLanguageValueDoesNotBlock).
	content := "pv_1.another_song.0.name=FACE\r\n" +
		"pv_1.another_song.0.name_en=\r\n" +
		"pv_1.another_song.1.name_en=English\r\n"
	src, err := ParseFile(writeTemp(t, "db.txt", content))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]map[string]string{
		"pv_1": {
			"another_song.0.name":    "FACE",
			"another_song.0.name_en": "",
			"another_song.1.name_en": "English",
		},
	}
	if len(src.PVs) != 1 || len(src.PVs["pv_1"]) != len(want["pv_1"]) {
		t.Fatalf("got %v, want %v", src.PVs, want)
	}
	for k, v := range want["pv_1"] {
		if got := src.PVs["pv_1"][k]; got != v {
			t.Errorf("pv_1.%s = %q, want %q", k, got, v)
		}
	}
}
