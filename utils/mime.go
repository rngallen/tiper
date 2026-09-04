package utils

import (
	"mime"
	"strings"
)

// GetMIME returns the IANA media type for a file extension (with or without
// a leading dot). Unknown extensions map to application/octet-stream.
func GetMIME(ext string) string {
	ext = strings.TrimSpace(ext)
	if ext == "" {
		return "application/octet-stream"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	t := mime.TypeByExtension(strings.ToLower(ext))
	if t == "" {
		return "application/octet-stream"
	}
	if i := strings.IndexByte(t, ';'); i >= 0 {
		t = t[:i]
	}
	return strings.TrimSpace(t)
}
