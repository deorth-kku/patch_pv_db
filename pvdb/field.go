package pvdb

import (
	"fmt"
	"io"
	"slices"
	"sort"
)

// MergeFields merges the field sources in priority order: earlier sources
// win every key conflict. The result maps pv id to field key to value.
//
// Unlike Merge, no another_song handling is performed: mod_pv_field.txt
// files are flat key=value files (field.NN.xxx entries) and are merged
// first-wins per key.
func MergeFields(sources []Source) map[string]map[string]string {
	fields := map[string]map[string]string{}
	for _, src := range sources {
		for pvID, kv := range src.PVs {
			dst, ok := fields[pvID]
			if !ok {
				dst = map[string]string{}
				fields[pvID] = dst
			}
			for key, value := range kv {
				if _, exists := dst[key]; !exists {
					dst[key] = value
				}
			}
		}
	}
	return fields
}

// RenderFields writes the merged field data to w, including only the pv ids
// in pvSet that have at least one field entry.
//
// Format: pv ids in lexicographic string order (pv_1000 before pv_999),
// fields within a pv in lexicographic order, one blank line between pv
// blocks, CRLF line endings, and a single trailing CRLF. No date field and
// no header are written.
// The count is the number of fully rendered pv ids; it is partial when
// the returned error is non-nil.
func RenderFields(w io.Writer, fields map[string]map[string]string, pvSet StringSet) (int, error) {
	ids := make([]string, 0, len(pvSet))
	for id := range pvSet {
		if len(fields[id]) > 0 {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)

	for i, id := range ids {
		if i > 0 {
			if _, err := fmt.Fprint(w, "\r\n"); err != nil {
				return i, err
			}
		}
		keys := fields[id]
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			if _, err := fmt.Fprintf(w, "%s.%s=%s\r\n", id, k, keys[k]); err != nil {
				return i, err
			}
		}
	}
	return len(ids), nil
}
