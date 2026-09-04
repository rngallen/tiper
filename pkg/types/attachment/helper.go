package attachment

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var nonFileChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// SanitizeLabel turns a human document label (GLR, customer code, original
// filename stem) into a single path segment. Empty after cleaning is "".
func SanitizeLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == unicode.ReplacementChar || r < 32 {
			continue
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteByte('-')
		default:
			b.WriteRune(r)
		}
	}
	s = nonFileChars.ReplaceAllString(b.String(), "_")
	for strings.Contains(s, "..") {
		s = strings.ReplaceAll(s, "..", ".")
	}
	s = strings.Trim(s, "._-")
	if len(s) > 48 {
		s = strings.Trim(s[:48], "._-")
	}
	return s
}

// EntityDirName is the per-document folder under uploads/{Kind}/YYYY/MM/.
// It is the document number or master code (RCPT:0826-0014, customer C2408067),
// sanitised to a single path segment. The entity ULID is not used on disk.
func EntityDirName(label string) string {
	s := SanitizeLabel(label)
	if s == "" {
		return "unknown"
	}
	return s
}

// fileNameWithoutExt splits a filename into its base name and extension.
func fileNameWithoutExt(fileName string) (name, extension string) {
	extension = filepath.Ext(fileName)
	name = strings.TrimSuffix(fileName, extension)
	return
}

// isPreviewableMime is true for types the SPA can show inline (PDF / images).
func isPreviewableMime(mime, ext string) bool {
	m := strings.ToLower(strings.TrimSpace(mime))
	if m == "image/svg+xml" {
		return false
	}
	if strings.HasPrefix(m, "image/") || m == "application/pdf" {
		return true
	}
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "pdf", "png", "jpg", "jpeg", "gif", "webp", "bmp":
		return true
	default:
		return false
	}
}
