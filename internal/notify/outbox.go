package notify

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"dfms/apps/models"
	"dfms/internal/jobs"
	"dfms/pkg/logs"
	"dfms/pkg/sms"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

const (
	outboxMaxAttempts = 12
	outboxMaxAge      = 72 * time.Hour
	outboxBatch       = 25
	outboxSendLease   = 5 * time.Minute
	outboxErrLimit    = 500
)

// NextBackoff is the wait after a failed attempt (1-based). Caps at 12 hours.
func NextBackoff(attempts uint) time.Duration {
	delays := []time.Duration{
		time.Minute,
		2 * time.Minute,
		4 * time.Minute,
		8 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		time.Hour,
		2 * time.Hour,
		4 * time.Hour,
		8 * time.Hour,
		12 * time.Hour,
	}
	if attempts == 0 {
		return 0
	}
	i := int(attempts) - 1
	if i >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[i]
}

// ShouldDeadLetter reports whether retries should stop.
func ShouldDeadLetter(attempts, maxAttempts uint, createdAt, now time.Time) bool {
	if maxAttempts == 0 {
		maxAttempts = outboxMaxAttempts
	}
	if attempts >= maxAttempts {
		return true
	}
	if createdAt.IsZero() {
		return false
	}
	return now.Sub(createdAt) >= outboxMaxAge
}

// Enqueue persists durable messages. OTP codes must never be passed here.
func Enqueue(db *gorm.DB, rows []models.MailOutbox) error {
	if db == nil || len(rows) == 0 {
		return nil
	}
	now := time.Now()
	for i := range rows {
		if rows[i].Status == "" {
			rows[i].Status = types.NotifyPending
		}
		if rows[i].MaxAttempts == 0 {
			rows[i].MaxAttempts = outboxMaxAttempts
		}
		if rows[i].NextAttemptAt.IsZero() {
			rows[i].NextAttemptAt = now
		}
		rows[i].ContentType = types.MailOutboxContent
	}
	return db.Create(&rows).Error
}

// KickDrain starts a background drain so a just-queued row is tried immediately.
func KickDrain(db *gorm.DB) {
	if db == nil {
		return
	}
	logs.GoSafe("notify.outbox.drain", func() {
		if err := Drain(context.Background(), db); err != nil {
			logs.Errorf("notify.outbox: %v", err)
		}
	})
}

// RegisterJobs wires the periodic outbox drain (default: every minute).
func RegisterJobs(m *jobs.Manager, db *gorm.DB) {
	if m == nil || db == nil {
		return
	}
	m.Register(jobs.NotifyOutbox, func() {
		if err := Drain(context.Background(), db); err != nil {
			logs.Errorf("notify.outbox: %v", err)
		}
	})
}

// Drain claims due rows and delivers them. Safe to run concurrently with KickDrain.
func Drain(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	var rows []models.MailOutbox
	err := db.WithContext(ctx).
		Where("Status IN ? AND NextAttemptAt <= ?", []string{types.NotifyPending, types.NotifyFailed}, now).
		Order("NextAttemptAt ASC").
		Limit(outboxBatch).
		Find(&rows).Error
	if err != nil {
		return err
	}
	for i := range rows {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		deliverOne(ctx, db, &rows[i], now)
	}
	return nil
}

func deliverOne(ctx context.Context, db *gorm.DB, row *models.MailOutbox, now time.Time) {
	attempts, ok := claimRow(db, row, now)
	if !ok {
		return
	}
	sendErr := deliverRow(ctx, row)
	updates := map[string]any{
		"Attempts":  attempts,
		"UpdatedAt": now,
	}
	if sendErr == nil {
		updates["Status"] = types.NotifySent
		updates["SentAt"] = now
		updates["LastError"] = ""
		if err := db.Model(row).Updates(updates).Error; err != nil {
			logs.Errorf("notify.outbox mark sent %s: %v", row.ID, err)
		}
		return
	}
	updates["LastError"] = truncateErr(sendErr.Error())
	if ShouldDeadLetter(attempts, row.MaxAttempts, row.CreatedAt, now) {
		updates["Status"] = types.NotifyDead
		logs.Errorf("notify.outbox dead-lettered id=%s kind=%s channel=%s to=%s: %v",
			row.ID, row.Kind, row.Channel, row.Recipient, sendErr)
	} else {
		updates["Status"] = types.NotifyFailed
		updates["NextAttemptAt"] = now.Add(NextBackoff(attempts))
		logs.Warnf("notify.outbox retry id=%s kind=%s attempt=%d: %v", row.ID, row.Kind, attempts, sendErr)
	}
	if err := db.Model(row).Updates(updates).Error; err != nil {
		logs.Errorf("notify.outbox mark failed %s: %v", row.ID, err)
	}
}

func claimRow(db *gorm.DB, row *models.MailOutbox, now time.Time) (attempts uint, ok bool) {
	res := db.Model(&models.MailOutbox{}).
		Where("ID = ? AND Status IN ? AND NextAttemptAt <= ?",
			row.ID, []string{types.NotifyPending, types.NotifyFailed}, now).
		Updates(map[string]any{
			"Attempts":      gorm.Expr("Attempts + 1"),
			"NextAttemptAt": now.Add(outboxSendLease),
			"UpdatedAt":     now,
		})
	if res.Error != nil {
		logs.Errorf("notify.outbox claim %s: %v", row.ID, res.Error)
		return 0, false
	}
	if res.RowsAffected != 1 {
		return 0, false
	}
	return row.Attempts + 1, true
}

func deliverRow(ctx context.Context, row *models.MailOutbox) error {
	switch row.Channel {
	case types.NotifyChannelEmail:
		return sendBranded(ctx, []string{row.Recipient}, row.Subject, row.Body)
	case types.NotifyChannelSMS:
		return sms.Send(ctx, row.Recipient, row.Body)
	default:
		return fmt.Errorf("unknown channel %q", row.Channel)
	}
}

func truncateErr(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= outboxErrLimit {
		return s
	}
	runes := []rune(s)
	return string(runes[:outboxErrLimit-1]) + "…"
}

func persistCredentials(db *gorm.DB, user *models.User, tempPassword string, kind types.CredentialKind) error {
	if user == nil {
		return fmt.Errorf("%s: user required", kind)
	}
	brand := loadBrand(context.Background(), db)
	subject, emailBody, smsBody := credentialMessages(user, tempPassword, brand, kind)
	rows := buildCredentialOutbox(user, subject, emailBody, smsBody, kind)
	if len(rows) == 0 {
		return fmt.Errorf("%s: no recipient", kind)
	}
	return Enqueue(db, rows)
}

func buildCredentialOutbox(user *models.User, subject, emailBody, smsBody string, kind types.CredentialKind) []models.MailOutbox {
	if user == nil {
		return nil
	}
	notifyKind := types.NotifyKindWelcome
	if kind == types.CredentialReset {
		notifyKind = types.NotifyKindReset
	}
	var uid *uint
	if user.ID != 0 {
		id := user.ID
		uid = &id
	}
	var rows []models.MailOutbox
	if email := strings.TrimSpace(user.Email); email != "" {
		rows = append(rows, models.MailOutbox{
			Kind:      notifyKind,
			Channel:   types.NotifyChannelEmail,
			Recipient: email,
			Subject:   subject,
			Body:      emailBody,
			UserID:    uid,
		})
	}
	phone := strings.TrimSpace(user.Profile.PhoneNumber)
	if phone != "" && strings.TrimSpace(smsBody) != "" {
		rows = append(rows, models.MailOutbox{
			Kind:      notifyKind,
			Channel:   types.NotifyChannelSMS,
			Recipient: phone,
			Body:      smsBody,
			UserID:    uid,
		})
	}
	return rows
}

func buildWorkflowOutbox(users []models.User, phones map[uint]string, subject, emailBody, smsBody string) []models.MailOutbox {
	seenEmail := make(map[string]struct{})
	seenPhone := make(map[string]struct{})
	rows := make([]models.MailOutbox, 0, len(users)*2)
	for _, u := range users {
		var uid *uint
		if u.ID != 0 {
			id := u.ID
			uid = &id
		}
		if email := strings.TrimSpace(u.Email); email != "" {
			key := strings.ToLower(email)
			if _, ok := seenEmail[key]; !ok {
				seenEmail[key] = struct{}{}
				rows = append(rows, models.MailOutbox{
					Kind:      types.NotifyKindWorkflow,
					Channel:   types.NotifyChannelEmail,
					Recipient: email,
					Subject:   subject,
					Body:      emailBody,
					UserID:    uid,
				})
			}
		}
		phone := strings.TrimSpace(u.Profile.PhoneNumber)
		if phone == "" && phones != nil {
			phone = strings.TrimSpace(phones[u.ID])
		}
		if phone != "" && strings.TrimSpace(smsBody) != "" {
			if _, ok := seenPhone[phone]; !ok {
				seenPhone[phone] = struct{}{}
				rows = append(rows, models.MailOutbox{
					Kind:      types.NotifyKindWorkflow,
					Channel:   types.NotifyChannelSMS,
					Recipient: phone,
					Body:      smsBody,
					UserID:    uid,
				})
			}
		}
	}
	return rows
}

func enqueueOrFallback(db *gorm.DB, rows []models.MailOutbox, fallback func()) {
	if len(rows) == 0 {
		return
	}
	if db == nil {
		if fallback != nil {
			logs.GoSafe("notify.outbox.fallback", fallback)
		}
		return
	}
	if err := Enqueue(db, rows); err != nil {
		logs.Errorf("notify.outbox enqueue: %v", err)
		if fallback != nil {
			logs.GoSafe("notify.outbox.fallback", fallback)
		}
		return
	}
	KickDrain(db)
}
