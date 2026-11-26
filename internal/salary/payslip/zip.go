package payslip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Zip puts the files in a ZIP archive. Files with the same name get " (2)",
// " (3)", … before the extension, in the order given, and the archive lists
// them by name.
func Zip(files []File, modified time.Time) ([]byte, error) {
	named := make([]File, len(files))
	taken := make(map[string]bool, len(files))
	for i, f := range files {
		name := f.Filename
		for n := 2; taken[name]; n++ {
			name = withSuffix(f.Filename, n)
		}
		taken[name] = true
		named[i] = File{Filename: name, Content: f.Content}
	}

	slices.SortFunc(named, func(a, b File) int { return strings.Compare(a.Filename, b.Filename) })

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range named {
		entry, err := w.CreateHeader(&zip.FileHeader{
			Name:     f.Filename,
			Method:   zip.Deflate,
			Modified: modified,
			// File names are UTF-8: employee names may have non-ASCII letters.
			NonUTF8: false,
		})
		if err != nil {
			return nil, fmt.Errorf("create zip entry %q: %w", f.Filename, err)
		}

		if _, err := entry.Write(f.Content); err != nil {
			return nil, fmt.Errorf("write zip entry %q: %w", f.Filename, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// withSuffix returns "a (n).pdf" for "a.pdf".
func withSuffix(filename string, n int) string {
	base, ext := filename, ""
	if i := strings.LastIndexByte(filename, '.'); i > 0 {
		base, ext = filename[:i], filename[i:]
	}

	return fmt.Sprintf("%s (%d)%s", base, n, ext)
}
