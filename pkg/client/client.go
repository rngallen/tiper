// Package client provides a small, opinionated HTTP client wrapper with
// exponential backoff + jitter retries, configurable headers, body, timeouts
// and per-request context cancellation.
//
// It is designed for outbound integrations (EWURA, SMS gateway, Sage, etc.)
// where transient network failures and 5xx errors should be retried
// automatically.
package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"dfms/pkg/logs"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"time"
)

// Retry configuration constants.
const (
	maxRetries     = 3
	baseBackoff    = 1 * time.Second
	backoffFactor  = 2.0
	maxBackoff     = 30 * time.Second
	defaultTimeout = 60 * time.Second
)

// ErrMaxRetriesExceeded is returned when all retry attempts are exhausted.
var ErrMaxRetriesExceeded = errors.New("failed after maximum retries")

// retryableStatusCodes lists HTTP status codes that warrant a retry.
var retryableStatusCodes = map[int]struct{}{
	http.StatusRequestTimeout:      {}, // 408
	http.StatusTooManyRequests:     {}, // 429
	http.StatusInternalServerError: {}, // 500
	http.StatusBadGateway:          {}, // 502
	http.StatusServiceUnavailable:  {}, // 503
	http.StatusGatewayTimeout:      {}, // 504
}

// Shared transport — safe for concurrent use. Tuned for typical web traffic.
var defaultTransport = &http.Transport{
	TLSHandshakeTimeout: 10 * time.Second,
	MaxIdleConns:        100,
	IdleConnTimeout:     30 * time.Second,
	MaxIdleConnsPerHost: 20,
	DisableCompression:  false, // gzip helps for typical JSON APIs
	DialContext: (&net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
}

// Default HTTP client (shared, concurrent-safe).
var defaultClient = &http.Client{
	Transport: defaultTransport,
	Timeout:   defaultTimeout,
}

// Header represents a key-value pair for HTTP headers.
type Header struct {
	Key   string
	Value string
}

// ClientOptions provides configuration for the HTTP client.
type ClientOptions struct {
	Headers    []Header
	Body       io.Reader
	Timeout    time.Duration
	Ctx        context.Context
	MaxRetries int
}

// ClientOption is a function that modifies ClientOptions.
type ClientOption func(*ClientOptions)

// WithHeaders sets custom headers for the request.
func WithHeaders(headers []Header) ClientOption {
	return func(opts *ClientOptions) {
		opts.Headers = append(opts.Headers, headers...)
	}
}

// WithBody sets the request body. The body is buffered into memory so it can
// safely be replayed across retries. Pass nil for requests without a body.
func WithBody(body io.Reader) ClientOption {
	return func(opts *ClientOptions) {
		opts.Body = body
	}
}

// WithTimeout sets a custom per-request timeout (overrides the default 60s).
// Use a non-positive value to disable the override.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Timeout = timeout
	}
}

// WithContext propagates a caller-supplied context for cancellation/deadline.
func WithContext(ctx context.Context) ClientOption {
	return func(opts *ClientOptions) {
		opts.Ctx = ctx
	}
}

// WithMaxRetries overrides the default retry count for a single call.
// Pass 0 to disable retries.
func WithMaxRetries(n int) ClientOption {
	return func(opts *ClientOptions) {
		if n < 0 {
			n = 0
		}
		opts.MaxRetries = n
	}
}

// Client makes an HTTP request with optional headers, body, timeout and context.
// It retries on network errors and retryable HTTP status codes (408, 429, 5xx)
// using exponential backoff with jitter.
//
// The caller is responsible for closing the returned response body.
func Client(method, url string, options ...ClientOption) (*http.Response, error) {
	if method == "" {
		return nil, errors.New("client: method is required")
	}
	if url == "" {
		return nil, errors.New("client: url is required")
	}

	opts := &ClientOptions{
		Ctx:        context.Background(),
		MaxRetries: maxRetries,
	}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}

	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	// Buffer the body once so we can safely retry without exhausting the reader.
	var bodyBytes []byte
	if opts.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("client: read request body: %w", err)
		}
	}

	httpClient := defaultClient
	if opts.Timeout > 0 && opts.Timeout != defaultTimeout {
		httpClient = &http.Client{
			Transport: defaultTransport,
			Timeout:   opts.Timeout,
		}
	}

	logs.Infof("HTTP %s %s", method, url)

	attempts := max(opts.MaxRetries+1, 1)

	var lastErr error
	for attempt := range attempts {
		// Honour cancellation between attempts.
		if err := opts.Ctx.Err(); err != nil {
			if lastErr == nil {
				return nil, fmt.Errorf("client: %s %s: %w", method, url, err)
			}
			return nil, fmt.Errorf("client: %s %s: %w (after: %v)", method, url, err, lastErr)
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(opts.Ctx, method, url, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("client: build request %s %s: %w", method, url, err)
		}

		for _, header := range opts.Headers {
			if header.Key == "" {
				continue
			}
			req.Header.Set(header.Key, header.Value)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			logs.Warnf("client: attempt %d/%d for %s %s failed: %v", attempt+1, attempts, method, url, err)
			if attempt < attempts-1 {
				if waitErr := sleepBackoff(opts.Ctx, attempt); waitErr != nil {
					return nil, fmt.Errorf("client: %s %s: %w", method, url, waitErr)
				}
				continue
			}
			break
		}

		// Retry on retryable server statuses.
		if _, retry := retryableStatusCodes[resp.StatusCode]; retry && attempt < attempts-1 {
			lastErr = fmt.Errorf("retryable status %d", resp.StatusCode)
			drainAndClose(resp)
			logs.Warnf("client: attempt %d/%d for %s %s got %d, retrying", attempt+1, attempts, method, url, resp.StatusCode)
			if waitErr := sleepBackoff(opts.Ctx, attempt); waitErr != nil {
				return nil, fmt.Errorf("client: %s %s: %w", method, url, waitErr)
			}
			continue
		}

		// Success or non-retryable status: hand the body back to the caller.
		return resp, nil
	}

	if lastErr == nil {
		lastErr = errors.New("unknown error")
	}
	return nil, fmt.Errorf("client: %s %s: %w: %v", method, url, ErrMaxRetriesExceeded, lastErr)
}

// sleepBackoff sleeps for an exponentially-increasing interval with ±25% jitter,
// honouring context cancellation.
func sleepBackoff(ctx context.Context, attempt int) error {
	backoff := min(time.Duration(float64(baseBackoff)*math.Pow(backoffFactor, float64(attempt))), maxBackoff)
	jitter := jitterFraction() // in [-0.25, +0.25]
	sleep := time.Duration(float64(backoff) * (1 + jitter))
	if sleep < 0 {
		sleep = backoff
	}

	timer := time.NewTimer(sleep)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// jitterFraction returns a pseudo-random value in the range [-0.25, +0.25].
// Falls back to 0 (no jitter) if the system random source is unavailable.
func jitterFraction() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	// Map uint64 -> [0,1) -> [-0.25,+0.25]
	u := binary.BigEndian.Uint64(b[:])
	f := float64(u) / float64(^uint64(0))
	return (f - 0.5) * 0.5
}

// drainAndClose drains and closes a response body so the underlying connection
// can be returned to the keep-alive pool.
func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
