package models

import (
	"strings"

	"gorm.io/gorm"
)

// CreatedByRef is the public creator shown on approval documents.
// The User row is never serialised (password, roles, session version).
type CreatedByRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// StampCreator builds a list/detail creator from a preloaded User.
func StampCreator(u *User) *CreatedByRef {
	if u == nil || u.ID == 0 {
		return nil
	}
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	if name == "" {
		name = u.Email
	}
	return &CreatedByRef{ID: u.UID, Name: name, Email: u.Email}
}

// PreloadCreatedBy loads only the columns needed for StampCreator.
func PreloadCreatedBy(db *gorm.DB) *gorm.DB {
	return db.Preload("CreatedBy", func(q *gorm.DB) *gorm.DB {
		return q.Select("ID", "UID", "Email", "FirstName", "LastName")
	})
}
