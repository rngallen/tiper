package main

import (
	"strings"

	"dfms/apps/models"

	"gorm.io/gorm"
)

// idByDjango returns the new DFMS primary key for a row imported from Django.
//
// Django integer IDs are not reused. A parent (Customer, ILR, Truck, …) stores
// its old pk on DjangoID. A child that referenced that pk (customer_id,
// request_id, …) must look up DjangoID and write the new ID into the FK.
//
// After warmDjangoIDs / rememberDjango this is an in-memory map hit. Do not
// call this in a tight loop against a cold cache — that is one SELECT per row.
func idByDjango(dest *gorm.DB, model any, djangoID int64) uint {
	if djangoID <= 0 {
		return 0
	}
	key := typeKey(model)
	if id, ok := copyCache.getID(key, djangoID); ok {
		return id
	}
	if dest == nil {
		return 0
	}
	var id uint
	dest.Model(model).Select("ID").Where("DjangoID = ?", djangoID).Limit(1).Scan(&id)
	if id != 0 {
		copyCache.putID(key, djangoID, id)
	}
	return id
}

func stampDjangoID(dest *gorm.DB, model any, dfmsID uint, djangoID int64) {
	if dest == nil || dfmsID == 0 || djangoID <= 0 {
		return
	}
	_ = dest.Model(model).Where("ID = ? AND (DjangoID IS NULL OR DjangoID = 0)", dfmsID).
		Update("DjangoID", djangoID).Error
	rememberDjango(model, djangoID, dfmsID)
}

func userByDjango(dest *gorm.DB, adminID uint, djangoUserID int64) uint {
	if id := idByDjango(dest, &models.User{}, djangoUserID); id != 0 {
		return id
	}
	return adminID
}

func idByDjangoPtr(dest *gorm.DB, model any, djangoID *int64) uint {
	if djangoID == nil {
		return 0
	}
	return idByDjango(dest, model, *djangoID)
}

func destIDByName(dest *gorm.DB, name string) uint {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	copyCache.mu.RLock()
	if id, ok := copyCache.destByName[name]; ok {
		copyCache.mu.RUnlock()
		return id
	}
	copyCache.mu.RUnlock()
	if dest == nil {
		return 0
	}
	var row models.Destination
	if dest.Where("Name = ?", name).First(&row).Error != nil {
		return 0
	}
	copyCache.mu.Lock()
	copyCache.destByName[name] = row.ID
	copyCache.mu.Unlock()
	return row.ID
}

func countryCodeByName(dest *gorm.DB, name string) *string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	copyCache.mu.RLock()
	if code, ok := copyCache.countryByName[name]; ok {
		copyCache.mu.RUnlock()
		return &code
	}
	upper := strings.ToUpper(name)
	if code, ok := copyCache.countryByName[upper]; ok {
		copyCache.mu.RUnlock()
		return &code
	}
	copyCache.mu.RUnlock()
	if dest == nil {
		return nil
	}
	var row models.Country
	if dest.Where("Name = ? OR Code = ?", name, upper).First(&row).Error != nil {
		return nil
	}
	code := row.Code
	copyCache.mu.Lock()
	copyCache.countryByName[name] = code
	copyCache.countryByName[upper] = code
	copyCache.mu.Unlock()
	return &code
}

func districtIDByName(dest *gorm.DB, destinationID uint, name string) uint {
	name = strings.TrimSpace(name)
	if destinationID == 0 || name == "" {
		return 0
	}
	key := districtKey(destinationID, name)
	copyCache.mu.RLock()
	if id, ok := copyCache.districtByKey[key]; ok {
		copyCache.mu.RUnlock()
		return id
	}
	copyCache.mu.RUnlock()
	if dest == nil {
		return 0
	}
	var row models.District
	if dest.Where("DestinationID = ? AND Name = ?", destinationID, name).First(&row).Error != nil {
		return 0
	}
	copyCache.mu.Lock()
	copyCache.districtByKey[key] = row.ID
	copyCache.mu.Unlock()
	return row.ID
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n > 0 && len(s) > n {
		return s[:n]
	}
	return s
}

func normPlate(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func normSeal(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

func lookupCustomer(dest *gorm.DB, djangoID int64, fallbackCode, fallbackName string) (uint, string, string) {
	if row, ok := customerSnap(djangoID); ok {
		return row.ID, row.Code, firstNonEmpty(row.Name, fallbackName)
	}
	id := idByDjango(dest, &models.Customer{}, djangoID)
	if id == 0 {
		return 0, clip(fallbackCode, 20), clip(fallbackName, 160)
	}
	var row models.Customer
	if dest == nil || dest.First(&row, id).Error != nil {
		return id, clip(fallbackCode, 20), clip(fallbackName, 160)
	}
	return row.ID, row.Code, firstNonEmpty(row.Name, fallbackName)
}

func lookupProduct(dest *gorm.DB, djangoID int64, fallbackCode, fallbackName string) (uint, string, string) {
	if row, ok := productSnap(djangoID, fallbackCode); ok {
		return row.ID, row.Code, firstNonEmpty(row.Name, fallbackName)
	}
	id := idByDjango(dest, &models.Product{}, djangoID)
	if id == 0 && dest != nil && strings.TrimSpace(fallbackCode) != "" {
		var p models.Product
		if dest.Where("Code = ?", strings.ToUpper(strings.TrimSpace(fallbackCode))).First(&p).Error == nil {
			return p.ID, p.Code, p.Name
		}
	}
	if id == 0 {
		return 0, clip(fallbackCode, 20), clip(fallbackName, 120)
	}
	var row models.Product
	if dest == nil || dest.First(&row, id).Error != nil {
		return id, clip(fallbackCode, 20), clip(fallbackName, 120)
	}
	return row.ID, row.Code, firstNonEmpty(row.Name, fallbackName)
}

func lookupStatus(dest *gorm.DB, djangoID int64, fallbackName string) (uint, string, string) {
	if row, ok := statusSnap(djangoID); ok {
		return row.ID, row.Code, firstNonEmpty(row.Name, fallbackName)
	}
	id := idByDjango(dest, &models.StockStatus{}, djangoID)
	if id == 0 {
		return 0, "", clip(fallbackName, 80)
	}
	var row models.StockStatus
	if dest == nil || dest.First(&row, id).Error != nil {
		return id, "", clip(fallbackName, 80)
	}
	return row.ID, row.Code, firstNonEmpty(row.Name, fallbackName)
}
