package pvdb

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
)

// headerPrefix marks the comment lines written at the top of a generated
// mod_pv_db.txt. ParseFile skips lines starting with '#'.
const headerPrefix = "# patch_pv_db "

// SourceStamp records one source file in a generated file's header: its
// path and modification time (RFC3339Nano), in input order.
type SourceStamp struct {
	Path  string
	Mtime string
}

// BuildHeader renders the comment header of a generated file: the mod
// version followed by one line per source file, in input order, then a
// blank line separating the header from the pv entries.
func BuildHeader(version string, stamps []SourceStamp) []byte {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%sversion=%s\r\n", headerPrefix, version)
	for _, s := range stamps {
		fmt.Fprintf(&buf, "%ssource %s %s\r\n", headerPrefix, s.Path, s.Mtime)
	}
	buf.WriteString("\r\n")
	return []byte(buf.String())
}

// ParseHeader extracts the version and the ordered source stamps from a
// generated file's comment header. ok is false when the file does not
// start with a version line or a source line is malformed.
func ParseHeader(data []byte) (version string, stamps []SourceStamp, ok bool) {
	return ParseHeaderStream(bytes.NewReader(data))
}

// ParseHeaderStream reads a generated file's comment header from r line
// by line, stopping at the first non-header line so the rest of the file
// is never read. It has the same semantics as ParseHeader.
func ParseHeaderStream(r io.Reader) (version string, stamps []SourceStamp, ok bool) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, headerPrefix) {
			if first {
				return "", nil, false
			}
			break
		}
		rest := line[len(headerPrefix):]
		if first {
			first = false
			if !strings.HasPrefix(rest, "version=") {
				return "", nil, false
			}
			version = rest[len("version="):]
			continue
		}
		if !strings.HasPrefix(rest, "source ") {
			continue
		}
		rest = rest[len("source "):]
		sp := strings.LastIndexByte(rest, ' ')
		if sp < 0 {
			return "", nil, false
		}
		if _, err := time.Parse(time.RFC3339Nano, rest[sp+1:]); err != nil {
			return "", nil, false
		}
		stamps = append(stamps, SourceStamp{Path: rest[:sp], Mtime: rest[sp+1:]})
	}
	if scanner.Err() != nil || first {
		return "", nil, false
	}
	return version, stamps, true
}
