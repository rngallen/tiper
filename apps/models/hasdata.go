package models

import "gorm.io/gorm"

// MarkHasData sticks HasData once a master row is used on a document so
// unused-only delete stays honest without waiting for the next delete attempt.
func MarkHasData(tx *gorm.DB, model any, id uint) {
	if tx == nil || id == 0 {
		return
	}
	_ = tx.Model(model).Where("ID = ? AND HasData = ?", id, false).Update("HasData", true).Error
}

func markHasDataPtr(tx *gorm.DB, model any, id *uint) {
	if id == nil {
		return
	}
	MarkHasData(tx, model, *id)
}
