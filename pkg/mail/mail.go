// Package mail provides SMTP email delivery used for 2FA one-time passwords and
// workflow notifications. When mail is disabled (Settings → Mail) a no-op
// logging mailer is used so the rest of the system keeps working in development.
//
// The SMTP client keeps one authenticated connection open and reuses it across
// sends (dial + AUTH on Init / warm-up, then Send). On send failure the
// connection is closed and re-authenticated once before retrying. Settings
// save rebuilds the client via Init.
//
// Delivery is implemented with github.com/wneessen/go-mail so later vendor
// approval emails can attach documents without hand-rolling MIME.
package mail

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dfms/pkg/config"
	"dfms/pkg/logs"

	gomail "github.com/wneessen/go-mail"
)

// Attachment is a file to include on an outbound message (e.g. a report PDF).
type Attachment struct {
	Filename    string
	ContentType string // optional MIME type; empty lets go-mail infer
	Content     []byte // in-memory payload (preferred for generated files)
	Path        string // filesystem path when Content is empty
}

// Message is a full outbound email, including optional attachments.
type Message struct {
	To          []string
	Subject     string
	HTMLBody    string
	Attachments []Attachment
	// Embeds are inline CID images referenced as src="cid:<Filename>" in HTML.
	// They travel with the message so mail clients do not fetch remote URLs.
	Embeds []Attachment
}

// Mailer sends HTML email (optionally with attachments) to one or more recipients.
type Mailer interface {
	Send(ctx context.Context, to []string, subject, htmlBody string) error
	SendMessage(ctx context.Context, msg Message) error
}

// LoggingMailer logs messages instead of sending them (development default).
type LoggingMailer struct{}

// Send implements Mailer by logging the recipients and subject at info level
// instead of delivering anything; the body is discarded. It never fails.
func (LoggingMailer) Send(_ context.Context, to []string, subject, _ string) error {
	logs.Infof("[mail:log] to=%s subject=%q (logging mailer — configure SMTP in production)", strings.Join(to, ","), subject)
	return nil
}

// SendMessage implements Mailer by logging the recipients, subject and
// attachment count instead of delivering anything; body and attachment
// contents are discarded. It never fails.
func (LoggingMailer) SendMessage(_ context.Context, msg Message) error {
	logs.Infof("[mail:log] to=%s subject=%q attachments=%d embeds=%d (logging mailer — configure SMTP in production)",
		strings.Join(msg.To, ","), msg.Subject, len(msg.Attachments), len(msg.Embeds))
	return nil
}

// SMTPMailer sends mail through a reused SMTP connection (dial + AUTH once,
// then Send until the connection dies or Init rebuilds the client).
type SMTPMailer struct {
	mu        sync.Mutex
	client    *gomail.Client
	fromName  string
	fromEmail string
	ready     bool // true after a successful DialWithContext
}

// Default is the process-wide mailer. It logs until Init configures SMTP.
var Default Mailer = LoggingMailer{}

// Init configures the default mailer from config. If mail is disabled it keeps
// the logging mailer. When enabled it builds the SMTP client, replaces Default,
// and warms an authenticated connection in the background.
func Init(cfg config.MailConfig) {
	Close()

	if !cfg.Enabled {
		logs.Info("mail disabled; using logging mailer")
		Default = LoggingMailer{}
		return
	}

	opts := []gomail.Option{
		gomail.WithPort(cfg.Port),
		gomail.WithTimeout(30 * time.Second),
	}
	mode := "plain"
	switch {
	case cfg.UseSSL || cfg.Port == 465:
		// Implicit TLS/SSL (SMTPS) — same as SSL=true in most mail clients.
		opts = append(opts, gomail.WithSSL())
		mode = "implicit-tls"
	case cfg.UseTLS || cfg.Port == 587:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
		mode = "starttls"
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	}
	if cfg.User != "" {
		// Auto-discover (LOGIN/PLAIN/…) after TLS — many hosts advertise AUTH
		// only over SSL/STARTTLS and may not offer PLAIN.
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(cfg.User),
			gomail.WithPassword(cfg.Password),
		)
	}

	client, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		logs.Errorf("mail: init smtp client: %v; falling back to logging mailer", err)
		Default = LoggingMailer{}
		return
	}

	mailer := &SMTPMailer{
		client:    client,
		fromName:  cfg.FromName,
		fromEmail: cfg.FromEmail,
	}
	Default = mailer
	logs.Infof("mail enabled via SMTP %s:%d (%s); warming connection", cfg.Host, cfg.Port, mode)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := mailer.ensureDialed(ctx); err != nil {
			logs.Warnf("mail: SMTP warm-up failed (will retry on first send): %v", err)
			return
		}
		logs.Info("mail: SMTP connection authenticated and ready")
	}()
}

// Send dispatches a simple HTML mail via the package default mailer.
func Send(ctx context.Context, to []string, subject, htmlBody string) error {
	return Default.Send(ctx, to, subject, htmlBody)
}

// Close releases the SMTP connection and resets Default to the logging mailer.
// Safe during shutdown or before Init replaces Default.
func Close() {
	prev := Default
	Default = LoggingMailer{}
	if m, ok := prev.(*SMTPMailer); ok {
		m.close()
	}
}

func (m *SMTPMailer) close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = false
	if m.client != nil {
		_ = m.client.Close()
	}
}

// Send implements Mailer for SMTPMailer.
func (m *SMTPMailer) Send(ctx context.Context, to []string, subject, htmlBody string) error {
	return m.SendMessage(ctx, Message{To: to, Subject: subject, HTMLBody: htmlBody})
}

// SendMessage implements Mailer for SMTPMailer. Reuses the dialed SMTP session;
// on failure reconnects once and retries.
func (m *SMTPMailer) SendMessage(ctx context.Context, msg Message) error {
	if len(msg.To) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	gm, err := m.buildMsg(msg)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureDialedLocked(ctx); err != nil {
		return fmt.Errorf("mail: dial: %w", err)
	}
	if err := m.client.Send(gm); err != nil {
		m.ready = false
		_ = m.client.Close()
		if err2 := m.ensureDialedLocked(ctx); err2 != nil {
			return fmt.Errorf("mail: send: %w (reconnect: %v)", err, err2)
		}
		if err2 := m.client.Send(gm); err2 != nil {
			m.ready = false
			_ = m.client.Close()
			return fmt.Errorf("mail: send: %w", err2)
		}
	}
	return nil
}

func (m *SMTPMailer) ensureDialed(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureDialedLocked(ctx)
}

func (m *SMTPMailer) ensureDialedLocked(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("smtp client not initialized")
	}
	if m.ready {
		return nil
	}
	if err := m.client.DialWithContext(ctx); err != nil {
		m.ready = false
		return err
	}
	m.ready = true
	return nil
}

func (m *SMTPMailer) buildMsg(msg Message) (*gomail.Msg, error) {
	gm := gomail.NewMsg()
	if m.fromName != "" {
		if err := gm.FromFormat(m.fromName, m.fromEmail); err != nil {
			return nil, fmt.Errorf("mail: from: %w", err)
		}
	} else if err := gm.From(m.fromEmail); err != nil {
		return nil, fmt.Errorf("mail: from: %w", err)
	}
	if err := gm.To(msg.To...); err != nil {
		return nil, fmt.Errorf("mail: to: %w", err)
	}
	gm.Subject(msg.Subject)
	gm.SetBodyString(gomail.TypeTextHTML, msg.HTMLBody)

	if err := addFiles(gm, msg.Attachments, false); err != nil {
		return nil, err
	}
	if err := addFiles(gm, msg.Embeds, true); err != nil {
		return nil, err
	}
	return gm, nil
}

func addFiles(gm *gomail.Msg, files []Attachment, embed bool) error {
	for _, att := range files {
		if att.Filename == "" && att.Path == "" && len(att.Content) == 0 {
			continue
		}
		var fileOpts []gomail.FileOption
		if att.ContentType != "" {
			fileOpts = append(fileOpts, gomail.WithFileContentType(gomail.ContentType(att.ContentType)))
		}
		name := att.Filename
		if name == "" {
			name = "attachment"
		}
		switch {
		case len(att.Content) > 0:
			if embed {
				if err := gm.EmbedReader(name, bytes.NewReader(att.Content), fileOpts...); err != nil {
					return fmt.Errorf("mail: embed %s: %w", name, err)
				}
			} else if err := gm.AttachReader(name, bytes.NewReader(att.Content), fileOpts...); err != nil {
				return fmt.Errorf("mail: attach %s: %w", name, err)
			}
		case att.Path != "":
			if att.Filename != "" {
				fileOpts = append(fileOpts, gomail.WithFileName(att.Filename))
			}
			if embed {
				gm.EmbedFile(att.Path, fileOpts...)
			} else {
				gm.AttachFile(att.Path, fileOpts...)
			}
		}
	}
	return nil
}
