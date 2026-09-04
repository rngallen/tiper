// Package config provides a thread-safe singleton configuration loader using viper,
// with automatic .env file loading, environment variable override support,
// and recursive whitespace trimming for all string fields.
//
// The package ensures that configuration is initialized only once,
// even in concurrent environments, and provides a global Config instance via Conf.
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// MaxConfigDepth caps recursion in trimStrings to defend against pathological
// or self-referential structures (in practice the config has no cycles, but the
// trimmer is generic).
const maxConfigDepth = 32

var (
	// Conf holds the global application configuration after successful initialization.
	Conf Config

	// once ensures InitConfig() runs exactly once on success. On error we reset
	// it via initOnce.Reset (see below) so the caller can retry after fixing
	// the underlying problem.
	once sync.Once

	// initMu guards re-initialization after a failed run.
	initMu sync.Mutex
)

// InitConfig initializes the global configuration (Conf) using the singleton pattern.
//
// Responsibilities:
//   - Loads .env from ProgramData/DFMS (Windows), /etc/dfms, ./config, or cwd
//   - Merges secrets.env from the platform config directory (keys, DB password)
//   - Supports DFMS_ prefixed environment variable overrides (highest precedence)
//   - Unmarshals into Conf struct
//   - Trims whitespace from all string fields
//   - Validates the entire configuration using struct tags
//
// If the .env file is not found but the required environment variables are set,
// the loader will still succeed.
//
// This function is safe to call concurrently — it will only run once successfully.
func InitConfig(ctx context.Context) error {
	initMu.Lock()
	defer initMu.Unlock()

	var initErr error
	once.Do(func() {
		initErr = loadConfig(ctx)
	})

	if initErr != nil {
		// Reset the once so that a subsequent call (after fixing config) can retry.
		once = sync.Once{}
		return initErr
	}
	return nil
}

func loadConfig(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	vp := viper.New()
	vp.SetConfigType("env")

	// Non-secret settings first, then ACL-protected secrets.env. OS env wins last.
	base := findConfigFile(".env", configSearchDirs())
	if err := readEnvFile(vp, base); err != nil {
		return err
	}
	secrets := findConfigFile("secrets.env", secretsSearchDirs())
	if secrets != "" && secrets != base {
		if err := mergeEnvFile(vp, secrets); err != nil {
			return err
		}
	}

	// DFMS_ not APP_: another process on the same host can use its own prefix
	// (ABC_LISTEN_ADDRESS) without colliding. BindEnv maps DFMS_SYMMETRIC_KEY
	// onto dfms.symmetric_key — AutomaticEnv alone would look for
	// DFMS_DFMS_SYMMETRIC_KEY.
	vp.SetEnvPrefix("DFMS")
	vp.AutomaticEnv()
	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	bindBootstrapEnv(vp)

	// ===================== Unmarshal =====================
	if err := vp.Unmarshal(&Conf); err != nil {
		return fmt.Errorf("config: unmarshal: %w", err)
	}

	// ===================== Trim all string fields =====================
	if err := trimStrings(&Conf); err != nil {
		return fmt.Errorf("config: trim whitespace: %w", err)
	}

	// ===================== Validate =====================
	if err := validateConfig(ctx, &Conf); err != nil {
		return fmt.Errorf("config: validation failed: %w", err)
	}

	return nil
}

func bindBootstrapEnv(vp *viper.Viper) {
	for _, p := range [][2]string{
		{"dfms.symmetric_key", "DFMS_SYMMETRIC_KEY"},
		{"dfms.refresh_key", "DFMS_REFRESH_KEY"},
		{"dfms.mfa_key", "DFMS_MFA_KEY"},
		{"dfms.db.password", "DFMS_DB_PASSWORD"},
		{"dfms.db.user", "DFMS_DB_USER"},
		{"dfms.db.host", "DFMS_DB_HOST"},
		{"dfms.db.name", "DFMS_DB_NAME"},
		{"dfms.db.port", "DFMS_DB_PORT"},
		{"dfms.db.encrypt", "DFMS_DB_ENCRYPT"},
		{"dfms.listen_address", "DFMS_LISTEN_ADDRESS"},
		{"dfms.debug", "DFMS_DEBUG"},
		{"dfms.allow_insecure_http", "DFMS_ALLOW_INSECURE_HTTP"},
		{"dfms.trust_forwarded_for", "DFMS_TRUST_FORWARDED_FOR"},
		{"dfms.shutdown_timeout", "DFMS_SHUTDOWN_TIMEOUT"},
	} {
		_ = vp.BindEnv(p[0], p[1])
	}
}

func readEnvFile(vp *viper.Viper, path string) error {
	if path == "" {
		return nil
	}
	vp.SetConfigFile(path)
	if err := vp.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok || os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	return nil
}

func mergeEnvFile(vp *viper.Viper, path string) error {
	vp.SetConfigFile(path)
	if err := vp.MergeInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok || os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: merge %s: %w", path, err)
	}
	return nil
}

// PlatformConfigDir is the machine-wide DFMS config directory:
// C:\ProgramData\DFMS on Windows, /etc/dfms elsewhere.
func PlatformConfigDir() string {
	if runtime.GOOS == "windows" {
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = `C:\ProgramData`
		}
		return filepath.Join(pd, "DFMS")
	}
	return "/etc/dfms"
}

func configSearchDirs() []string {
	dirs := make([]string, 0, 4)
	if p := strings.TrimSpace(os.Getenv("DFMS_CONFIG_DIR")); p != "" {
		dirs = append(dirs, p)
	}
	dirs = append(dirs, PlatformConfigDir(), ".", "config")
	return dirs
}

// secretsSearchDirs is ProgramData / /etc/dfms (and DFMS_CONFIG_DIR). Secrets
// are not loaded from the install directory on purpose.
func secretsSearchDirs() []string {
	dirs := make([]string, 0, 2)
	if p := strings.TrimSpace(os.Getenv("DFMS_CONFIG_DIR")); p != "" {
		dirs = append(dirs, p)
	}
	return append(dirs, PlatformConfigDir())
}

// findConfigFile returns the first existing path matching name in the given
// search directories. Returns "" when no candidate exists.
func findConfigFile(name string, dirs []string) string {
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		var candidate string
		if dir == "." {
			candidate = name
		} else {
			candidate = filepath.Join(dir, name)
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// trimStrings recursively trims whitespace from all string fields.
func trimStrings(config any) error {
	visited := make(map[uintptr]bool)
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("trimStrings: expected non-nil pointer, got %T", config)
	}
	return trimValue(v.Elem(), "", visited, 0)
}

// trimValue processes a single reflect.Value recursively based on its kind.
// The path parameter is used for meaningful error messages, and depth guards
// against runaway recursion.
func trimValue(v reflect.Value, path string, visited map[uintptr]bool, depth int) error {
	if depth > maxConfigDepth {
		return fmt.Errorf("trimStrings: max recursion depth (%d) exceeded at %q", maxConfigDepth, path)
	}

	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			trimmed := strings.TrimSpace(v.String())
			if trimmed != v.String() {
				v.SetString(trimmed)
			}
		}

	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() || !field.CanInterface() {
				continue
			}

			fieldName := t.Field(i).Name
			fieldPath := fieldName
			if path != "" {
				fieldPath = path + "." + fieldName
			}

			if err := trimValue(field, fieldPath, visited, depth+1); err != nil {
				return err
			}
		}

	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		ptr := v.Pointer()
		if visited[ptr] {
			return nil
		}
		visited[ptr] = true
		return trimValue(v.Elem(), path, visited, depth+1)

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := trimValue(v.Index(i), fmt.Sprintf("%s[%d]", path, i), visited, depth+1); err != nil {
				return err
			}
		}

	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		// Map values are not addressable; if they are strings, replace them via SetMapIndex.
		for _, key := range v.MapKeys() {
			val := v.MapIndex(key)
			if val.Kind() == reflect.String {
				trimmed := strings.TrimSpace(val.String())
				if trimmed != val.String() {
					v.SetMapIndex(key, reflect.ValueOf(trimmed))
				}
				continue
			}
			if err := trimValue(val, fmt.Sprintf("%s[%v]", path, key.Interface()), visited, depth+1); err != nil {
				return err
			}
		}

	case reflect.Interface:
		if !v.IsNil() {
			return trimValue(v.Elem(), path, visited, depth+1)
		}
	}

	return nil
}
