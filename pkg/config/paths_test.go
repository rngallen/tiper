package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("DFMS.DEBUG=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := findConfigFile(".env", []string{dir, "."})
	if got != path {
		t.Fatalf("got %q want %q", got, path)
	}
	if findConfigFile("missing.env", []string{dir}) != "" {
		t.Fatal("missing file must be empty")
	}
}

func TestSecretsSearchDirsExcludeCwd(t *testing.T) {
	t.Setenv("DFMS_CONFIG_DIR", "")
	dirs := secretsSearchDirs()
	for _, d := range dirs {
		if d == "." || d == "config" {
			t.Fatalf("secrets must not load from %q", d)
		}
	}
	if got := PlatformConfigDir(); got == "" {
		t.Fatal("platform dir")
	}
	if runtime.GOOS == "windows" {
		if filepath.Base(PlatformConfigDir()) != "DFMS" {
			t.Fatalf("windows dir: %s", PlatformConfigDir())
		}
	}
}

func TestConfigSearchDirsHonourOverride(t *testing.T) {
	t.Setenv("DFMS_CONFIG_DIR", `D:\dfms-config`)
	dirs := configSearchDirs()
	if len(dirs) == 0 || dirs[0] != `D:\dfms-config` {
		t.Fatalf("override first: %#v", dirs)
	}
}
