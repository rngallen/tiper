package client

import "net/http"

// Convenience wrappers for HTTP methods with retries and exponential backoff.

func Post(url string, opts ...ClientOption) (*http.Response, error) {
	return Client(http.MethodPost, url, opts...)
}
