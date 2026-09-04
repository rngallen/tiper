package app

import (
	"context"
	"sync"
	"time"

	authmiddleware "dfms/apps/auth/middleware"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"dfms/pkg/mail"

	"github.com/gofiber/fiber/v3"
	"github.com/robfig/cron/v3"
	"gopkg.in/natefinch/lumberjack.v2"
)

// shutdown performs an orderly teardown: stop accepting requests, stop cron,
// wait for background goroutines, then close databases and log files. Errors
// are logged but do not abort the sequence so every resource gets a chance to
// release.
func shutdown(
	httpApp *fiber.App,
	cronScheduler *cron.Cron,
	dbLogger, appLogger, accessLogger *lumberjack.Logger,
	wg *sync.WaitGroup,
	timeout time.Duration,
) error {
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	logs.Info("Shutting down HTTP server...")
	if err := httpApp.ShutdownWithContext(shutdownCtx); err != nil {
		logs.Errorf("http shutdown: %v", err)
	}

	logs.Info("Stopping cron scheduler...")
	if cronScheduler != nil {
		stopCtx := cronScheduler.Stop()
		select {
		case <-stopCtx.Done():
		case <-shutdownCtx.Done():
			logs.Warn("timeout stopping cron scheduler")
		}
	}

	logs.Info("Waiting for background workers...")
	wg.Wait()

	logs.Info("Closing databases...")
	if err := db.CloseDatabase(shutdownCtx); err != nil {
		logs.Errorf("close app db: %v", err)
	}
	if err := db.CloseSageDb(shutdownCtx); err != nil {
		logs.Errorf("close sage db: %v", err)
	}

	mail.Close()
	authmiddleware.ClosePermissionCache()
	if err := db.CloseDbLogger(); err != nil {
		logs.Errorf("close db logger: %v", err)
	}

	logs.Info("Closing loggers...")
	for _, l := range []*lumberjack.Logger{dbLogger, appLogger, accessLogger} {
		if l != nil {
			_ = l.Close()
		}
	}
	return nil
}
