// Package logs is the process logger: rotating files (app/access/db), zerolog
// wrappers, and panic-safe goroutines. Close lumberjack handles on shutdown.
package logs

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logsDir           = "logs"
	appLogFileName    = "app.log"
	accessLogFileName = "access.log"
	dbLogFileName     = "db.log"

	// Default rotation policy.
	defaultMaxSizeMB = 100
)

// Setup opens rotating files and points every logger (pkg/logs, zerolog,
// Fiber, stdlib, GORM) at them. This is the initializer callers should use.
func Setup() (appLogger, accessLogger, dbLogger *lumberjack.Logger, err error) {
	appLogger, accessLogger, dbLogger, err = InitLogs()
	if err != nil {
		return nil, nil, nil, err
	}
	InitZerolog(appLogger)
	InitDBZerolog(dbLogger)
	return appLogger, accessLogger, dbLogger, nil
}

// InitLogs creates the logs/ directory and three rotating writers (app, access,
// db). Prefer Setup(), which also wires zerolog onto those writers.
func InitLogs() (appLogger, accessLogger, dbLogger *lumberjack.Logger, err error) {
	if mkErr := os.MkdirAll(logsDir, 0o755); mkErr != nil {
		return nil, nil, nil, fmt.Errorf("logs: create dir %q: %w", logsDir, mkErr)
	}

	appLogger = newRotatingLogger(appLogFileName)
	accessLogger = newRotatingLogger(accessLogFileName)
	dbLogger = newRotatingLogger(dbLogFileName)

	return appLogger, accessLogger, dbLogger, nil
}

// newRotatingLogger constructs a lumberjack writer with the project's default
// rotation policy.
func newRotatingLogger(filename string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:  filepath.Join(logsDir, filename),
		MaxSize:   defaultMaxSizeMB,
		LocalTime: true,
		Compress:  false,
	}
}
