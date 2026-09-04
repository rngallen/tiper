// Package db provides thread-safe, singleton database connection management
// for Microsoft SQL Server (MSSQL) using GORM.
//
// This file specifically handles connection to the **Application database** (or any other MSSQL instance)
// defined in config.Conf.App.Db.
//
// The global instance is exposed as db.Db and must be initialized once
// by calling ConnectDatabase() during application startup.
//
// Features:
//   - Singleton pattern with sync.Once
//   - Safe password URL encoding
//   - Proper MSSQL connection pooling
//   - Custom table naming strategy (singular, no lowercase, CIB → CIB fix)
//   - Graceful shutdown with timeout support
//   - Comprehensive logging and error handling
package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dfms/pkg/config"
	"dfms/pkg/logs"

	"gorm.io/gorm"
)

var (
	// Db holds the global GORM database instance.
	// It is nil until ConnectDatabase() is successfully called.
	Db *gorm.DB

	// once ensures thread-safe singleton initialization of the database connection.
	once sync.Once
)

// Pool configuration constants - tuned for typical web applications
const (
	// defaultMaxIdleConns is the maximum number of connections kept idle in the pool.
	defaultMaxIdleConns = 10

	// defaultMaxOpenConns is the maximum number of open connections to the database.
	// Includes both idle and in-use connections.
	defaultMaxOpenConns = 100

	// defaultConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Helps avoid issues with long-running apps and database restarts.
	defaultConnMaxLifetime = 30 * time.Minute
)

// ConnectDatabase establishes a connection to the MSSQL database using configuration
// from config.Conf.Db. It uses the singleton pattern to ensure only one connection
// is ever created, even under concurrent calls.
//
// The function is safe to call multiple times — only the first call performs initialization.
//
// Features:
//   - Properly URL-encodes the password to support special characters (@, /, etc.)
//   - Sets sensible connection pool limits
//   - Uses Africa/Dar_es_Salaam timezone by default
//   - Includes context-aware ping on startup
//   - Configures GORM logger with custom settings
//
// Returns an error if connection fails or context is canceled.
func ConnectDatabase(ctx context.Context) error {
	var initErr error

	once.Do(func() {
		initErr = connectWithRetry(ctx, "Application", config.Conf.App.Db, &Db)
	})
	return initErr
}

// CloseDatabase gracefully closes the Application database connection with timeout support.
//
// It runs Close() in a goroutine to avoid hanging forever if the server is unresponsive.
// Respects context deadline — ideal for graceful shutdown during SIGTERM.
//
// Safe to call even if connection was never established.
func CloseDatabase(ctx context.Context) error {
	if Db == nil {
		logs.Warn("CloseDatabase called but Db instance is nil")
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	sqlDb, err := Db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB for closing: %w", err)
	}

	// Non-blocking close with timeout
	done := make(chan error, 1)
	go func() {
		done <- sqlDb.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("error closing Application database connection: %w", err)
		}
		logs.Info("Application database connection closed gracefully")
		return nil
	case <-ctx.Done():
		logs.Warn("Timeout while closing Application database connection")
		return ctx.Err()
	}
}
