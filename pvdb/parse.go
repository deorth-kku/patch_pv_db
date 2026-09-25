package pvdb

import (
	"bufio"
	"os"
	"strings"
)

// maxLineSize bounds a single line in a mod_pv_db.txt file. Values are
// short text fields, so 1 MiB is far beyond any realistic line.
const maxLineSize = 1024 * 1024

// ParseFile parses a mod_pv_db.txt style file: lines of "key=value"
// separated by the first '='. Empty lines are skipped, a UTF-8 BOM is
// stripped, and both CRLF and LF line endings are accepted. Values are
// kept byte-for-byte (they may contain '=' or be empty). The file is
// read line by line, so memory use stays flat for very large files.
func ParseFile(path string) (Source, error) {
	f, err := os.Open(path)
	if err != nil {
		return Source{}, err
	}
	defer f.Close()

	src := Source{
		Path: path,
		PVs:  map[string]map[string]string{},
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			line = strings.TrimPrefix(line, "\uFEFF")
			first = false
		}
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		before, after, ok := strings.Cut(key, ".")
		if !ok {
			continue
		}
		pvID := before
		field := after
		if !strings.HasPrefix(pvID, "pv_") {
			continue
		}
		if src.PVs[pvID] == nil {
			src.PVs[pvID] = map[string]string{}
		}
		src.PVs[pvID][field] = value
	}
	if err := scanner.Err(); err != nil {
		return Source{}, err
	}
	return src, nil
}
