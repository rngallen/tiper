package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func copyLicenses(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	rows, err := pg.Query(ctx, `
		SELECT COALESCE(licence_number,''), COALESCE(licencee,''), COALESCE(licence_class,''),
		       COALESCE(licence_type,''), COALESCE(sector,''), COALESCE(zone_name,''),
		       COALESCE(region_name,''), COALESCE(district_name,''), COALESCE(tin_number,''),
		       COALESCE(phone,''), COALESCE(email,''), issue_date, expiry_date
		FROM "EwuraPetroleumLicences" ORDER BY licence_number`)
	if err != nil {
		fmt.Printf("  skip EWURA licences (%v)\n", err)
		return nil
	}
	defer rows.Close()
	now := time.Now().UTC().Truncate(24 * time.Hour)
	batch := make([]models.EwuraPetroleumLicense, 0, 80)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := dest.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(batch, 80).Error; err != nil {
			return err
		}
		fmt.Printf("  + %d EWURA licences\n", len(batch))
		batch = batch[:0]
		return nil
	}
	for rows.Next() {
		var number, licensee, class, ltype, sector, zone, region, district, tin, phone, email string
		var issue, expiry *time.Time
		if err := rows.Scan(&number, &licensee, &class, &ltype, &sector, &zone, &region, &district, &tin, &phone, &email, &issue, &expiry); err != nil {
			return err
		}
		number = strings.ToUpper(strings.TrimSpace(number))
		if number == "" {
			continue
		}
		active := true
		if expiry != nil && !expiry.After(now) {
			active = false
		}
		batch = append(batch, models.EwuraPetroleumLicense{
			LicenseNumber: number,
			ContentType:   types.EwuraLicenseContent,
			Licensee:      strings.TrimSpace(licensee),
			LicenseClass:  strings.TrimSpace(class),
			LicenseType:   strings.TrimSpace(ltype),
			Sector:        strings.TrimSpace(sector),
			ZoneName:      strings.TrimSpace(zone),
			RegionName:    strings.TrimSpace(region),
			DistrictName:  strings.TrimSpace(district),
			TinNumber:     strings.TrimSpace(tin),
			Phone:         strings.TrimSpace(phone),
			Email:         strings.TrimSpace(email),
			IssueDate:     issue,
			ExpiryDate:    expiry,
			IsActive:      active,
		})
		if len(batch) >= 80 {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return flush()
}

func copyCustomers(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(CAST(code AS text),''), COALESCE(value,''), COALESCE(email,''),
		       COALESCE(ewura_licence,'NA'), COALESCE(is_active,true), created
		FROM "SageCustomer" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var code, name, email, lic string
		var active bool
		var created time.Time
		if err := rows.Scan(&id, &code, &name, &email, &lic, &active, &created); err != nil {
			return err
		}
		if idByDjango(dest, &models.Customer{}, id) != 0 {
			continue
		}
		code = strings.TrimSpace(code)
		if code == "" {
			code = fmt.Sprintf("%d", id+10000)
		}
		name = firstNonEmpty(name, code)
		if lic == "" {
			lic = "NA"
		}
		var existing models.Customer
		if dest.Where("Code = ?", code).First(&existing).Error == nil {
			stampDjangoID(dest, &models.Customer{}, existing.ID, id)
			continue
		}
		row := models.Customer{
			Code:         code,
			Name:         name,
			Email:        strings.TrimSpace(email),
			KycNumber:    code,
			EwuraLicense: lic,
			IsActive:     active,
			CreatedByID:  adminID,
			CreatedAt:    created,
			UpdatedAt:    created,
			DjangoID:     uint(id),
		}
		if err := dest.Create(&row).Error; err != nil {
			return fmt.Errorf("customer %s: %w", code, err)
		}
		applyLicenseContact(dest, &row)
		fmt.Printf("  + customer %s (django %d → %d)\n", code, id, row.ID)
	}
	return rows.Err()
}

func applyLicenseContact(dest *gorm.DB, row *models.Customer) {
	if row == nil || row.EwuraLicense == "" || row.EwuraLicense == "NA" {
		return
	}
	var lic models.EwuraPetroleumLicense
	if dest.Where("LicenseNumber = ?", row.EwuraLicense).First(&lic).Error != nil {
		return
	}
	updates := map[string]any{}
	if strings.TrimSpace(row.Email) == "" && strings.TrimSpace(lic.Email) != "" {
		updates["Email"] = strings.ToLower(strings.TrimSpace(lic.Email))
	}
	if strings.TrimSpace(row.TinNumber) == "" && strings.TrimSpace(lic.TinNumber) != "" {
		updates["TinNumber"] = strings.TrimSpace(lic.TinNumber)
	}
	if strings.TrimSpace(row.Phone) == "" && strings.TrimSpace(lic.Phone) != "" {
		updates["Phone"] = strings.TrimSpace(lic.Phone)
	}
	if len(updates) == 0 {
		return
	}
	_ = dest.Model(row).Updates(updates).Error
}

func copyCustomerDocs(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	return copyNamedAttachments(ctx, pg, dest, adminID, "customer docs",
		`SELECT id, customer_id, COALESCE(description,''), COALESCE(attachment,''),
		        COALESCE(extension,'pdf'), COALESCE(size,'0'), uploaded_by_id
		 FROM "SageCustomerDoc" ORDER BY id`,
		"django-customer-%d", "django-customer-%", types.CustomerContent, &models.Customer{})
}

func copyProducts(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	catID := petroleumCategoryID(dest, adminID)
	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(itemno,''), COALESCE(description,''), COALESCE(is_active,true)
		FROM "SageItem" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var itemNo, name string
		var active bool
		if err := rows.Scan(&id, &itemNo, &name, &active); err != nil {
			return err
		}
		itemNo = strings.ToUpper(strings.TrimSpace(itemNo))
		if itemNo == "" {
			continue
		}
		if idByDjango(dest, &models.Product{}, id) != 0 {
			continue
		}
		var existing models.Product
		q := dest.Where("Code = ?", itemNo)
		if itemNo == "1002" {
			q = dest.Where("Code IN ?", []string{itemNo, "AGO"})
		}
		if itemNo == "1001" {
			q = dest.Where("Code IN ?", []string{itemNo, "PMS", "MOGAS"})
		}
		if q.First(&existing).Error == nil {
			stampDjangoID(dest, &models.Product{}, existing.ID, id)
			if existing.Code != itemNo {
				_ = dest.Model(&existing).Update("Code", itemNo).Error
			}
			if existing.StockCategoryID == 0 && catID > 0 {
				_ = dest.Model(&existing).Update("StockCategoryID", catID).Error
			}
			continue
		}
		row := models.Product{
			Code:            itemNo,
			Name:            firstNonEmpty(name, itemNo),
			Unit:            "L",
			StockCategoryID: catID,
			IsActive:        active,
			CreatedByID:     adminID,
			DjangoID:        uint(id),
		}
		if err := dest.Create(&row).Error; err != nil {
			fmt.Printf("  skip product %s: %v\n", itemNo, err)
			continue
		}
		fmt.Printf("  + product %s (django %d → %d)\n", itemNo, id, row.ID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if catID > 0 {
		_ = dest.Model(&models.Product{}).Where("StockCategoryID <> ?", catID).Update("StockCategoryID", catID).Error
		_ = dest.Where("ID <> ?", catID).Delete(&models.StockCategory{}).Error
	}
	return nil
}

func petroleumCategoryID(dest *gorm.DB, adminID uint) uint {
	row := models.StockCategory{Name: "Petroleum products", CreatedByID: adminID, IsActive: true}
	if err := dest.Where(models.StockCategory{Name: row.Name}).Attrs(row).FirstOrCreate(&row).Error; err != nil {
		return 0
	}
	return row.ID
}
