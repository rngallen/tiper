package middleware

import (
	"context"
	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"gorm.io/gorm"
)

// permissionEntry holds the actual data + expiration time
type permissionEntry struct {
	Permissions []string
	ExpiresAt   time.Time
}

type sessionVersionEntry struct {
	Version   uint
	ExpiresAt time.Time
}

var (
	PermissionCache     *ristretto.Cache[string, permissionEntry]
	SessionVersionCache *ristretto.Cache[string, sessionVersionEntry]
)

// SessionVersionTTL bounds how long a cached SessionVersion may be used
// without re-reading the DB. Writes (password change / reset) always
// SetSessionVersion immediately so revocation does not wait on TTL.
const SessionVersionTTL = 15 * time.Minute

// InitPermissionCache initializes Ristretto caches for permissions and
// session versions. Safe to call once at process startup.
func InitPermissionCache() error {
	perms, err := ristretto.NewCache(&ristretto.Config[string, permissionEntry]{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	if err != nil {
		return fmt.Errorf("failed to create permission cache: %w", err)
	}
	sv, err := ristretto.NewCache(&ristretto.Config[string, sessionVersionEntry]{
		NumCounters: 1e6,
		MaxCost:     1 << 20, // ~1MB — only stores a uint per user
		BufferItems: 64,
	})
	if err != nil {
		perms.Close()
		return fmt.Errorf("failed to create session-version cache: %w", err)
	}
	PermissionCache = perms
	SessionVersionCache = sv
	return nil
}

// SetUserPermissions stores permissions with a cost (you can adjust cost based on size)
func SetUserPermissions(userID string, permissions []string, ttl time.Duration) bool {
	entry := permissionEntry{
		Permissions: permissions,
		ExpiresAt:   time.Now().Add(ttl),
	}
	return PermissionCache.Set(userID, entry, 1)
}

// GetUserPermissions retrieves permissions from cache
func GetUserPermissions(userID string) ([]string, bool) {
	entry, found := PermissionCache.Get(userID)
	if !found {
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		PermissionCache.Del(userID) // Remove expired entry
		return nil, false
	}

	return entry.Permissions, true
}

// InvalidateUserPermissions removes a user's permissions from cache
func InvalidateUserPermissions(userID string) {
	if PermissionCache == nil || userID == "" {
		return
	}
	PermissionCache.Del(userID)
}

// InvalidatePermissionsForRole drops cached permission sets for every user
// assigned to the role. Call after RolePermission changes so new grants
// (e.g. reports.read) take effect without waiting for PermsTTL.
func InvalidatePermissionsForRole(ctx context.Context, gdb *gorm.DB, roleID uint) {
	if gdb == nil || roleID == 0 {
		return
	}
	var role models.Role
	if err := gdb.WithContext(ctx).Select("ID").Preload("Users").First(&role, roleID).Error; err != nil {
		logs.Errorf("invalidate permissions for role %d: %v", roleID, err)
		return
	}
	for _, u := range role.Users {
		InvalidateUserPermissions(u.UID)
	}
}

// SetSessionVersion caches the authoritative SessionVersion after a DB read
// or after a bump (password change / reset).
func SetSessionVersion(userUID string, version uint, ttl time.Duration) bool {
	if SessionVersionCache == nil || userUID == "" {
		return false
	}
	ok := SessionVersionCache.Set(userUID, sessionVersionEntry{
		Version:   version,
		ExpiresAt: time.Now().Add(ttl),
	}, 1)
	SessionVersionCache.Wait()
	return ok
}

// GetSessionVersion returns a cached SessionVersion when present and unexpired.
func GetSessionVersion(userUID string) (uint, bool) {
	if SessionVersionCache == nil || userUID == "" {
		return 0, false
	}
	entry, found := SessionVersionCache.Get(userUID)
	if !found {
		return 0, false
	}
	if time.Now().After(entry.ExpiresAt) {
		SessionVersionCache.Del(userUID)
		return 0, false
	}
	return entry.Version, true
}

// Properly close cache on shutdown
func ClosePermissionCache() {
	if PermissionCache != nil {
		PermissionCache.Close()
	}
	if SessionVersionCache != nil {
		SessionVersionCache.Close()
	}
}

func loadPermissionsFromDb(ctx context.Context, userId uint) []string {
	var user models.User
	var held []string
	err := db.Db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "id = ?", userId).Error
	if err != nil {
		logs.Errorf("permission middleware %v", err)
		return held
	}
	if !user.IsActive || user.IsLocked {
		logs.Error("account is not active")
		return held
	}

	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			held = append(held, perm.Code)
		}
	}

	held = uniqueStrings(held)

	return held
}

func loadSessionVersionFromDb(ctx context.Context, userID uint) (uint, error) {
	var user models.User
	err := db.Db.WithContext(ctx).
		Select("SessionVersion").
		Where("ID = ?", userID).
		First(&user).Error
	return user.SessionVersion, err
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, val := range input {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{} // mark as seen
			result = append(result, val)
		}
	}
	return result
}
