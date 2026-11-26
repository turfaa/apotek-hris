package payslip

import (
	"fmt"
	"strings"
	"unicode"
)

// SlipFilename returns a slip's file name: "2026-10 - Nama Karyawan - Slip
// Gaji.pdf". The name is trimmed, its whitespace collapsed, and characters
// that are not allowed in file names replaced by "-".
func SlipFilename(year, month int, employeeName string) string {
	return fmt.Sprintf("%04d-%02d - %s - Slip Gaji.pdf", year, month, sanitizeFilenamePart(employeeName))
}

// ZipFilename returns the file name of a month's archive of slips:
// "2026-10 - Slip Gaji.zip".
func ZipFilename(year, month int) string {
	return fmt.Sprintf("%04d-%02d - Slip Gaji.zip", year, month)
}

func sanitizeFilenamePart(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// Collapsed below, tabs and newlines included.
			return ' '
		}
		if unicode.IsControl(r) || strings.ContainsRune(`/\:*?"|<>`, r) {
			return '-'
		}
		return r
	}, s)

	return strings.Join(strings.Fields(s), " ")
}

// ContentDisposition returns an attachment Content-Disposition with both a
// plain ASCII filename and the UTF-8 one (RFC 6266).
func ContentDisposition(filename string) string {
	return fmt.Sprintf(
		`attachment; filename="%s"; filename*=UTF-8''%s`,
		asciiFilename(filename),
		encodeExtValue(filename),
	)
}

// asciiFilename replaces the characters that cannot go in the plain filename
// parameter: non-ASCII ones, and quotes and backslashes, which would end or
// escape the quoted string.
func asciiFilename(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, s)
}

// encodeExtValue percent-encodes s for an RFC 5987 ext-value: every byte
// outside attr-char, including ";", ",", "'", "(" and ")", which would end or
// confuse the parameter.
func encodeExtValue(s string) string {
	const hex = "0123456789ABCDEF"

	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isAttrChar(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0f])
	}

	return b.String()
}

// isAttrChar reports whether c is an RFC 5987 attr-char.
func isAttrChar(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}

	return strings.IndexByte("!#$&+-.^_`|~", c) >= 0
}
