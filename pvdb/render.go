package pvdb

import (
	"bytes"
	"sort"
)

// Render renders the merged db to the output format.
//
// Only pv ids with sourceCount >= 2, or that were touched by a patch,
// are included. All included pv ids get date = date.
//
// Format: pv ids in lexicographic string order (pv_1000 before pv_999),
// fields within a pv in lexicographic order, one blank line between pv
// blocks, CRLF line endings, and a single trailing CRLF.
// The second return value is the number of rendered pv ids.
func Render(db map[string]*PV, sourceCount map[string]int, patched map[string]bool, date string) ([]byte, int) {
	ids := make([]string, 0, len(sourceCount))
	for id, count := range sourceCount {
		if count >= 2 || patched[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	var buf bytes.Buffer
	for i, id := range ids {
		if i > 0 {
			buf.WriteString("\r\n")
		}
		keys := db[id].renderKeys(date)
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			buf.WriteString(id)
			buf.WriteByte('.')
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(keys[k])
			buf.WriteString("\r\n")
		}
	}
	return buf.Bytes(), len(ids)
}
