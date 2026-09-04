// Package notify adapts mail/SMS to OTP delivery and workflow email notifications.
package notify

import (
	"context"
	"fmt"
	"strings"

	"dfms/apps/models"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/logs"
	"dfms/pkg/portalurl"
	"dfms/pkg/sms"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// WorkflowNotifier emails (and SMS) assignees when new workflow tasks are generated.
// Delivery runs in the background so it never blocks the approval transaction.
type WorkflowNotifier struct {
	db *gorm.DB
}

// NewWorkflowNotifier constructs a WorkflowNotifier. db may be nil (email/SMS
// still send using instance.Summary only).
func NewWorkflowNotifier(db *gorm.DB) *WorkflowNotifier {
	return &WorkflowNotifier{db: db}
}

// NotifyTasks implements workflow.Notifier for the users who own the task.
func (n *WorkflowNotifier) NotifyTasks(instance *models.ProcessInstance, node *models.Node, users []models.User) {
	n.dispatchApprovalMail(instance, node, users, nil, "")
}

// NotifySubstituteTasks implements workflow.Notifier for a delegate covering
// a principal. The email/SMS state that this is substitute work.
func (n *WorkflowNotifier) NotifySubstituteTasks(instance *models.ProcessInstance, node *models.Node, delegate, principal models.User) {
	p := principal
	n.dispatchApprovalMail(instance, node, []models.User{delegate}, &p, "")
}

// NotifyInfo implements workflow.Notifier for FYI / outcome mail (no inbox task).
func (n *WorkflowNotifier) NotifyInfo(instance *models.ProcessInstance, node *models.Node, users []models.User, event types.NotifyEvent) {
	n.dispatchApprovalMail(instance, node, users, nil, event)
}

func (n *WorkflowNotifier) dispatchApprovalMail(instance *models.ProcessInstance, node *models.Node, users []models.User, principal *models.User, event types.NotifyEvent) {
	if instance == nil || node == nil || len(users) == 0 {
		return
	}

	brand := EmailBrand{}
	if n != nil && n.db != nil {
		brand = loadBrand(context.Background(), n.db)
	}
	subject, emailBody, smsBody := n.approvalMessages(instance, node, brand, principal, event)

	userIDs := make([]uint, 0, len(users))
	for _, u := range users {
		if u.ID != 0 {
			userIDs = append(userIDs, u.ID)
		}
	}
	var db *gorm.DB
	if n != nil {
		db = n.db
	}
	phones := phonesFor(db, userIDs)
	rows := buildWorkflowOutbox(users, phones, subject, emailBody, smsBody)
	if len(rows) == 0 {
		return
	}
	enqueueOrFallback(db, rows, func() {
		ctx := context.Background()
		for _, row := range rows {
			if err := deliverRow(ctx, &row); err != nil {
				logs.Errorf("workflow notify %s to %s failed: %v", row.Channel, row.Recipient, err)
			}
		}
	})
}

type mailIntent struct {
	subjectPrefix string
	badge         string
	kind          types.EmailBadge
	introVerb     string
}

func notifyIntentFor(instance *models.ProcessInstance, node *models.Node, event types.NotifyEvent) mailIntent {
	switch event {
	case types.NotifySubmit:
		return mailIntent{
			subjectPrefix: "Submitted",
			badge:         "Submitted",
			kind:          types.EmailBadgeInfo,
			introVerb:     "has been submitted for approval",
		}
	case types.NotifyComplete:
		return mailIntent{
			subjectPrefix: "Approved",
			badge:         "Approved",
			kind:          types.EmailBadgeInfo,
			introVerb:     "has been approved",
		}
	case types.NotifyReject:
		return mailIntent{
			subjectPrefix: "Rejected",
			badge:         "Rejected",
			kind:          types.EmailBadgeAction,
			introVerb:     "has been rejected",
		}
	}
	if instance != nil && (instance.Status == types.NodeRejected || (node != nil && node.Status == types.NodeRejected)) {
		return mailIntent{
			subjectPrefix: "Rejected",
			badge:         "Rejected",
			kind:          types.EmailBadgeAction,
			introVerb:     "has been rejected",
		}
	}
	if instance != nil && (instance.Status == types.NodeCompleted || (node != nil && node.Status == types.NodeCompleted)) {
		return mailIntent{
			subjectPrefix: "Approved",
			badge:         "Approved",
			kind:          types.EmailBadgeInfo,
			introVerb:     "has been approved",
		}
	}
	if node != nil && node.Status == types.NodeDraft {
		return mailIntent{
			subjectPrefix: "Returned",
			badge:         "Returned",
			kind:          types.EmailBadgeWarning,
			introVerb:     "has been returned for correction",
		}
	}
	return mailIntent{
		subjectPrefix: "Action required",
		badge:         "Action required",
		kind:          types.EmailBadgeAction,
		introVerb:     "is awaiting your action",
	}
}

func infoNext(event types.NotifyEvent, fallback string) string {
	switch event {
	case types.NotifySubmit:
		return "This message is for information only. Approvers have been notified."
	case types.NotifyComplete:
		return "The request is fully approved. No further action is required."
	case types.NotifyReject:
		return "The request was terminated. No further action is required."
	default:
		return fallback
	}
}

func infoSecurity(event types.NotifyEvent, fallback string) string {
	if event != "" {
		return "You are receiving this because you are listed for workflow notifications."
	}
	return fallback
}

func infoCTA(event types.NotifyEvent, fallback string) string {
	if event != "" {
		return "Open DFMS"
	}
	return fallback
}

func phonesFor(db *gorm.DB, userIDs []uint) map[uint]string {
	out := make(map[uint]string)
	if db == nil || len(userIDs) == 0 {
		return out
	}
	type phoneRow struct {
		UserID      uint
		PhoneNumber string
	}
	var rows []phoneRow
	_ = db.Model(&models.Profile{}).
		Select("UserID", "PhoneNumber").
		Where("UserID IN ? AND PhoneNumber <> '' AND PhoneNumber IS NOT NULL", userIDs).
		Find(&rows)
	for _, r := range rows {
		if p := strings.TrimSpace(r.PhoneNumber); p != "" {
			out[r.UserID] = p
		}
	}
	return out
}

func coveringName(u *models.User) string {
	if u == nil {
		return ""
	}
	if n := strings.TrimSpace(u.FullName()); n != "" {
		return n
	}
	return strings.TrimSpace(u.Email)
}

func overlaySubstitute(principal *models.User, intent *mailIntent, intro, next, sms *string) {
	name := coveringName(principal)
	if name == "" || intent == nil {
		return
	}
	intent.badge = "Acting as substitute"
	intent.kind = types.EmailBadgeInfo
	intent.subjectPrefix = "Substitute · " + intent.subjectPrefix
	if intro != nil {
		*intro = fmt.Sprintf("You are covering for <strong>%s</strong>. %s", htmlEscape(name), *intro)
	}
	if next != nil && strings.TrimSpace(*next) != "" {
		*next = fmt.Sprintf("Your decision is recorded as acting for %s. %s", name, *next)
	}
	if sms != nil {
		*sms = truncateSMS("Substitute for " + name + ". " + *sms)
	}
}

func documentApprovalMessages(instance *models.ProcessInstance, node *models.Node, brand EmailBrand, principal *models.User, event types.NotifyEvent) (subject, emailBody, smsBody string) {
	return documentApprovalMessagesWithFacts(instance, node, brand, principal, event, nil)
}

func (n *WorkflowNotifier) approvalMessages(instance *models.ProcessInstance, node *models.Node, brand EmailBrand, principal *models.User, event types.NotifyEvent) (subject, emailBody, smsBody string) {
	var facts *wfengine.DocumentFacts
	if n != nil && n.db != nil && instance != nil {
		facts = wfengine.LoadDocumentFacts(n.db, instance.DocContentType, instance.ObjectID)
	}
	return documentApprovalMessagesWithFacts(instance, node, brand, principal, event, facts)
}

func documentApprovalMessagesWithFacts(instance *models.ProcessInstance, node *models.Node, brand EmailBrand, principal *models.User, event types.NotifyEvent, facts *wfengine.DocumentFacts) (subject, emailBody, smsBody string) {
	step := strings.TrimSpace(node.Name)
	intent := notifyIntentFor(instance, node, event)
	kind := types.ContentTypeLabel(instance.DocContentType)
	ref := readableDocumentRef(instance, facts, kind)
	ctaURL := brand.link(types.ApprovalInboxPath(instance.DocContentType))
	intro := fmt.Sprintf("%s <strong>%s</strong> %s at the <strong>%s</strong> step.", htmlEscape(kind), htmlEscape(ref), intent.introVerb, htmlEscape(step))
	next := infoNext(event, "Open Approvals to review this request.")
	security := infoSecurity(event, "If you did not expect this message, contact your system administrator.")
	cta := infoCTA(event, "Review document")
	smsBody = truncateSMS(fmt.Sprintf("%s: %s %s %s at %s. Open DFMS.", brand.smsName(), intent.subjectPrefix, kind, ref, step))
	if event == "" {
		overlaySubstitute(principal, &intent, &intro, &next, &smsBody)
	}
	subject = fmt.Sprintf("%s: %s · %s", intent.subjectPrefix, kind, ref)
	rows := mailDocumentRows(kind, ref, step, instance, facts)
	if event == "" {
		if n := coveringName(principal); n != "" {
			rows = append(rows, kv{Label: "Acting for", Value: n})
		}
	}
	emailBody = renderCorporateEmail(brand, intent.badge, intent.kind,
		"Hello,", intro, "Document summary", rows, next, security, cta, ctaURL,
	)
	return subject, emailBody, smsBody
}

func readableDocumentRef(instance *models.ProcessInstance, facts *wfengine.DocumentFacts, kind string) string {
	candidates := make([]string, 0, 4)
	if facts != nil {
		candidates = append(candidates, facts.DocumentNumber)
	}
	if instance != nil {
		candidates = append(candidates, instance.No, instance.Summary)
	}
	for _, c := range candidates {
		if s := strings.TrimSpace(c); s != "" && !looksLikeULID(s) {
			return s
		}
	}
	if facts != nil {
		pair := strings.TrimSpace(facts.FromCurrency + "/" + facts.ToCurrency)
		if pair != "/" {
			return pair
		}
	}
	if kind != "" {
		return kind
	}
	return "Document"
}

func looksLikeULID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 26 {
		return false
	}
	for _, r := range strings.ToUpper(s) {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			return false
		}
		switch r {
		case 'I', 'L', 'O', 'U':
			return false
		}
	}
	return true
}

func mailDocumentRows(kind, ref, step string, instance *models.ProcessInstance, facts *wfengine.DocumentFacts) []kv {
	rows := []kv{
		{Label: "Document", Value: ref},
		{Label: "Type", Value: kind},
	}
	add := func(label, value string) {
		if s := strings.TrimSpace(value); s != "" && !looksLikeULID(s) {
			rows = append(rows, kv{Label: label, Value: s})
		}
	}
	if facts != nil {
		add("Customer", facts.CustomerName)
		add("Customer code", facts.CustomerCode)
		add("Product", facts.Product)
		add("Quantity", facts.Quantity)
		add("Vessel", facts.Vessel)
		add("Batch", facts.BatchNumber)
		if facts.FromCurrency != "" && facts.ToCurrency != "" {
			add("Pair", facts.FromCurrency+"/"+facts.ToCurrency)
		}
		add("Rate", facts.Rate)
		add("Effective from", facts.EffectiveFrom)
		if facts.Amount != "" {
			add("Amount", strings.TrimSpace(facts.CurrencyCode+" "+facts.Amount))
		} else {
			add("Currency", facts.CurrencyCode)
		}
		add("Description", facts.Description)
	} else if instance != nil {
		add("Description", instance.Summary)
	}
	add("Step", step)
	return rows
}

func dash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func truncateSMS(s string) string {
	const max = 160
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// htmlEscape performs minimal escaping for the small set of fields we embed.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

// SendWelcomeCredentialsAsync queues a welcome email (and SMS when a phone is
// on the profile) with the temporary password. The outbox retries delivery.
func SendWelcomeCredentialsAsync(db *gorm.DB, user *models.User, tempPassword string) {
	queueCredentials(db, user, tempPassword, types.CredentialWelcome)
}

// SendPasswordResetCredentialsAsync queues a reset email (and SMS when a phone
// is on the profile) with a new temporary password.
func SendPasswordResetCredentialsAsync(db *gorm.DB, user *models.User, tempPassword string) {
	queueCredentials(db, user, tempPassword, types.CredentialReset)
}

func queueCredentials(db *gorm.DB, user *models.User, tempPassword string, kind types.CredentialKind) {
	if user == nil {
		return
	}
	if db == nil {
		u := *user
		pwd := tempPassword
		logs.GoSafe("user."+string(kind)+".credentials", func() {
			deliverTempPassword(context.Background(), nil, &u, pwd, kind)
		})
		return
	}
	if err := persistCredentials(db, user, tempPassword, kind); err != nil {
		logs.Errorf("%s enqueue for %s: %v", kind, user.Email, err)
		logs.Warnf("%s enqueue failed for %s — temporary password (relay manually): %s", kind, user.Email, tempPassword)
		return
	}
	KickDrain(db)
}

func credentialMessages(user *models.User, tempPassword string, brand EmailBrand, kind types.CredentialKind) (subject, emailBody, smsBody string) {
	if user == nil {
		return "", "", ""
	}
	name := strings.TrimSpace(user.FullName())
	if name == "" {
		name = user.Email
	}
	cta := brand.link("/dashboard")
	greeting := fmt.Sprintf("Hello <strong>%s</strong>,", htmlEscape(name))
	var badge, intro, next, security string
	var kindBadge types.EmailBadge
	switch kind {
	case types.CredentialReset:
		subject = "Your TIPER DFMS password has been reset"
		badge = "Password reset"
		kindBadge = types.EmailBadgeWarning
		intro = fmt.Sprintf("An administrator has reset your TIPER DFMS password. Sign in with <strong>%s</strong> and the temporary password below, then choose a new password when prompted.", htmlEscape(user.Email))
		next = "Click the password box to select it, copy, then sign in and replace this temporary password. It should not be shared."
		security = "If you did not request this change, contact your system administrator immediately and do not use the password in this email."
		smsBody = fmt.Sprintf(
			"%s: your password was reset. Temporary password: %s. Sign in and change it immediately.",
			brand.smsName(), tempPassword,
		)
	default:
		subject = "Welcome to TIPER DFMS"
		badge = "Account ready"
		kindBadge = types.EmailBadgeSuccess
		intro = fmt.Sprintf("An administrator has created an account for you. Sign in with <strong>%s</strong> and the temporary password below. You will be asked to choose a new password on first login, then complete verification with a one-time code when prompted.", htmlEscape(user.Email))
		next = "Click the password box to select it, copy, then open DFMS, sign in, set your own password, and complete the one-time verification code."
		security = "If you did not expect this email, contact your system administrator and do not sign in."
		smsBody = fmt.Sprintf(
			"%s: your account was created. Temporary password: %s. Sign in and change it on first login.",
			brand.smsName(), tempPassword,
		)
	}
	if cta != "" {
		smsBody = truncateSMS(smsBody + " " + cta)
	}
	emailBody = renderCorporateEmail(brand, badge, kindBadge, greeting, intro, "Sign-in details",
		[]kv{
			{Label: "Email", Value: user.Email},
			{Label: "Temporary password", Value: tempPassword, Highlight: true, Secret: true},
		},
		next, security, "Open DFMS", cta,
	)
	return subject, emailBody, smsBody
}

func deliverTempPassword(ctx context.Context, db *gorm.DB, user *models.User, tempPassword string, kind types.CredentialKind) {
	brand := loadBrand(ctx, db)
	if brand.PortalURL == "" && db != nil {
		brand.PortalURL = portalurl.Base(ctx, db)
	}
	subject, emailBody, smsBody := credentialMessages(user, tempPassword, brand, kind)
	if err := sendBranded(ctx, []string{user.Email}, subject, emailBody); err != nil {
		logs.Errorf("%s email to %s failed: %v", kind, user.Email, err)
		logs.Warnf("%s email failed for %s — temporary password (relay manually): %s", kind, user.Email, tempPassword)
	}
	phone := strings.TrimSpace(user.Profile.PhoneNumber)
	if phone == "" {
		return
	}
	if err := sms.Send(ctx, phone, smsBody); err != nil {
		logs.Errorf("%s sms to %s failed: %v", kind, phone, err)
	}
}
