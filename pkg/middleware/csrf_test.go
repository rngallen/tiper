package middleware

import "testing"

func TestShouldSkipCSRF(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, access, refresh, auth string
		skip                              bool
	}{
		{name: "login has no session", path: "/api/v1/auth/login", skip: false},
		{name: "mfa bearer no cookie", path: "/api/v1/auth/mfa/verify", auth: "Bearer mfa-token", skip: true},
		{name: "mfa leftover access cookie", path: "/api/v1/auth/mfa/verify", access: "stale", auth: "Bearer mfa-token", skip: true},
		{name: "mfa resend leftover refresh", path: "/api/v1/auth/mfa/resend", refresh: "stale", auth: "Bearer mfa-token", skip: true},
		{name: "empty bearer still csrf", path: "/api/v1/stock/receipts", auth: "Bearer ", skip: false},
		{name: "bearer plus access cookie", path: "/api/v1/stock/receipts", access: "cookie", auth: "Bearer attacker", skip: false},
		{name: "bearer plus refresh cookie", path: "/api/v1/auth/refresh", refresh: "cookie", auth: "Bearer attacker", skip: false},
		{name: "spa cookie no bearer", path: "/api/v1/stock/receipts", access: "tok", skip: false},
		{name: "session touch skips csrf", path: "/api/v1/auth/session/touch", access: "tok", refresh: "ref", skip: true},
		{name: "session touch trailing slash", path: "/api/v1/auth/session/touch/", access: "tok", skip: true},
		{name: "public document verify", path: "/api/v1/public/documents/ilr/uid/sig/pdf", skip: true},
		{name: "refresh still csrf", path: "/api/v1/auth/refresh", access: "tok", skip: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldSkipCSRF(tc.path, tc.access, tc.refresh, tc.auth)
			if got != tc.skip {
				t.Fatalf("skip=%v want %v", got, tc.skip)
			}
		})
	}
}
