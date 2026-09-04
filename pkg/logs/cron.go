package logs

import (
	"context"
	"fmt"

	"dfms/internal/jobs"

	"gopkg.in/natefinch/lumberjack.v2"
)

// RegisterLogRotationJob wires daily (or Settings-configured) log rotation onto
// the jobs manager. Specs come from Settings → Schedules. writers are typically
// app, access, then db lumberjack handles from Setup().
func RegisterLogRotationJob(ctx context.Context, m *jobs.Manager, writers ...*lumberjack.Logger) error {
	if m == nil {
		return fmt.Errorf("logs: jobs manager is nil")
	}
	if len(writers) == 0 {
		return fmt.Errorf("logs: no log writers to rotate")
	}
	for i, w := range writers {
		if w == nil {
			return fmt.Errorf("logs: logger %d is nil", i)
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	labels := []string{"application", "access", "database"}
	m.Register(jobs.LogRotation, func() {
		if ctx.Err() != nil {
			Warn("Log rotation skipped due to context cancellation")
			return
		}
		Info("Rotating log files")
		for i, w := range writers {
			label := fmt.Sprintf("log[%d]", i)
			if i < len(labels) {
				label = labels[i]
			}
			if err := w.Rotate(); err != nil {
				Errorf("Failed to rotate %s log: %v", label, err)
			} else {
				Infof("%s log rotated successfully", label)
			}
		}
	})
	return nil
}
