package db

import (
	"sync"
	"time"

	"dfms/pkg/config"
	"dfms/pkg/logs"

	"gorm.io/gorm/logger"
)

var (
	logFileMutex sync.Mutex
	cachedLogger logger.Interface // Cached so the app DB and Sage DB share one GORM logger
)

// dbLogger returns a GORM logger that writes structured SQL through zerolog
// to logs/db.log (see logs.InitDBZerolog / logs.Setup).
//
// Behaviour:
//   - Thread-safe; the logger is built at most once and shared across callers.
//   - In debug mode, all queries are logged at Info with a 50ms slow-query
//     threshold; queries include concrete values.
//   - In non-debug mode, queries are parameterised and only errors and slow
//     queries are emitted.
func dbLogger() logger.Interface {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	if cachedLogger != nil {
		return cachedLogger
	}

	debug := config.Conf.App.Debug

	level := logger.Warn
	slow := 200 * time.Millisecond
	if debug {
		level = logger.Info
		slow = 50 * time.Millisecond
	}

	cachedLogger = logger.New(
		logs.GormWriter{},
		logger.Config{
			SlowThreshold:             slow,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      !debug, // hide real values outside debug mode
			Colorful:                  false,
		},
	)
	return cachedLogger
}

// CloseDbLogger drops the cached GORM logger so shutdown can close the
// lumberjack db.log handle afterwards. Safe to call multiple times.
func CloseDbLogger() error {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()
	cachedLogger = nil
	return nil
}
