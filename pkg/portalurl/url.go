// Package portalurl resolves the public frontend base URL for email CTAs.
package portalurl

import (
	"context"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/docsig"

	"gorm.io/gorm"
)

// Base returns Company.PortalURL when configured (no trailing slash).
func Base(ctx context.Context, db *gorm.DB) string {
	if db == nil {
		return ""
	}
	var company models.Company
	if err := db.WithContext(ctx).Select("PortalURL").First(&company).Error; err != nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(company.PortalURL), "/")
}

// Join appends a path to base (base may be empty → returns "").
func Join(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// DocumentVerifyURL is the public (no-login) scan page with HMAC path segments.
// Shape: {portalUrl}/verify/document/{kind}/{uid}/{sig}
func DocumentVerifyURL(ctx context.Context, db *gorm.DB, kind, uid string) string {
	rel := docsig.Path(kind, uid)
	if rel == "" {
		return ""
	}
	return Join(Base(ctx, db), rel)
}
