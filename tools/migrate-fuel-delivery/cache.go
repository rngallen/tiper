package main

import (
	"fmt"
	"strings"
	"sync"

	"dfms/apps/models"

	"gorm.io/gorm"
)

// copyCache holds DjangoID → DFMS ID maps and small master snapshots so a
// 300k-row copy does not run a SELECT per foreign key. Stages stay serial
// (customers before ILR before ILO before comps); only lookups and inserts
// inside one table are concurrent.
type idCache struct {
	mu sync.RWMutex

	ids map[string]map[int64]uint // fmt %T of model pointer → django pk → DFMS ID

	customers      map[int64]namedRow
	productsDjango map[int64]namedRow
	productsCode   map[string]namedRow
	statuses       map[int64]namedRow

	ilr    map[uint]ilrLite
	trucks map[uint]truckLite

	destByName    map[string]uint
	districtByKey map[string]uint
	countryByName map[string]string
}

type namedRow struct {
	ID   uint
	Code string
	Name string
}

type ilrLite struct {
	ProductID     uint
	StockStatusID uint
}

type truckLite struct {
	PlateNumber string
	Trailer     string
	TrailerTwo  string
}

var copyCache = newIDCache()

func newIDCache() *idCache {
	return &idCache{
		ids:            make(map[string]map[int64]uint),
		customers:      make(map[int64]namedRow),
		productsDjango: make(map[int64]namedRow),
		productsCode:   make(map[string]namedRow),
		statuses:       make(map[int64]namedRow),
		ilr:            make(map[uint]ilrLite),
		trucks:         make(map[uint]truckLite),
		destByName:     make(map[string]uint),
		districtByKey:  make(map[string]uint),
		countryByName:  make(map[string]string),
	}
}

func resetIDCache() { copyCache = newIDCache() }

func typeKey(model any) string { return fmt.Sprintf("%T", model) }

func (c *idCache) getID(key string, djangoID int64) (uint, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.ids[key][djangoID]
	return id, ok
}

func (c *idCache) putID(key string, djangoID int64, dfmsID uint) {
	if djangoID <= 0 || dfmsID == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	m := c.ids[key]
	if m == nil {
		m = make(map[int64]uint)
		c.ids[key] = m
	}
	m[djangoID] = dfmsID
}

func rememberDjango(model any, djangoID int64, dfmsID uint) {
	copyCache.putID(typeKey(model), djangoID, dfmsID)
}

func rememberDjangoUint(model any, djangoID, dfmsID uint) {
	rememberDjango(model, int64(djangoID), dfmsID)
}

type djangoPair struct {
	ID       uint
	DjangoID uint
}

func warmDjangoIDs(dest *gorm.DB, model any) {
	if dest == nil {
		return
	}
	var rows []djangoPair
	dest.Model(model).Select("ID", "DjangoID").Where("DjangoID IS NOT NULL AND DjangoID > 0").Scan(&rows)
	key := typeKey(model)
	for _, r := range rows {
		copyCache.putID(key, int64(r.DjangoID), r.ID)
	}
}

func warmMasters(dest *gorm.DB) {
	if dest == nil {
		return
	}
	var customers []struct {
		ID, DjangoID uint
		Code, Name   string
	}
	dest.Model(&models.Customer{}).Select("ID", "Code", "Name", "DjangoID").Scan(&customers)
	copyCache.mu.Lock()
	for _, r := range customers {
		row := namedRow{ID: r.ID, Code: r.Code, Name: r.Name}
		if r.DjangoID > 0 {
			copyCache.customers[int64(r.DjangoID)] = row
		}
		copyCache.putIDLocked(typeKey(&models.Customer{}), int64(r.DjangoID), r.ID)
	}
	copyCache.mu.Unlock()

	var products []struct {
		ID, DjangoID uint
		Code, Name   string
	}
	dest.Model(&models.Product{}).Select("ID", "Code", "Name", "DjangoID").Scan(&products)
	copyCache.mu.Lock()
	for _, r := range products {
		row := namedRow{ID: r.ID, Code: r.Code, Name: r.Name}
		if r.DjangoID > 0 {
			copyCache.productsDjango[int64(r.DjangoID)] = row
		}
		if code := strings.ToUpper(strings.TrimSpace(r.Code)); code != "" {
			copyCache.productsCode[code] = row
		}
		copyCache.putIDLocked(typeKey(&models.Product{}), int64(r.DjangoID), r.ID)
	}
	copyCache.mu.Unlock()

	var statuses []struct {
		ID, DjangoID uint
		Code, Name   string
	}
	dest.Model(&models.StockStatus{}).Select("ID", "Code", "Name", "DjangoID").Scan(&statuses)
	copyCache.mu.Lock()
	for _, r := range statuses {
		row := namedRow{ID: r.ID, Code: r.Code, Name: r.Name}
		if r.DjangoID > 0 {
			copyCache.statuses[int64(r.DjangoID)] = row
		}
		copyCache.putIDLocked(typeKey(&models.StockStatus{}), int64(r.DjangoID), r.ID)
	}
	copyCache.mu.Unlock()
}

func (c *idCache) putIDLocked(key string, djangoID int64, dfmsID uint) {
	if djangoID <= 0 || dfmsID == 0 {
		return
	}
	m := c.ids[key]
	if m == nil {
		m = make(map[int64]uint)
		c.ids[key] = m
	}
	m[djangoID] = dfmsID
}

func warmPlaces(dest *gorm.DB) {
	if dest == nil {
		return
	}
	var dests []models.Destination
	dest.Select("ID", "Name", "DjangoID").Find(&dests)
	copyCache.mu.Lock()
	for _, d := range dests {
		name := strings.TrimSpace(d.Name)
		if name != "" {
			copyCache.destByName[name] = d.ID
		}
		copyCache.putIDLocked(typeKey(&models.Destination{}), int64(d.DjangoID), d.ID)
	}
	copyCache.mu.Unlock()

	var districts []models.District
	dest.Select("ID", "DestinationID", "Name", "DjangoID").Find(&districts)
	copyCache.mu.Lock()
	for _, d := range districts {
		copyCache.districtByKey[districtKey(d.DestinationID, d.Name)] = d.ID
		copyCache.putIDLocked(typeKey(&models.District{}), int64(d.DjangoID), d.ID)
	}
	copyCache.mu.Unlock()

	var countries []models.Country
	dest.Select("Code", "Name").Find(&countries)
	copyCache.mu.Lock()
	for _, c := range countries {
		copyCache.countryByName[strings.TrimSpace(c.Name)] = c.Code
		copyCache.countryByName[strings.ToUpper(strings.TrimSpace(c.Code))] = c.Code
	}
	copyCache.mu.Unlock()
}

func warmILR(dest *gorm.DB) {
	if dest == nil {
		return
	}
	var rows []struct {
		ID, ProductID, StockStatusID, DjangoID uint
	}
	dest.Model(&models.GantryLoadingRequest{}).Select("ID", "ProductID", "StockStatusID", "DjangoID").Scan(&rows)
	copyCache.mu.Lock()
	defer copyCache.mu.Unlock()
	for _, r := range rows {
		copyCache.ilr[r.ID] = ilrLite{ProductID: r.ProductID, StockStatusID: r.StockStatusID}
		copyCache.putIDLocked(typeKey(&models.GantryLoadingRequest{}), int64(r.DjangoID), r.ID)
	}
}

func warmTrucks(dest *gorm.DB) {
	if dest == nil {
		return
	}
	var rows []models.Truck
	dest.Select("ID", "PlateNumber", "Trailer", "TrailerTwo", "DjangoID").Find(&rows)
	copyCache.mu.Lock()
	defer copyCache.mu.Unlock()
	for _, t := range rows {
		copyCache.trucks[t.ID] = truckLite{PlateNumber: t.PlateNumber, Trailer: t.Trailer, TrailerTwo: t.TrailerTwo}
		copyCache.putIDLocked(typeKey(&models.Truck{}), int64(t.DjangoID), t.ID)
	}
}

func ilrProduct(dfmsID uint) (uint, bool) {
	row, ok := ilrFields(dfmsID)
	return row.ProductID, ok && row.ProductID != 0
}

func ilrFields(dfmsID uint) (ilrLite, bool) {
	copyCache.mu.RLock()
	defer copyCache.mu.RUnlock()
	row, ok := copyCache.ilr[dfmsID]
	return row, ok
}

func putILR(dfmsID, productID, statusID uint) {
	if dfmsID == 0 {
		return
	}
	copyCache.mu.Lock()
	copyCache.ilr[dfmsID] = ilrLite{ProductID: productID, StockStatusID: statusID}
	copyCache.mu.Unlock()
}

func putTruck(t models.Truck) {
	if t.ID == 0 {
		return
	}
	rememberDjangoUint(&models.Truck{}, t.DjangoID, t.ID)
	copyCache.mu.Lock()
	copyCache.trucks[t.ID] = truckLite{PlateNumber: t.PlateNumber, Trailer: t.Trailer, TrailerTwo: t.TrailerTwo}
	copyCache.mu.Unlock()
}

func truckPlates(dfmsID uint) (truckLite, bool) {
	copyCache.mu.RLock()
	defer copyCache.mu.RUnlock()
	t, ok := copyCache.trucks[dfmsID]
	return t, ok
}

// warmGantryParents loads DjangoID maps and small snapshots once before a
// large fact-table copy. Cheap tables are fully scanned; do this at the start
// of ILO / comps / loadings, not per row.
func warmGantryParents(dest *gorm.DB) {
	if dest == nil {
		return
	}
	warmMasters(dest)
	warmPlaces(dest)
	warmILR(dest)
	warmTrucks(dest)
	for _, m := range []any{
		&models.User{},
		&models.Transporter{},
		&models.Driver{},
		&models.TruckTank{},
		&models.TankCalibration{},
		&models.TankCompartment{},
		&models.RfidBadge{},
		&models.GantryLoadingLine{},
		&models.GantryLoadingRequest{},
		&models.GantryRequestVessel{},
		&models.Vessel{},
		&models.Compartmentalization{},
		&models.CompartmentalizationLine{},
		&models.GantryLoading{},
		&models.GantryVesselLoading{},
	} {
		warmDjangoIDs(dest, m)
	}
}

func districtKey(destinationID uint, name string) string {
	return fmt.Sprintf("%d\x00%s", destinationID, strings.TrimSpace(name))
}

func customerSnap(djangoID int64) (namedRow, bool) {
	copyCache.mu.RLock()
	defer copyCache.mu.RUnlock()
	row, ok := copyCache.customers[djangoID]
	return row, ok
}

func productSnap(djangoID int64, code string) (namedRow, bool) {
	copyCache.mu.RLock()
	defer copyCache.mu.RUnlock()
	if djangoID > 0 {
		if row, ok := copyCache.productsDjango[djangoID]; ok {
			return row, true
		}
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return namedRow{}, false
	}
	row, ok := copyCache.productsCode[code]
	return row, ok
}

func statusSnap(djangoID int64) (namedRow, bool) {
	copyCache.mu.RLock()
	defer copyCache.mu.RUnlock()
	row, ok := copyCache.statuses[djangoID]
	return row, ok
}

func warmStoredNames(dest *gorm.DB, like string) map[string]struct{} {
	out := map[string]struct{}{}
	if dest == nil || like == "" {
		return out
	}
	var names []string
	dest.Model(&models.Attachment{}).Where("StoredName LIKE ?", like).Pluck("StoredName", &names)
	for _, n := range names {
		out[n] = struct{}{}
	}
	return out
}
