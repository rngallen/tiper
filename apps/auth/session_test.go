package auth

import (
	"testing"
	"time"

	"dfms/apps/models"
)

func TestActivityOfPrefersLastSeen(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	seen := time.Date(2026, 8, 1, 10, 8, 0, 0, time.UTC)
	got := activityOf(models.RefreshToken{CreatedAt: created, LastSeen: seen})
	if !got.Equal(seen) {
		t.Fatalf("got %s", got)
	}
}

func TestActivityOfFallsBackToCreatedAt(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	got := activityOf(models.RefreshToken{CreatedAt: created})
	if !got.Equal(created) {
		t.Fatalf("got %s", got)
	}
}
