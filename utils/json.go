package utils

import (
	"bytes"
	"fmt"
	"io"

	"github.com/goccy/go-json"
)

// ConvertToBytes serializes v to JSON and returns an io.Reader suitable for
// HTTP request bodies (e.g. client.WithBody). Prefer this over calling
// encoding/json or goccy/go-json directly from call sites.
func ConvertToBytes(v any) (io.Reader, error) {
	if v == nil {
		return nil, fmt.Errorf("json: cannot encode nil value")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json encoding failed: %w", err)
	}
	return bytes.NewReader(b), nil
}
