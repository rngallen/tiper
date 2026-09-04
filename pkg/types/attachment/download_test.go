package attachment

import (
	"path/filepath"
	"testing"
)

func TestResolveSafeUploadPath(t *testing.T) {
	root, err := filepath.Abs(Root())
	if err != nil {
		t.Fatal(err)
	}
	ok := filepath.Join(root, "Invoices", "2026", "01", "1", "doc.pdf")
	got, err := resolveSafeUploadPath(ok)
	if err != nil {
		t.Fatalf("expected ok path: %v", err)
	}
	if got != filepath.Clean(ok) {
		t.Fatalf("got %q want %q", got, filepath.Clean(ok))
	}

	for _, bad := range []string{
		"",
		"../etc/passwd",
		filepath.Join(root, "..", "etc", "passwd"),
		"/etc/passwd",
	} {
		if _, err := resolveSafeUploadPath(bad); err == nil {
			t.Errorf("expected rejection for %q", bad)
		}
	}
}
