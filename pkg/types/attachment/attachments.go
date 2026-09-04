// Package attachment implements generic file-attachment handling shared by
// every entity that carries documents (receipts, orders, billing runs): multipart
// upload validation (size, extension whitelist, per-request count), storage on
// disk under {root}/{Category}/YYYY/MM/{document-number-or-code} (Settings → Attachments), and
// download/count helpers backed by the Attachment model.
package attachment

import (
	"dfms/pkg/config"
	"dfms/pkg/types"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// ProcessBodyLimit is Fiber's hard request-body cap (64 MiB). Raising this
// requires a process restart. A 1 GiB cap would let a single request pin a
// large buffer and is not used. Settings → Attachments can only go up to this.
const ProcessBodyLimit = 64 << 20

const (
	defaultFileSizeMB = 10
	defaultMaxFiles   = 5
	minFileSizeMB     = 2
	maxFileSizeMBCap  = 25
	minFiles          = 1
	maxFilesCap       = 10
	defaultUploadDir  = "./uploads"
	maxUploadDirRunes = 260
)

var (
	maxFileSize atomic.Int64
	maxFiles    atomic.Int64
	uploadDir   atomic.Value // string
)

func init() {
	ApplyLimits(config.UploadsConfig{
		Directory:          defaultUploadDir,
		MaxFileSizeMB:      defaultFileSizeMB,
		MaxFilesPerRequest: defaultMaxFiles,
	})
}

// Root is the live directory new attachments are written under (Settings → Attachments).
func Root() string {
	if v, ok := uploadDir.Load().(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return defaultUploadDir
}

// MaxFileSize is the live per-file upload limit in bytes (Settings → Attachments).
func MaxFileSize() int64 { return maxFileSize.Load() }

// MaxFilesPerRequest is the live per-request file count cap.
func MaxFilesPerRequest() int { return int(maxFiles.Load()) }

// ApplyLimits installs operator-chosen upload caps, clamped so
// files × size cannot exceed ProcessBodyLimit.
func ApplyLimits(cfg config.UploadsConfig) config.UploadsConfig {
	cfg = ClampUploads(cfg)
	maxFileSize.Store(int64(cfg.MaxFileSizeMB) * 1024 * 1024)
	maxFiles.Store(int64(cfg.MaxFilesPerRequest))
	uploadDir.Store(cfg.Directory)
	return cfg
}

// EnsureDir creates the attachment root so a Settings save fails if the path is not writable.
func EnsureDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = defaultUploadDir
	}
	return os.MkdirAll(dir, 0755)
}

// ClampUploads enforces ERP bounds: 1–25 MB per file, 1–10 files, product ≤ 64 MiB.
// MaxFileSizeMB is a ceiling — files smaller than 1 MB are always accepted.
func ClampUploads(cfg config.UploadsConfig) config.UploadsConfig {
	mb := cfg.MaxFileSizeMB
	n := cfg.MaxFilesPerRequest
	if mb < minFileSizeMB {
		mb = minFileSizeMB
	}
	if mb > maxFileSizeMBCap {
		mb = maxFileSizeMBCap
	}
	if n < minFiles {
		n = minFiles
	}
	if n > maxFilesCap {
		n = maxFilesCap
	}
	for int64(mb)*int64(n)*1024*1024 > ProcessBodyLimit {
		if n > minFiles {
			n--
			continue
		}
		if mb > minFileSizeMB {
			mb--
			continue
		}
		break
	}
	return config.UploadsConfig{
		Directory:          clampUploadDir(cfg.Directory),
		MaxFileSizeMB:      mb,
		MaxFilesPerRequest: n,
	}
}

func clampUploadDir(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.Contains(s, "..") || strings.ContainsRune(s, 0) || len(s) > maxUploadDirRunes {
		return defaultUploadDir
	}
	s = filepath.Clean(s)
	if s == "." || strings.Contains(s, "..") {
		return defaultUploadDir
	}
	return s
}

// ──────────────────────────────────────────────────────────────
//  File Upload — Allowed MIME Types & Extensions
// ──────────────────────────────────────────────────────────────

// File extensions (with dot) — used in upload validation
const (
	extExcel     = ".xls"
	extExcelX    = ".xlsx"
	extCSV       = ".csv"
	extWord      = ".doc"
	extWordX     = ".docx"
	extPDF       = ".pdf"
	extPublisher = ".pub"
	extMsg       = ".msg" // Outlook desktop message
	extEml       = ".eml" // standard / Outlook web message
	extJPEG      = ".jpeg"
	extJPG       = ".jpg"
	extPNG       = ".png"
	extGIF       = ".gif"
	extAVIF      = ".avif"
	extTIFF      = ".tiff"
)

var allowedFileExtensions = []string{
	extExcel, extExcelX, extCSV, extWord, extWordX, extPDF, extPublisher,
	extMsg, extEml, extJPEG, extJPG, extPNG, extGIF, extAVIF, extTIFF,
}

// isAllowedFileExtension reports whether the extension is on the allow list (case-insensitive).
func isAllowedFileExtension(extension string) bool {
	return slices.Contains(allowedFileExtensions, strings.ToLower(extension))
}

func categoryDir(ct types.ContentType) string {
	if s := types.ContentTypeFolder(ct); s != "" {
		return s
	}
	if s := SanitizeLabel(types.ContentTypeLabel(ct)); s != "" && s != "Document" {
		return s
	}
	return fmt.Sprintf("type-%d", ct)
}

// createEntityPath returns (and creates) {root}/{Kind}/YYYY/MM/{folderKey}.
func createEntityPath(ct types.ContentType, folderKey string) (string, error) {
	folderKey = strings.TrimSpace(folderKey)
	if folderKey == "" || strings.Contains(folderKey, "..") || strings.ContainsAny(folderKey, `/\`) {
		return "", fmt.Errorf("invalid attachment folder")
	}
	now := time.Now()
	dirPath := filepath.Join(
		Root(),
		categoryDir(ct),
		fmt.Sprintf("%d", now.Year()),
		now.Format("01"),
		folderKey,
	)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", err
	}
	return dirPath, nil
}

// byteSize converts a byte count to a clean, human-readable string (e.g. "1.80 MB", "245.00 KB").
func byteSize(bytes int64) string {
	const unit = 1024
	if bytes == 0 {
		return "0 B"
	}

	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	size := float64(bytes)

	for size >= unit && i < len(units)-1 {
		size /= unit
		i++
	}

	return fmt.Sprintf("%.2f %s", size, units[i])
}
