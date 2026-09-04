// Package audit provides a synchronous, append-only audit-trail recorder.
//
// Every meaningful system action (login, CRUD, approval decision,
// cron run) should be recorded so there is a tamper-evident history
// of who did what. Writes are synchronous and best-effort: a failure to record
// must never block or fail the underlying business operation, so callers
// typically ignore the returned error (it is logged internally).
package audit

import (
	"context"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// Default is the process-wide recorder, set during application startup.
var Default *Recorder

// Recorder writes audit entries to the application database.
type Recorder struct {
	db *gorm.DB
}

// NewRecorder constructs a Recorder bound to the given database handle.
func NewRecorder(db *gorm.DB) *Recorder {
	return &Recorder{db: db}
}

// Entry is the data captured for a single audit record.
type Entry struct {
	UserID      *uint
	UserName    string
	IPAddress   string
	UserAgent   string
	RequestID   string
	Module      types.Module
	Action      types.Action
	RecordType  types.ContentType
	RecordID    string
	Description string
	Changes     map[string]any
	Metadata    map[string]any
}

// Record persists an audit entry. If tx is non-nil the write joins that
// transaction; otherwise the recorder's own handle is used. Errors are logged
// and returned but should generally not abort the caller's operation.
func (r *Recorder) Record(ctx context.Context, tx *gorm.DB, e Entry) error {
	if r == nil {
		return nil
	}
	db := r.db
	if tx != nil {
		db = tx
	}
	if db == nil {
		return nil
	}

	trail := models.AuditTrail{
		UserID:      e.UserID,
		UserName:    e.UserName,
		IPAddress:   e.IPAddress,
		UserAgent:   e.UserAgent,
		RequestID:   e.RequestID,
		Module:      e.Module,
		Action:      e.Action,
		RecordType:  e.RecordType,
		RecordID:    e.RecordID,
		Description: e.Description,
		Changes:     e.Changes,
		Metadata:    e.Metadata,
	}
	if err := db.WithContext(ctx).Create(&trail).Error; err != nil {
		logs.Errorf("audit: record %s/%s: %v", e.Module, e.Action, err)
		return err
	}
	return nil
}

// EntryFromRequest is a convenience builder for the common HTTP case.
func EntryFromRequest(userID uint, module types.Module, userName string, action types.Action, ip, userAgent, requestID string, recordID string, recordType types.ContentType, description string) Entry {
	var uid *uint
	if userID != 0 {
		uid = &userID
	}
	return Entry{
		UserID:      uid,
		IPAddress:   ip,
		UserAgent:   userAgent,
		UserName:    userName,
		Module:      module,
		Action:      action,
		RecordID:    recordID,
		RecordType:  recordType,
		Description: description,
	}
}
