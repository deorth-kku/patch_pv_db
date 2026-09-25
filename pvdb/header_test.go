package pvdb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildHeader(t *testing.T) {
	stamps := []SourceStamp{
		{Path: `C:\game\mods\a\rom\mod_pv_db.txt`, Mtime: "2025-09-25T12:00:00.1234567+08:00"},
		{Path: `C:\game\mods\b\rom\patch_pv_db.txt`, Mtime: "2025-09-25T12:00:01+08:00"},
	}
	want := "# patch_pv_db version=1.2.3\r\n" +
		"# patch_pv_db source C:\\game\\mods\\a\\rom\\mod_pv_db.txt 2025-09-25T12:00:00.1234567+08:00\r\n" +
		"# patch_pv_db source C:\\game\\mods\\b\\rom\\patch_pv_db.txt 2025-09-25T12:00:01+08:00\r\n" +
		"\r\n"
	if got := string(BuildHeader("1.2.3", stamps)); got != want {
		t.Errorf("BuildHeader = %q, want %q", got, want)
	}
}

func TestParseHeaderRoundTrip(t *testing.T) {
	stamps := []SourceStamp{
		{Path: `C:\game\mods\a\rom\mod_pv_db.txt`, Mtime: "2025-01-01T00:00:00Z"},
		{Path: `C:\game\mods b\rom\mod_pv_db.txt`, Mtime: "2025-01-02T00:00:00Z"}, // path with a space
		{Path: `C:\game\mods\a\rom\patch_pv_db.txt`, Mtime: "2025-01-03T00:00:00Z"},
	}
	data := BuildHeader("0.9", stamps)
	version, got, ok := ParseHeader(data)
	if !ok || version != "0.9" || len(got) != len(stamps) {
		t.Fatalf("ParseHeader = %q, %v, %v", version, got, ok)
	}
	for i := range stamps {
		if got[i] != stamps[i] {
			t.Errorf("stamp[%d] = %v, want %v", i, got[i], stamps[i])
		}
	}
}

func TestParseHeaderNoHeader(t *testing.T) {
	if _, _, ok := ParseHeader([]byte("pv_1.bpm=1\r\n")); ok {
		t.Error("ParseHeader on a plain file must not report ok")
	}
	if _, _, ok := ParseHeader([]byte("")); ok {
		t.Error("ParseHeader on an empty file must not report ok")
	}
}

func TestParseHeaderMalformed(t *testing.T) {
	bad := "# patch_pv_db version=1\r\n# patch_pv_db source a.txt notatime\r\n"
	if _, _, ok := ParseHeader([]byte(bad)); ok {
		t.Error("ParseHeader must not report ok for a malformed source line")
	}
}

func TestParseFileIgnoresHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mod_pv_db.txt")
	data := BuildHeader("1.0", []SourceStamp{{Path: "x", Mtime: "2025-01-01T00:00:00Z"}})
	data = append(data, []byte("pv_1.bpm=120\r\n")...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.PVs) != 1 || src.PVs["pv_1"]["bpm"] != "120" {
		t.Errorf("ParseFile = %v, want only pv_1.bpm=120", src.PVs)
	}
}
