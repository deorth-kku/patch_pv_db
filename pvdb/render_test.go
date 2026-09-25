package pvdb

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	db := map[string]*PV{
		"pv_999":  {Base: map[string]string{"bpm": "1"}},
		"pv_1000": {Base: map[string]string{"bpm": "2"}},
		"pv_500":  {Base: map[string]string{"bpm": "3"}}, // single source: filtered out
		"pv_600":  {Base: map[string]string{"bpm": "4"}}, // single source but patched
	}
	sourceCount := map[string]int{"pv_999": 2, "pv_1000": 2, "pv_500": 1, "pv_600": 1}
	patched := map[string]bool{"pv_600": true}

	data, n := Render(db, sourceCount, patched, "20260101")
	if n != 3 {
		t.Fatalf("rendered %d pv, want 3", n)
	}

	want := "pv_1000.bpm=2\r\n" +
		"pv_1000.date=20260101\r\n" +
		"\r\n" +
		"pv_600.bpm=4\r\n" +
		"pv_600.date=20260101\r\n" +
		"\r\n" +
		"pv_999.bpm=1\r\n" +
		"pv_999.date=20260101\r\n"
	if string(data) != want {
		t.Errorf("render = %q, want %q", string(data), want)
	}
	if strings.Contains(string(data), "pv_500") {
		t.Error("single-source pv_500 must not be rendered")
	}
}

func TestRenderEmptyValue(t *testing.T) {
	db := map[string]*PV{"pv_1": {Base: map[string]string{"songinfo.illustrator": ""}}}
	sourceCount := map[string]int{"pv_1": 2}
	data, _ := Render(db, sourceCount, nil, "20260101")
	want := "pv_1.date=20260101\r\npv_1.songinfo.illustrator=\r\n"
	if string(data) != want {
		t.Errorf("render = %q, want %q", string(data), want)
	}
}
