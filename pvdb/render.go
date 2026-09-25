package pvdb

import (
	"fmt"
	"io"
	"slices"
	"sort"
)

// Render renders the merged db to the output format and writes it to w.
//
// Only pv ids with sourceCount >= 2, or that were touched by a patch,
// are included. All included pv ids get date = date.
//
// Format: pv ids in lexicographic string order (pv_1000 before pv_999),
// fields within a pv in lexicographic order, one blank line between pv
// blocks, CRLF line endings, and a single trailing CRLF.
// The count is the number of fully rendered pv ids; it is partial when
// the returned error is non-nil.
func Render(w io.Writer, db map[string]*PV, sourceCount map[string]int, patched StringSet, date string) (int, error) {
	ids := make([]string, 0, len(sourceCount))
	for id, count := range sourceCount {
		if _, isPatched := patched[id]; count >= 2 || isPatched {
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
		keys := db[id].renderKeys(date)
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
