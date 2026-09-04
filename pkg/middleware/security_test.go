package middleware

import (
	"strings"
	"testing"

	"dfms/pkg/config"
)

func TestContentSecurityPolicyHTTP(t *testing.T) {
	prev := config.Conf
	t.Cleanup(func() { config.Conf = prev })

	config.Conf.App.AllowInsecureHTTP = true
	config.Conf.App.Debug = false
	if strings.Contains(contentSecurityPolicy(), "upgrade-insecure-requests") {
		t.Fatal("plain HTTP must not send upgrade-insecure-requests")
	}

	config.Conf.App.AllowInsecureHTTP = false
	config.Conf.App.Debug = false
	if !strings.Contains(contentSecurityPolicy(), "upgrade-insecure-requests") {
		t.Fatal("HTTPS must send upgrade-insecure-requests")
	}
}
