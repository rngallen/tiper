package auth

import (
	"errors"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/internal/integrations"
	"dfms/pkg/constants"
	"dfms/pkg/logs"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

var errRefreshIdle = errors.New("refresh session idle or missing")

func activityOf(row models.RefreshToken) time.Time {
	cutoff := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if row.LastSeen.After(cutoff) {
		return row.LastSeen
	}
	return row.CreatedAt
}

// findLiveRefresh returns the newest unused refresh row for the user that is
// still inside the idle window. Lookup is by subject only — NAT, VPN, and
// proxy hops change c.IP() between requests and must not end a live session.
func findLiveRefresh(db *gorm.DB, subject string, now, idleSince time.Time) (*models.RefreshToken, error) {
	var rows []models.RefreshToken
	q := db.Where("Subject = ? AND ExpiresAt > ?", subject, now).Order("CreatedAt DESC")
	if err := q.Find(&rows).Error; err != nil {
		// LastSeen may be missing before migrate; retry without that column.
		if err := db.Select("Subject", "IpAddress", "ExpiresAt", "CreatedAt").
			Where("Subject = ? AND ExpiresAt > ?", subject, now).
			Order("CreatedAt DESC").Find(&rows).Error; err != nil {
			return nil, err
		}
	}
	for i := range rows {
		if !activityOf(rows[i]).Before(idleSince) {
			return &rows[i], nil
		}
	}
	return nil, errRefreshIdle
}

func sessionWindow() (now, idleSince time.Time) {
	now = time.Now().UTC()
	return now, now.Add(-middleware.SessionIdleWindow())
}

// touchSession slides LastSeen and the refresh cookie expiry so a live
// operator is not signed out while they work. Continue and the SPA heartbeat
// call this; /auth/refresh still rotates tokens when the access cookie is near
// expiry.
func (h *AuthHandler) touchSession(c fiber.Ctx) error {
	uid := middleware.GetUserUIDFromContext(c)
	if uid == "" {
		return response.Unauthorized(c, constants.ErrInvalidRefreshToken.Error())
	}
	now, idleSince := sessionWindow()
	row, err := findLiveRefresh(h.db.WithContext(c.Context()), uid, now, idleSince)
	if err != nil {
		if errors.Is(err, errRefreshIdle) {
			return response.Unauthorized(c, constants.ErrInvalidRefreshToken.Error())
		}
		logs.Errorf("session touch lookup sub=%s: %v", uid, err)
		return response.InternalServerError(c)
	}

	expires := now.Add(middleware.RefreshTokenDuration)
	res := h.db.WithContext(c.Context()).Model(&models.RefreshToken{}).
		Where("Subject = ? AND IpAddress = ?", row.Subject, row.IpAddress).
		Updates(map[string]any{"LastSeen": now, "ExpiresAt": expires})
	if res.Error != nil {
		// Column LastSeen may not exist yet — slide expiry only.
		res = h.db.WithContext(c.Context()).Model(&models.RefreshToken{}).
			Where("Subject = ? AND IpAddress = ?", row.Subject, row.IpAddress).
			Update("ExpiresAt", expires)
	}
	if res.Error != nil {
		return response.InternalServerError(c)
	}

	if val := c.Cookies(types.RefreshCookieName); val != "" {
		c.Cookie(authCookieBase(types.RefreshCookieName, val, expires))
	}

	sess := integrations.LiveSession()
	return response.OkDetail(c, fiber.Map{
		"sessionIdleMinutes":     sess.IdleMinutes,
		"sessionIdleWarnSeconds": sess.WarnSeconds,
		"touchedAt":              now,
	})
}
