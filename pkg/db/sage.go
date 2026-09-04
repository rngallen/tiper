package db

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"dfms/pkg/config"
	"dfms/pkg/logs"

	"gorm.io/gorm"
)

var sageMu sync.RWMutex

// sageDB is the live Sage 200 connection. Nil until Settings → Sage 200
// is saved and a ping succeeds. Replaced on save without restarting the API.
var sageDB *gorm.DB

// Sage returns the current Sage 200 handle, or nil if not connected.
func Sage() *gorm.DB {
	sageMu.RLock()
	defer sageMu.RUnlock()
	return sageDB
}

// SageConfigured reports whether host/user/password/name/port are all set.
func SageConfigured(cfg config.DbConfig) bool {
	return strings.TrimSpace(cfg.Host) != "" &&
		strings.TrimSpace(cfg.User) != "" &&
		strings.TrimSpace(cfg.Password) != "" &&
		strings.TrimSpace(cfg.Name) != "" &&
		strings.TrimSpace(cfg.Port) != ""
}

// ApplySageConfig opens Sage 200 with cfg and swaps the live handle.
// Incomplete cfg disconnects (portal stays up). Connect failure disconnects
// the previous handle so operators are not talking to a stale host.
func ApplySageConfig(ctx context.Context, cfg config.DbConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !SageConfigured(cfg) {
		swapSage(nil)
		logs.Info("Sage 200 is not configured — set it under Settings → Sage 200")
		return nil
	}
	gdb, err := openAndPing(ctx, cfg)
	if err != nil {
		swapSage(nil)
		return fmt.Errorf("Sage 200: %w", err)
	}
	swapSage(gdb)
	logs.Info("Sage 200 database connected")
	return nil
}

// PingSage opens a throwaway connection to verify credentials without swapping
// the live handle. Caller must pass a complete cfg.
func PingSage(ctx context.Context, cfg config.DbConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !SageConfigured(cfg) {
		return fmt.Errorf("Sage 200 host, user, password, database and port are required")
	}
	gdb, err := openAndPing(ctx, cfg)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	_ = sqlDB.Close()
	return nil
}

func swapSage(next *gorm.DB) {
	sageMu.Lock()
	old := sageDB
	sageDB = next
	sageMu.Unlock()
	if old == nil {
		return
	}
	sqlDB, err := old.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

// CloseSageDb closes the live Sage connection during shutdown.
func CloseSageDb(ctx context.Context) error {
	sageMu.Lock()
	old := sageDB
	sageDB = nil
	sageMu.Unlock()
	if old == nil {
		logs.Warn("CloseSageDb called but Sage connection is nil")
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sqlDb, err := old.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB for closing: %w", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- sqlDb.Close()
	}()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("error closing Sage database connection: %w", err)
		}
		logs.Info("Sage database connection closed gracefully")
		return nil
	case <-ctx.Done():
		logs.Warn("Timeout while closing Sage database connection")
		return ctx.Err()
	}
}
