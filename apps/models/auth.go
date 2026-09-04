// Package models holds the GORM persistence models for TIPER DFMS.
//
// Conventions:
//   - Numeric primary keys (ID) are used internally for foreign keys.
//   - Public-facing identifiers use ULIDs (UID) so internal counts are not leaked.
//   - The database is Microsoft SQL Server: JSON is stored as nvarchar(max) via
//     a GORM json serializer, booleans use bit defaults (default:1 / default:0),
//     and foreign keys use ON DELETE NO ACTION to avoid MSSQL's
//     "multiple cascade paths" migration errors and cascade surprises on
//     shared reference rows.
package models

import (
	"dfms/pkg/constants"
	"dfms/pkg/types"
	"dfms/utils"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MaxFailedAttempts is the number of consecutive failed logins before an
// account is automatically locked.
const MaxFailedAttempts = 3

// RefreshToken binds a refresh token to a specific user + IP address so that a
// stolen token cannot be replayed from a different network location. The
// composite (Subject, IpAddress) primary key allows one active refresh token
// per IP per user. Subject is User.UID (FK).
type RefreshToken struct {
	ContentType types.ContentType `gorm:"default:1;not null;check:ContentType=1" json:"-"`
	Subject     string            `gorm:"primaryKey;not null;type:varchar(26)" json:"-"`
	User        User              `gorm:"foreignKey:Subject;references:UID;constraint:OnDelete:NO ACTION" json:"-"`
	IpAddress   string            `gorm:"primaryKey;not null;type:varchar(45)" json:"-"`
	ExpiresAt   time.Time         `gorm:"not null;index" json:"-"`
	// CreatedAt is the issue time of this refresh row (login or last rotate).
	CreatedAt time.Time `gorm:"not null" json:"-"`
	// LastSeen slides on Continue and the SPA activity heartbeat. Idle is
	// measured from LastSeen (CreatedAt if LastSeen is zero / unset).
	LastSeen time.Time `gorm:"index:idx_refreshLastSeen" json:"-"`
}

// Profile holds non-authentication user data (display preferences, contact).
type Profile struct {
	ID uint `gorm:"primaryKey" json:"-"`
	// Non-empty PhoneNumber uniqueness: filtered index idx_uniqueProfilePhone (migrate).
	PhoneNumber       string            `gorm:"type:varchar(30)" json:"phoneNumber,omitempty"`
	Title             string            `gorm:"type:varchar(60);index:idx_profileTitle" json:"title"`
	AppearanceSetting map[string]any    `gorm:"serializer:json;type:nvarchar(max)" json:"appearanceSettings"`
	UserID            uint              `gorm:"uniqueIndex:idx_uniqueProfile;not null" json:"-"`
	ContentType       types.ContentType `gorm:"default:2;not null;check:ContentType=2" json:"-"`
	CreatedAt         time.Time         `json:"-"`
	UpdatedAt         time.Time         `json:"-"`
}

// User is an authenticating principal in TIPER DFMS.
type User struct {
	ID                 uint              `gorm:"primaryKey" json:"-"`
	ContentType        types.ContentType `gorm:"default:3;not null;check:ContentType=3" json:"-"`
	Attempts           uint              `gorm:"default:0;not null;check:chk_user_attempts,[Attempts] >= 0" json:"-"`
	UID                string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueUId;not null" json:"id"`
	Email              string            `gorm:"type:varchar(120);not null;uniqueIndex:idx_uniqueUserEmail;check:chk_user_email,[Email] <> ''" json:"email"`
	FirstName          string            `gorm:"type:varchar(60);not null;index:idx_userDetail;check:chk_user_first,[FirstName] <> ''" json:"firstName"`
	LastName           string            `gorm:"type:varchar(60);not null;index:idx_userDetail;check:chk_user_last,[LastName] <> ''" json:"lastName"`
	Password           string            `gorm:"type:varchar(250);not null" json:"-"`
	IsActive           bool              `gorm:"default:1;not null;index:idx_userActive,priority:1" json:"isActive"`
	IsSuperUser        bool              `gorm:"default:0;not null" json:"-"`
	IsLocked           bool              `gorm:"default:0;not null;index:idx_userActive,priority:2" json:"isLocked"`
	MustChangePassword bool              `gorm:"default:0;not null" json:"mustChangePassword"`
	// SessionVersion is stamped into access/refresh tokens. Bumping it (e.g. on
	// password change) invalidates every outstanding session for this user.
	SessionVersion uint       `gorm:"not null;default:0" json:"-"`
	CreatedAt      time.Time  `gorm:"index:idx_userCreatedAt" json:"-"`
	UpdatedAt      time.Time  `json:"-"`
	LastLogin      *time.Time `gorm:"index:idx_userLastLogin" json:"lastLogin,omitempty"`
	DjangoID       uint       `gorm:"index" json:"-"`
	Roles          []Role     `gorm:"many2many:UserRole;constraint:OnDelete:NO ACTION" json:"roles"`
	Profile        Profile    `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:NO ACTION" json:"profile"`
	// CanDelete is computed on list: never signed in, not super-user, not self.
	CanDelete bool `gorm:"-" json:"canDelete"`
}

// BeforeCreate assigns a ULID public identifier.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	u.UID = uid
	return nil
}

// BeforeDelete blocks hard-delete after the user has signed in at least once.
func (u *User) BeforeDelete(tx *gorm.DB) error {
	if u.LastLogin != nil {
		return constants.ErrUserInUse
	}
	if err := tx.Where("UserID = ?", u.ID).Delete(&PasswordHistory{}).Error; err != nil {
		return err
	}
	return nil
}

// FullName returns the concatenated first and last name.
func (u *User) FullName() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

// UpdateLastLogin records a successful login and clears lock/attempt state.
func (u *User) UpdateLastLogin(db *gorm.DB) error {
	updateData := map[string]any{"LastLogin": time.Now(), "IsLocked": false, "Attempts": 0}
	return db.Model(u).Select("LastLogin", "IsLocked", "Attempts").Updates(updateData).Error
}

// EncryptPassword hashes the plaintext password in-place with bcrypt.
func (u *User) EncryptPassword() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = string(hash)
	return nil
}

// ComparePassword verifies a plaintext password against the stored hash.
func (u *User) ComparePassword(password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return constants.ErrInvalidCredentials
	}
	return nil
}

// IncrementFailedAttempts atomically bumps the failed-login counter and locks
// the account once MaxFailedAttempts is reached. The post-update counter is
// re-read from the DB because it is the source of truth under concurrency.
func (u *User) IncrementFailedAttempts(db *gorm.DB) error {
	if err := db.Model(u).Update("Attempts", gorm.Expr("Attempts + 1")).Error; err != nil {
		return fmt.Errorf("increment failed attempts: %w", err)
	}

	var current struct{ Attempts uint }
	if err := db.Model(&User{}).Select("Attempts").Where("ID = ?", u.ID).Take(&current).Error; err != nil {
		return fmt.Errorf("reload attempts: %w", err)
	}
	u.Attempts = current.Attempts

	if u.Attempts >= MaxFailedAttempts && !u.IsLocked {
		if err := db.Model(u).Update("IsLocked", true).Error; err != nil {
			return fmt.Errorf("lock account: %w", err)
		}
		u.IsLocked = true
	}
	return nil
}

// Permission is a globally-scoped capability code in the form
// {module}.{resource}.{action} (e.g. inventory.read, billing.run).
type Permission struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	ContentType types.ContentType `gorm:"default:4;not null;check:ContentType=4" json:"-"`
	Code        string            `gorm:"type:varchar(120);not null;uniqueIndex:idx_uniquePerm;check:chk_perm_code,[Code] <> ''" json:"code"`
	Description string            `gorm:"type:varchar(160);not null" json:"description"`
	Module      string            `gorm:"type:varchar(120);not null;index:idx_permDetails" json:"module"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
}

// Role groups permissions. A user may hold any number of roles and inherits
// the union of their permissions.
type Role struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	UID         string            `gorm:"type:varchar(26);uniqueIndex:idx_uniqueRoleUID;not null" json:"-"`
	ContentType types.ContentType `gorm:"default:5;not null;check:ContentType=5" json:"-"`
	Name        string            `gorm:"type:varchar(120);not null;uniqueIndex:idx_uniqueRoleName;check:chk_role_name,[Name] <> ''" json:"name"`
	Description string            `gorm:"type:varchar(160);not null" json:"description"`
	Category    types.RoleFamily  `gorm:"type:varchar(16);not null;default:system;index:idx_roleCategory" json:"category"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
	Permissions []Permission      `gorm:"many2many:RolePermission;constraint:OnDelete:NO ACTION" json:"permissions,omitempty"`
	Users       []User            `gorm:"many2many:UserRole;constraint:OnDelete:NO ACTION" json:"users,omitempty"`
	// PermissionCount is populated on list/options (not a column). Detail
	// endpoints set it from the preloaded Permissions slice.
	PermissionCount int `gorm:"-" json:"permissionCount"`
	// CanDelete is computed on list/detail: no UserRole and no NodeOperatorRole.
	// Seeded names are not special — unused bootstrap roles may be removed.
	CanDelete bool `gorm:"-" json:"canDelete"`
}

// BeforeCreate assigns a ULID public identifier.
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	r.UID = uid
	return nil
}

// UserRole is the join table between users and roles.
// PK (UserID, RoleID) serves "roles for this user". idx_userRoleByRole
// (RoleID, UserID) serves operator resolution and role-delete checks.
type UserRole struct {
	UserID uint `gorm:"primaryKey;index:idx_userRoleByRole,priority:2" json:"userId"`
	RoleID uint `gorm:"primaryKey;index:idx_userRoleByRole,priority:1" json:"roleId"`
}

// TableName pins the join table to the singular PascalCase "UserRole" used by
// the many2many tags and raw SQL, overriding GORM's snake_case pluralization.
func (UserRole) TableName() string { return "UserRole" }

// RolesPermission is the join table between roles and permissions.
// PK (RoleID, PermissionID) serves "permissions for this role".
// idx_rolePermByPerm (PermissionID, RoleID) serves permission-code lookups.
type RolesPermission struct {
	RoleID       uint `gorm:"primaryKey;index:idx_rolePermByPerm,priority:2" json:"roleId"`
	PermissionID uint `gorm:"primaryKey;index:idx_rolePermByPerm,priority:1" json:"permissionId"`
}

// TableName maps the struct to "RolePermission" — the name declared in the
// Role.Permissions many2many tag — since the struct name (RolesPermission)
// would otherwise derive a different table.
func (RolesPermission) TableName() string { return "RolePermission" }

// Title is a selectable user job title (e.g. "Head of Finance").
type Title struct {
	ID          string            `gorm:"primaryKey;type:varchar(26)" json:"id"`
	ContentType types.ContentType `gorm:"default:6;not null;check:ContentType=6" json:"-"`
	Name        string            `gorm:"type:varchar(60);uniqueIndex:idx_uniqueTitleName;not null;check:chk_title_name,[Name] <> ''" json:"name"`
	CreatedByID uint              `gorm:"index;not null" json:"-"`
	HasData     bool              `gorm:"default:0;not null" json:"hasData"` // sticky: set when a profile uses this title; unlike roles, titles do not re-check live refs on delete
	CreatedBy   User              `gorm:"foreignKey:CreatedByID;constraint:OnDelete:NO ACTION" json:"-"`
	CreatedAt   time.Time         `json:"-"`
	UpdatedAt   time.Time         `json:"-"`
}

// BeforeCreate assigns a ULID identifier.
func (t *Title) BeforeCreate(tx *gorm.DB) error {
	uid, err := utils.GetULID()
	if err != nil {
		return err
	}
	t.ID = uid
	return nil
}

// UpdateHasData marks a title as referenced.
func (t *Title) UpdateHasData(tx *gorm.DB) error {
	if t.HasData {
		return nil
	}
	return tx.Model(t).Update("HasData", true).Error
}

// UserOTPChallenge stores a hashed one-time password issued during 2FA login.
type UserOTPChallenge struct {
	ID          string            `gorm:"type:varchar(26);primaryKey" json:"id"`
	ContentType types.ContentType `gorm:"default:10;not null;check:ContentType=10" json:"-"`
	IpAddress   string            `gorm:"primaryKey;not null;type:varchar(45)" json:"-"`
	UserID      uint              `gorm:"primaryKey;not null" json:"-"`
	User        User              `gorm:"foreignKey:UserID;constraint:OnDelete:NO ACTION" json:"-"`
	CodeHash    string            `gorm:"type:varchar(128);not null" json:"-"`
	Attempts    uint              `gorm:"default:0;not null;check:chk_otp_attempts,[Attempts] >= 0" json:"-"`
	ExpiresAt   time.Time         `gorm:"not null;index" json:"expiresAt"`
	ConsumedAt  *time.Time        `json:"-"`
	CreatedAt   time.Time         `json:"-"`
}
