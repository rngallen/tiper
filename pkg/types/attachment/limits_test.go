package attachment

import (
	"path/filepath"
	"testing"

	"dfms/pkg/config"
)

func TestClampUploads(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		in     config.UploadsConfig
		wantMB int
		wantN  int
	}{
		{
			name:   "zero becomes floor",
			in:     config.UploadsConfig{},
			wantMB: minFileSizeMB,
			wantN:  minFiles,
		},
		{
			name:   "caps file size",
			in:     config.UploadsConfig{MaxFileSizeMB: 100, MaxFilesPerRequest: 1},
			wantMB: maxFileSizeMBCap,
			wantN:  1,
		},
		{
			name:   "caps file count",
			in:     config.UploadsConfig{MaxFileSizeMB: 2, MaxFilesPerRequest: 99},
			wantMB: 2,
			wantN:  maxFilesCap,
		},
		{
			name:   "product stays under process body limit",
			in:     config.UploadsConfig{MaxFileSizeMB: 25, MaxFilesPerRequest: 10},
			wantMB: 25,
			wantN:  2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClampUploads(tc.in)
			if got.MaxFileSizeMB != tc.wantMB || got.MaxFilesPerRequest != tc.wantN {
				t.Fatalf("got %+v want mb=%d n=%d", got, tc.wantMB, tc.wantN)
			}
			if got.Directory != defaultUploadDir {
				t.Fatalf("empty directory should default, got %q", got.Directory)
			}
		})
	}
}

func TestClampUploadsDirectory(t *testing.T) {
	t.Parallel()
	got := ClampUploads(config.UploadsConfig{Directory: `D:\dfms\files`, MaxFileSizeMB: 5, MaxFilesPerRequest: 2})
	if got.Directory == defaultUploadDir || got.Directory == "" {
		t.Fatalf("kept operator path: %q", got.Directory)
	}
	if ClampUploads(config.UploadsConfig{Directory: "../etc"}).Directory != defaultUploadDir {
		t.Fatal(".. must fall back to default")
	}
}

func TestApplyLimitsDirectory(t *testing.T) {
	t.Cleanup(func() {
		ApplyLimits(config.UploadsConfig{
			Directory:          defaultUploadDir,
			MaxFileSizeMB:      defaultFileSizeMB,
			MaxFilesPerRequest: defaultMaxFiles,
		})
	})
	dir := filepath.Join(t.TempDir(), "uploads")
	ApplyLimits(config.UploadsConfig{Directory: dir, MaxFileSizeMB: 5, MaxFilesPerRequest: 2})
	if Root() != filepath.Clean(dir) {
		t.Fatalf("Root()=%q want %q", Root(), filepath.Clean(dir))
	}
}
