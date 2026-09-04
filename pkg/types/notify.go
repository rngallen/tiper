package types

// EmailBadge is the colour chip on branded mail (not an RFID gantry badge).
type EmailBadge string

const (
	EmailBadgeAction  EmailBadge = "action"
	EmailBadgeSuccess EmailBadge = "success"
	EmailBadgeWarning EmailBadge = "warning"
	EmailBadgeInfo    EmailBadge = "info"
)

// CredentialKind is a temporary-password mail (welcome vs admin reset).
type CredentialKind string

const (
	CredentialWelcome CredentialKind = "welcome"
	CredentialReset   CredentialKind = "reset"
)

// Durable outbox kinds, channels, and row status. OTP codes are never queued.
const (
	NotifyKindWelcome  = "welcome"
	NotifyKindReset    = "reset"
	NotifyKindWorkflow = "workflow"

	NotifyChannelEmail = "email"
	NotifyChannelSMS   = "sms"

	NotifyPending = "pending"
	NotifySent    = "sent"
	NotifyFailed  = "failed"
	NotifyDead    = "dead"
)
