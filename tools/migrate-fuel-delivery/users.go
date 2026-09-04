package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/utils"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
)

type djangoProfile struct {
	Phone string
	Title string
}

func pickPhone(phone, mobile string) string {
	phone = strings.TrimSpace(phone)
	if phone != "" {
		return phone
	}
	return strings.TrimSpace(mobile)
}

func copyTitles(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	if adminID == 0 {
		return nil
	}
	rows, err := pg.Query(ctx, `SELECT COALESCE(name,'') FROM "UserTitle" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	created := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		var exists models.Title
		if dest.Where("Name = ?", name).First(&exists).Error == nil {
			continue
		}
		row := models.Title{Name: name, CreatedByID: adminID}
		if err := dest.Create(&row).Error; err != nil {
			fmt.Printf("  skip title %s: %v\n", name, err)
			continue
		}
		created++
	}
	if created > 0 {
		fmt.Printf("  + %d user titles\n", created)
	}
	return rows.Err()
}

func copyUsers(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	profiles := loadDjangoProfiles(ctx, pg)
	rows, err := pg.Query(ctx, `
		SELECT id, email, COALESCE(first_name,''), COALESCE(last_name,''), COALESCE(is_active,true)
		FROM "AuthUser" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var email, first, last string
		var active bool
		if err := rows.Scan(&id, &email, &first, &last, &active); err != nil {
			return err
		}
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			continue
		}
		prof := profiles[id]
		if uid := idByDjango(dest, &models.User{}, id); uid != 0 {
			ensureUserProfile(dest, uid, adminID, prof)
			continue
		}
		var existing models.User
		if dest.Where("Email = ?", email).First(&existing).Error == nil {
			stampDjangoID(dest, &models.User{}, existing.ID, id)
			ensureUserProfile(dest, existing.ID, adminID, prof)
			continue
		}
		pwd, err := utils.GenerateSecurePassword(16)
		if err != nil {
			return err
		}
		phone := availableProfilePhone(dest, prof.Phone, 0)
		title := ensureTitleName(dest, adminID, prof.Title)
		row := models.User{
			Email:              email,
			FirstName:          firstNonEmpty(first, "Imported"),
			LastName:           firstNonEmpty(last, "User"),
			Password:           pwd,
			IsActive:           active,
			MustChangePassword: true,
			DjangoID:           uint(id),
			Profile: models.Profile{
				PhoneNumber: phone,
				Title:       title,
				AppearanceSetting: map[string]any{
					"theme":        "light",
					"compactMode":  true,
					"largeText":    false,
					"sidebarState": true,
				},
			},
		}
		if err := row.EncryptPassword(); err != nil {
			return err
		}
		if err := dest.Create(&row).Error; err != nil {
			fmt.Printf("  skip user %s: %v\n", email, err)
			continue
		}
		if title != "" {
			_ = markTitleUsed(dest, title)
		}
		fmt.Printf("  + user %s (django %d → %d)\n", email, id, row.ID)
	}
	return rows.Err()
}

func loadDjangoProfiles(ctx context.Context, pg *pgx.Conn) map[int64]djangoProfile {
	out := map[int64]djangoProfile{}
	queries := []string{
		`SELECT p.user_id, COALESCE(p.phone,''), COALESCE(p.mobile,''), COALESCE(t.name,'')
		 FROM "UserProfile" p LEFT JOIN "UserTitle" t ON t.id = p.title_id`,
		`SELECT user_id, COALESCE(phone,''), COALESCE(mobile,''), COALESCE(title_name,'')
		 FROM "UserProfile"`,
		`SELECT user_id, COALESCE(phone_number,''), COALESCE(mobile_number,''), COALESCE(title,'')
		 FROM "UserProfile"`,
	}
	for _, q := range queries {
		rows, err := pg.Query(ctx, q)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id int64
			var phone, mobile, title string
			if err := rows.Scan(&id, &phone, &mobile, &title); err != nil {
				rows.Close()
				out = map[int64]djangoProfile{}
				break
			}
			out[id] = djangoProfile{Phone: pickPhone(phone, mobile), Title: strings.TrimSpace(title)}
		}
		scanErr := rows.Err()
		rows.Close()
		if scanErr == nil && len(out) > 0 {
			return out
		}
	}
	return out
}

func ensureUserProfile(dest *gorm.DB, userID, adminID uint, prof djangoProfile) {
	if userID == 0 {
		return
	}
	title := ensureTitleName(dest, adminID, prof.Title)
	var p models.Profile
	err := dest.Where("UserID = ?", userID).First(&p).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		phone := availableProfilePhone(dest, prof.Phone, userID)
		_ = dest.Create(&models.Profile{
			UserID:      userID,
			PhoneNumber: phone,
			Title:       title,
			AppearanceSetting: map[string]any{
				"theme":        "light",
				"compactMode":  true,
				"largeText":    false,
				"sidebarState": true,
			},
		}).Error
		if title != "" {
			_ = markTitleUsed(dest, title)
		}
		return
	}
	updates := map[string]any{}
	if strings.TrimSpace(p.Title) == "" && title != "" {
		updates["Title"] = title
	}
	if strings.TrimSpace(p.PhoneNumber) == "" {
		if phone := availableProfilePhone(dest, prof.Phone, userID); phone != "" {
			updates["PhoneNumber"] = phone
		}
	}
	if len(updates) == 0 {
		return
	}
	_ = dest.Model(&p).Updates(updates).Error
	if title != "" {
		_ = markTitleUsed(dest, title)
	}
}

func availableProfilePhone(dest *gorm.DB, phone string, excludeUserID uint) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	q := dest.Model(&models.Profile{}).Where("PhoneNumber = ?", phone)
	if excludeUserID != 0 {
		q = q.Where("UserID <> ?", excludeUserID)
	}
	var n int64
	if err := q.Limit(1).Count(&n).Error; err != nil || n > 0 {
		return ""
	}
	return phone
}

func ensureTitleName(dest *gorm.DB, adminID uint, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var title models.Title
	if dest.Where("Name = ?", name).First(&title).Error == nil {
		return title.Name
	}
	if adminID == 0 {
		return ""
	}
	title = models.Title{Name: name, CreatedByID: adminID}
	if err := dest.Create(&title).Error; err != nil {
		var again models.Title
		if dest.Where("Name = ?", name).First(&again).Error == nil {
			return again.Name
		}
		return ""
	}
	return title.Name
}

func markTitleUsed(dest *gorm.DB, name string) error {
	var title models.Title
	if err := dest.Where("Name = ?", name).First(&title).Error; err != nil {
		return err
	}
	return title.UpdateHasData(dest)
}
