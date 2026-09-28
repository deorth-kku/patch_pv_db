package pvdb

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestMergeFields(t *testing.T) {
	// Two sources in priority order: earlier wins per key, later fills in
	// keys the earlier source did not define.
	a := Source{PVs: map[string]map[string]string{
		"pv_803": {
			"field.01.light": "A_LT",
			"field.01.stage": "A_STG",
			"field.length":   "10",
		},
		"pv_900": {
			"field.01.light": "A_LT",
		},
	}}
	b := Source{PVs: map[string]map[string]string{
		"pv_803": {
			"field.01.light": "B_LT", // loses to a
			"field.02.light": "B_LT", // only in b
			"field.length":   "20",   // loses to a
		},
		"pv_950": {
			"field.01.stage": "B_STG",
		},
	}}

	got := MergeFields([]Source{a, b})

	want := map[string]map[string]string{
		"pv_803": {
			"field.01.light": "A_LT",
			"field.01.stage": "A_STG",
			"field.02.light": "B_LT",
			"field.length":   "10",
		},
		"pv_900": {
			"field.01.light": "A_LT",
		},
		"pv_950": {
			"field.01.stage": "B_STG",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MergeFields = %v, want %v", got, want)
	}
}

func TestMergeFieldsEmpty(t *testing.T) {
	if got := MergeFields(nil); len(got) != 0 {
		t.Errorf("MergeFields(nil) = %v, want empty", got)
	}
}

func TestRenderFields(t *testing.T) {
	fields := map[string]map[string]string{
		"pv_999":  {"field.01.light": "L1", "field.length": "2"},
		"pv_1000": {"field.01.stage": "S1"},
		"pv_500":  {"field.01.light": "L5"}, // not in pvSet: filtered out
		"pv_600":  {},                       // in pvSet but empty: filtered out
	}
	pvSet := StringSet{"pv_999": {}, "pv_1000": {}, "pv_600": {}}

	var buf bytes.Buffer
	n, err := RenderFields(&buf, fields, pvSet)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("rendered %d pv, want 2", n)
	}
	want := "pv_1000.field.01.stage=S1\r\n" +
		"\r\n" +
		"pv_999.field.01.light=L1\r\n" +
		"pv_999.field.length=2\r\n"
	if buf.String() != want {
		t.Errorf("render = %q, want %q", buf.String(), want)
	}
	if strings.Contains(buf.String(), "pv_500") {
		t.Error("pv_500 not in pvSet must not be rendered")
	}
	if strings.Contains(buf.String(), "pv_600") {
		t.Error("pv_600 with no field entries must not be rendered")
	}
}

func TestRenderFieldsEmpty(t *testing.T) {
	var buf bytes.Buffer
	n, err := RenderFields(&buf, map[string]map[string]string{}, StringSet{})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("rendered %d pv, want 0", n)
	}
	if buf.String() != "" {
		t.Errorf("render = %q, want empty", buf.String())
	}
}
