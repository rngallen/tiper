// Package sms sends SMS messages (OTP codes). When disabled (Settings → SMS) a
// logging sender is used so local development can proceed without an SMS gateway.
package sms

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"dfms/pkg/client"
	"dfms/pkg/config"
	"dfms/pkg/logs"
	"dfms/utils"
)

// requestTimeout bounds a single OTP SMS call; login waits on this, so it is
// kept short (retries are handled inside pkg/client).
const requestTimeout = 15 * time.Second

// Sender delivers a text message to a single E.164-style number (no + prefix).
type Sender interface {
	Send(ctx context.Context, to, message string) error
}

// LoggingSender logs SMS payloads instead of sending them.
type LoggingSender struct{}

// Send implements Sender by logging the destination and message at info level
// instead of calling any gateway. It never fails.
func (LoggingSender) Send(_ context.Context, to, message string) error {
	logs.Infof("[sms:log] to=%s message=%q (logging sender — configure SMS in production)", to, message)
	return nil
}

// HTTPSender posts JSON to a configurable gateway endpoint via pkg/client,
// which provides retries with exponential backoff + jitter and a shared,
// pooled transport.
// Expected body: {"to":"<phone>","message":"<text>","sender":"<senderId>"}
type HTTPSender struct {
	apiURL   string
	apiKey   string
	senderID string
}

// Default is the process-wide SMS sender.
var Default Sender = LoggingSender{}

// Init configures the default SMS sender from application config.
func Init(cfg config.SMSConfig) {
	if !cfg.Enabled {
		logs.Info("sms disabled; using logging sender")
		Default = LoggingSender{}
		return
	}
	Default = &HTTPSender{
		apiURL:   cfg.APIURL,
		apiKey:   cfg.APIKey,
		senderID: cfg.SenderID,
	}
	logs.Infof("sms enabled via HTTP gateway %s", cfg.APIURL)
}

// Send dispatches an SMS using the package default sender.
func Send(ctx context.Context, to, message string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("sms: recipient required")
	}
	return Default.Send(ctx, to, message)
}

// Send implements Sender by POSTing a JSON payload with fields phone_number,
// message and sender_id to the configured gateway URL, authenticated with a
// Bearer API key. The call goes through pkg/client (retries with backoff)
// under a 15-second timeout. It returns an error if the payload cannot be
// encoded, the request fails after retries, or the gateway responds with a
// non-2xx status — in the last case the first 512 bytes of the response body
// are included in the error.
func (s *HTTPSender) Send(ctx context.Context, to, message string) error {
	body, err := utils.ConvertToBytes(map[string]string{
		"phone_number": to,
		"message":      message,
		"sender_id":    s.senderID,
	})
	if err != nil {
		return fmt.Errorf("sms: encode payload: %w", err)
	}

	headers := []client.Header{
		{Key: "Content-Type", Value: "application/json"},
		{Key: "Authorization", Value: "Bearer " + s.apiKey},
	}

	res, err := client.Post(s.apiURL,
		client.WithContext(ctx),
		client.WithTimeout(requestTimeout),
		client.WithHeaders(headers),
		client.WithBody(body),
	)

	if err != nil {
		return fmt.Errorf("sms: request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("sms: gateway returned %d: %s", res.StatusCode, strings.TrimSpace(string(respBody)))
	}
	logs.Info("sms sent successfully")
	return nil
}
