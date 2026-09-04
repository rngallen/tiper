package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"dfms/pkg/config"
	"dfms/pkg/logs"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const connectAttempts = 3

// connectWithRetry opens and pings an MSSQL database up to connectAttempts
// times with exponential backoff (2s, 4s). Windows SQL Server services often
// accept TCP a few seconds after the process starts; a single ping at boot
// is a false "unavailable". After the last failure the error is fatal.
func connectWithRetry(ctx context.Context, name string, cfg config.DbConfig, dest **gorm.DB) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var last error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		gdb, err := openAndPing(ctx, cfg)
		if err == nil {
			*dest = gdb
			if attempt == 1 {
				logs.Infof("%s (MSSQL) database connected successfully", name)
			} else {
				logs.Infof("%s (MSSQL) database connected successfully (attempt %d/%d)", name, attempt, connectAttempts)
			}
			return nil
		}
		last = err
		if attempt == connectAttempts {
			break
		}
		backoff := time.Duration(1<<uint(attempt-1)) * 2 * time.Second
		logs.Warnf("%s database connect attempt %d/%d failed: %v; retrying in %s",
			name, attempt, connectAttempts, err, backoff)
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s database: %w", name, ctx.Err())
		case <-time.After(backoff):
		}
	}
	return fmt.Errorf("%s database: failed after %d attempts: %w", name, connectAttempts, last)
}

func openAndPing(ctx context.Context, cfg config.DbConfig) (*gorm.DB, error) {
	encodedUser := url.QueryEscape(cfg.User)
	encodedPassword := url.QueryEscape(cfg.Password)
	hostPort := cfg.Host
	if cfg.Port != "" {
		hostPort = cfg.Host + ":" + cfg.Port
	}
	path := ""
	if cfg.Instance != "" {
		path = "/" + url.PathEscape(cfg.Instance)
	}
	encrypt := "disable"
	if cfg.Encrypt {
		encrypt = "true"
	}
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s%s?database=%s&connection+timeout=30&encrypt=%s&trust+server+certificate=true",
		encodedUser,
		encodedPassword,
		hostPort,
		path,
		url.QueryEscape(cfg.Name),
		encrypt,
	)

	gormDb, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{
		DisableAutomaticPing: false,
		QueryFields:          true,
		Logger:               dbLogger(),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
			NoLowerCase:   true,
			NameReplacer:  strings.NewReplacer("", ""),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	sqlDB, err := gormDb.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying *sql.DB from GORM: %w", err)
	}
	sqlDB.SetMaxIdleConns(defaultMaxIdleConns)
	sqlDB.SetMaxOpenConns(defaultMaxOpenConns)
	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)

	if pingErr := sqlDB.PingContext(ctx); pingErr != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping: %w", pingErr)
	}
	return gormDb, nil
}
