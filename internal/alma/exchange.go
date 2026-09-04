package alma

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"github.com/fsnotify/fsnotify"
	"gorm.io/gorm"
)

// Paths is the ATLAS NEO drop-folder layout used on the gantry share.
type Paths struct {
	Root string
}

func (p Paths) In() string       { return filepath.Join(p.Root, "In") }
func (p Paths) Files() string    { return filepath.Join(p.Root, "Alma", "Files") }
func (p Paths) Archived() string { return filepath.Join(p.Root, "Alma", "Archieved") }
func (p Paths) Rejected() string { return filepath.Join(p.Root, "Alma", "Rejected") }

func (p Paths) Ensure() error {
	for _, d := range []string{p.In(), p.Files(), p.Archived(), p.Rejected()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Completer posts a parsed SAP3R file onto the stock ledger.
type Completer interface {
	CompleteFromAlma(res Result, fileName string) error
}

// WriteOrder writes a SAP3C file into In/ and records AlmaFileLog.
func WriteOrder(db *gorm.DB, paths Paths, o Order, fileName string) (string, error) {
	if err := paths.Ensure(); err != nil {
		return "", err
	}
	if fileName == "" {
		fileName = NewFileName(time.Now())
	}
	body := BuildSAP3C(o)
	tmp := filepath.Join(paths.Root, fileName)
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return "", err
	}
	dest := filepath.Join(paths.In(), fileName)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	_ = db.Create(&models.AlmaFileLog{
		Direction:   types.AlmaOut,
		FileName:    fileName,
		OrderNumber: o.DocNumber,
		Message:     "SAP3C written",
		OK:          true,
	}).Error
	return dest, nil
}

// Watch starts fsnotify on Alma/Files and completes loads after a successful parse.
func Watch(ctx context.Context, db *gorm.DB, paths Paths, done Completer) error {
	if strings.TrimSpace(paths.Root) == "" || done == nil {
		return nil
	}
	if err := paths.Ensure(); err != nil {
		return err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	if err := w.Add(paths.Files()); err != nil {
		_ = w.Close()
		return err
	}
	logs.Infof("ALMA watcher on %s", paths.Files())
	go func() {
		defer w.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Op&fsnotify.Create == 0 {
					continue
				}
				if strings.ToLower(filepath.Ext(ev.Name)) != ".dat" {
					moveDated(ev.Name, paths.Rejected())
					continue
				}
				handleInbound(db, paths, done, ev.Name)
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				logs.Errorf("ALMA watch: %v", err)
			}
		}
	}()
	return nil
}

func handleInbound(db *gorm.DB, paths Paths, done Completer, name string) {
	f, err := openRetry(name, 30)
	if err != nil {
		logs.Errorf("ALMA open %s: %v", name, err)
		moveDated(name, paths.Rejected())
		return
	}
	res, err := ParseSAP3R(f)
	_ = f.Close()
	if err != nil {
		_ = db.Create(&models.AlmaFileLog{
			Direction: types.AlmaIn, FileName: filepath.Base(name), Message: err.Error(),
		}).Error
		moveDated(name, paths.Rejected())
		return
	}
	if err := done.CompleteFromAlma(res, filepath.Base(name)); err != nil {
		_ = db.Create(&models.AlmaFileLog{
			Direction: types.AlmaIn, FileName: filepath.Base(name), OrderNumber: res.OrderNumber,
			Message: err.Error(),
		}).Error
		moveDated(name, paths.Rejected())
		return
	}
	_ = db.Create(&models.AlmaFileLog{
		Direction: types.AlmaIn, FileName: filepath.Base(name), OrderNumber: res.OrderNumber,
		Message: "loaded", OK: true,
	}).Error
	moveDated(name, paths.Archived())
}

func openRetry(path string, n int) (*os.File, error) {
	var last error
	for i := 0; i < n; i++ {
		f, err := os.Open(path)
		if err == nil {
			return f, nil
		}
		last = err
		time.Sleep(time.Second)
	}
	return nil, last
}

func moveDated(src, destRoot string) {
	if src == "" {
		return
	}
	day := filepath.Join(destRoot, time.Now().Format("2006/01/02"))
	_ = os.MkdirAll(day, 0o755)
	dest := filepath.Join(day, filepath.Base(src))
	if err := os.Rename(src, dest); err != nil {
		_ = os.Remove(src)
		logs.Errorf("ALMA move %s: %v", src, err)
	}
}

func EnabledRoot(root string) (Paths, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Paths{}, false
	}
	return Paths{Root: root}, true
}
