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
