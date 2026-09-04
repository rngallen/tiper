package models

import (
	"dfms/pkg/types"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PasswordHistoryKeep is how many previous hashes are retained per user.
// Change-password rejects a new password that matches the current hash or
// any of these retained hashes.
const PasswordHistoryKeep = 5

// PasswordHistory stores a prior bcrypt hash so operators cannot cycle back
// through recent passwords. Rows are pruned to PasswordHistoryKeep per user.
type PasswordHistory struct {
	ID           uint              `gorm:"primaryKey" json:"-"`
	ContentType  types.ContentType `gorm:"default:23;not null;check:ContentType=23" json:"-"`
	UserID       uint              `gorm:"index:idx_pwdHistoryUser,priority:1;not null" json:"-"`
	User         User              `gorm:"foreignKey:UserID;constraint:OnDelete:NO ACTION" json:"-"`
	PasswordHash string            `gorm:"type:varchar(250);not null;check:chk_pwd_history_hash,[PasswordHash] <> ''" json:"-"`
	CreatedAt    time.Time         `gorm:"index:idx_pwdHistoryUser,priority:2" json:"-"`
}

// PasswordReused reports whether plaintext matches the current hash or any of
// the last PasswordHistoryKeep stored hashes for userID.
func PasswordReused(tx *gorm.DB, userID uint, plaintext, currentHash string) (bool, error) {
	if currentHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(plaintext)); err == nil {
			return true, nil
		}
	}
	var rows []PasswordHistory
	if err := tx.Where("UserID = ?", userID).
		Order("CreatedAt DESC").
		Limit(PasswordHistoryKeep).
		Find(&rows).Error; err != nil {
		return false, err
	}
	for _, r := range rows {
		if err := bcrypt.CompareHashAndPassword([]byte(r.PasswordHash), []byte(plaintext)); err == nil {
			return true, nil
		}
	}
	return false, nil
}

// RecordPasswordHash appends a hash and deletes older rows beyond PasswordHistoryKeep.
func RecordPasswordHash(tx *gorm.DB, userID uint, hash string) error {
	if hash == "" {
		return nil
	}
	row := PasswordHistory{UserID: userID, PasswordHash: hash, ContentType: types.PasswordHistoryContent}
	if err := tx.Create(&row).Error; err != nil {
		return err
	}

	subQuery := tx.Model(&PasswordHistory{}).
		Select("ID").
		Where("UserID = ?", userID).
		Order("CreatedAt DESC").
		Limit(PasswordHistoryKeep)

	return tx.Where("UserID = ? AND ID NOT IN (?)", userID, subQuery).
		Delete(&PasswordHistory{}).Error
}
