package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func copyExtra(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	if err := copyDepots(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyDistricts(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyBadges(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyTanks(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyStorageTanks(ctx, pg, dest, adminID); err != nil {
		return err
	}
	return copyCalibrations(ctx, pg, dest)
}

func copySageStatuses(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	rows, err := pg.Query(ctx, `SELECT id, COALESCE(value,''), COALESCE(is_active,true) FROM "SageStatus" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var transit models.StockStatus
	_ = dest.Where("Code = ?", types.StockTransit).First(&transit).Error
	for rows.Next() {
		var id int64
		var name string
		var active bool
		if err := rows.Scan(&id, &name, &active); err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if idByDjango(dest, &models.StockStatus{}, id) != 0 {
			continue
		}
		var exists models.StockStatus
		if dest.Where("Name = ? OR Code = ?", name, models.SlugStatusCode(name)).First(&exists).Error == nil {
			stampDjangoID(dest, &models.StockStatus{}, exists.ID, id)
			continue
		}
		t, loc, mine, pror := classifyDjangoStatus(name)
		row := models.StockStatus{
			Code: models.SlugStatusCode(name), Name: name,
			IsTransit: t, IsLocal: loc, IsMining: mine, IsProration: pror,
			IsActive: active, DjangoID: uint(id),
		}
		if t && transit.ID > 0 && !strings.EqualFold(row.Code, string(types.StockTransit)) {
			row.ParentID = &transit.ID
		}
		if err := dest.Create(&row).Error; err != nil {
			fmt.Printf("  skip status %s: %v\n", name, err)
			continue
		}
		fmt.Printf("  + status %s (django %d → %d)\n", name, id, row.ID)
	}
	return rows.Err()
}

func copyDepots(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	cols := pgColumns(ctx, pg, "PipelineDepot")
	if len(cols) == 0 {
		cols = map[string]bool{
			"id": true, "name": true, "code": true,
			"ewura_licence": true, "is_active": true,
		}
	}
	nameCol := firstCol(cols, "name", "value", "description")
	if nameCol == "" {
		fmt.Printf("  skip depots: no name column on PipelineDepot\n")
		return nil
	}
	codeCol := firstCol(cols, "code", "depot_code")
	licCol := firstCol(cols, "ewura_licence", "ewura_license", "licence")
	activeCol := firstCol(cols, "is_active", "active")
	internalCol := firstCol(cols, "is_internal", "internal")
	codeSQL := "''"
	if codeCol != "" {
		codeSQL = "COALESCE(CAST(" + codeCol + " AS text),'')"
	}
	sel := fmt.Sprintf(
		`SELECT id, %s, COALESCE(%s,''), %s, %s, %s FROM "PipelineDepot" ORDER BY id`,
		codeSQL, nameCol, coalesceStrSQL(licCol), coalesceBoolSQL(activeCol), coalesceBoolInternal(internalCol),
	)
	rows, err := pg.Query(ctx, sel)
	if err != nil {
		fmt.Printf("  skip depots: %v\n", err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var code, name, lic string
		var active, internal bool
		if err := rows.Scan(&id, &code, &name, &lic, &active, &internal); err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if name == "" || idByDjango(dest, &models.Depot{}, id) != 0 {
			continue
		}
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" {
			code = fmt.Sprintf("%d", id+3000)
		}
		if len(code) > 20 {
			code = code[:20]
		}
		if strings.Contains(strings.ToUpper(name), "TIPER") {
			internal = true
		}
		var existing models.Depot
		if dest.Where("Code = ? OR Name = ?", code, name).First(&existing).Error == nil {
			stampDjangoID(dest, &models.Depot{}, existing.ID, id)
			if existing.Code == "" {
				_ = dest.Model(&existing).Update("Code", code).Error
			}
			continue
		}
		row := models.Depot{
			Name: name, Code: code, EwuraLicense: lic,
			IsInternal: internal, IsActive: active,
			CreatedByID: adminID, DjangoID: uint(id),
		}
		if err := dest.Create(&row).Error; err != nil {
			fmt.Printf("  skip depot %s: %v\n", name, err)
		} else {
			fmt.Printf("  + depot %s code %s (django %d → %d)\n", name, row.Code, id, row.ID)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	linkDefaultDepotCustomers(dest)
	return nil
}

func linkDefaultDepotCustomers(dest *gorm.DB) {
	var id uint
	dest.Model(&models.CustomerBillingAccount{}).
		Select("CustomerID").
		Where("FeeCode = ? AND IsActive = ?", types.FeeKOJ, true).
		Order("ID ASC").Limit(1).Scan(&id)
	if id == 0 {
		dest.Model(&models.Customer{}).Select("ID").Where("IsActive = ?", true).Order("ID ASC").Limit(1).Scan(&id)
	}
	if id == 0 {
		return
	}
	_ = dest.Model(&models.Depot{}).Where("CustomerID IS NULL").Update("CustomerID", id).Error
}

func copyDistricts(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.District{})
	warmDjangoIDs(dest, &models.Destination{})
	seen := map[string]uint{}
	var have []models.District
	dest.Select("ID", "DestinationID", "Name", "DjangoID").Find(&have)
	for _, r := range have {
		seen[districtKey(r.DestinationID, r.Name)] = r.ID
		rememberDjangoUint(&models.District{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `SELECT id, destination_id, COALESCE(name,'') FROM "GantryDistrict"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("districts", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.District) {
		for i := range batch {
			rememberDjangoUint(&models.District{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, destDjango int64
		var name string
		if err := rows.Scan(&id, &destDjango, &name); err != nil {
			w.close()
			return err
		}
		name = strings.TrimSpace(name)
		if name == "" || idByDjango(dest, &models.District{}, id) != 0 {
			continue
		}
		parentID := idByDjango(dest, &models.Destination{}, destDjango)
		if parentID == 0 {
			continue
		}
		key := districtKey(parentID, name)
		if existID, ok := seen[key]; ok {
			stampDjangoID(dest, &models.District{}, existID, id)
			continue
		}
		seen[key] = 1
		w.add(models.District{DestinationID: parentID, Name: name, IsActive: true, DjangoID: uint(id)})
	}
	w.close()
	return rows.Err()
}

func copyBadges(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.RfidBadge{})
	byCode := map[string]uint{}
	var have []models.RfidBadge
	dest.Select("ID", "Code", "DjangoID").Find(&have)
	for _, r := range have {
		byCode[r.Code] = r.ID
		rememberDjangoUint(&models.RfidBadge{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `SELECT id, code, COALESCE(is_active,true), COALESCE(is_available,true) FROM "GantryRfid"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("RFID badges", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.RfidBadge) {
		for i := range batch {
			rememberDjangoUint(&models.RfidBadge{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id int64
		var code int
		var active, avail bool
		if err := rows.Scan(&id, &code, &active, &avail); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.RfidBadge{}, id) != 0 {
			continue
		}
		tag := fmt.Sprintf("%d", code)
		if existID, ok := byCode[tag]; ok {
			stampDjangoID(dest, &models.RfidBadge{}, existID, id)
			continue
		}
		byCode[tag] = 1
		w.add(models.RfidBadge{Code: tag, IsActive: active, IsAvailable: avail, DjangoID: uint(id)})
	}
	w.close()
	return rows.Err()
}

func copyTanks(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.TruckTank{})
	warmDjangoIDs(dest, &models.Truck{})
	seen := map[string]uint{}
	var have []models.TruckTank
	dest.Select("ID", "TruckID", "PlateNumber", "[Index]", "DjangoID").Find(&have)
	for _, r := range have {
		seen[tankKey(derefUint(r.TruckID), r.PlateNumber, r.Index)] = r.ID
		rememberDjangoUint(&models.TruckTank{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, truck_id, COALESCE(plate_number,''), COALESCE(index,1), COALESCE(is_active,true)
		FROM "GantryTank"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("truck tanks", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.TruckTank) {
		for i := range batch {
			rememberDjangoUint(&models.TruckTank{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id int64
		var truckDjango *int64
		var plate string
		var idx int
		var active bool
		if err := rows.Scan(&id, &truckDjango, &plate, &idx, &active); err != nil {
			w.close()
			return err
		}
		plate = strings.TrimSpace(plate)
		if plate == "" || idByDjango(dest, &models.TruckTank{}, id) != 0 {
			continue
		}
		truckID := idByDjangoPtr(dest, &models.Truck{}, truckDjango)
		key := tankKey(truckID, plate, idx)
		if existID, ok := seen[key]; ok {
			stampDjangoID(dest, &models.TruckTank{}, existID, id)
			continue
		}
		seen[key] = 1
		w.add(models.TruckTank{TruckID: uintPtr(truckID), PlateNumber: plate, Index: idx, IsActive: active, DjangoID: uint(id)})
	}
	w.close()
	return rows.Err()
}

func tankKey(truckID uint, plate string, idx int) string {
	return fmt.Sprintf("%d:%s:%d", truckID, plate, idx)
}

func copyStorageTanks(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	table := storageTankTable(ctx, pg)
	if table == "" {
		return nil
	}
	cols := pgColumns(ctx, pg, table)
	codeCol := firstCol(cols, "code", "tank_code", "tankno")
	nameCol := firstCol(cols, "description", "name", "tank_name")
	activeCol := firstCol(cols, "is_active", "active")
	hasDataCol := firstCol(cols, "has_data", "hasdata")
	capCol := firstCol(cols, "capacity", "capacity_in_litre", "maximum_capacity", "max_capacity")
	deadCol := firstCol(cols, "dead_stock", "deadstock", "dead_stock_in_litre")
	productCol := firstCol(cols, "product_id", "item_id")
	if codeCol == "" {
		fmt.Printf("  skip storage tanks: no code column on %s\n", table)
		return nil
	}
	if nameCol == "" {
		nameCol = codeCol
	}
	sel := fmt.Sprintf(`SELECT id, COALESCE(%s,''), COALESCE(%s,''), %s, %s, %s, %s, %s FROM "%s"`,
		codeCol, nameCol,
		coalesceBoolSQL(activeCol),
		coalesceBoolSQL(hasDataCol),
		coalesceNumSQL(capCol),
		coalesceNumSQL(deadCol),
		nullIntSQL(productCol),
		table,
	)
	rows, err := pg.Query(ctx, sel)
	if err != nil {
		fmt.Printf("  skip storage tanks (%s): %v\n", table, err)
		return nil
	}
	defer rows.Close()
	warmDjangoIDs(dest, &models.Tank{})
	warmDjangoIDs(dest, &models.Product{})
	n := 0
	for rows.Next() {
		var id int64
		var code, name string
		var active, hasData bool
		var cap, dead float64
		var productDjango *int64
		if err := rows.Scan(&id, &code, &name, &active, &hasData, &cap, &dead, &productDjango); err != nil {
			return err
		}
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" || idByDjango(dest, &models.Tank{}, id) != 0 {
			continue
		}
		var existing models.Tank
		if dest.Where("Code = ?", code).First(&existing).Error == nil {
			stampDjangoID(dest, &models.Tank{}, existing.ID, id)
			continue
		}
		pid := idByDjangoPtr(dest, &models.Product{}, productDjango)
		if pid == 0 {
			fmt.Printf("  skip tank %s: no product\n", code)
			continue
		}
		capD := decimal.NewFromFloat(cap)
		deadD := decimal.NewFromFloat(dead)
		if !capD.GreaterThan(deadD) {
			if !capD.IsPositive() {
				fmt.Printf("  skip tank %s: capacity must be greater than dead stock\n", code)
				continue
			}
			fmt.Printf("  tank %s: dead stock %s >= capacity %s; storing dead stock as 0\n",
				code, deadD.String(), capD.String())
			deadD = decimal.Zero
		}
		row := models.Tank{
			Code:            code,
			Name:            firstNonEmpty(strings.TrimSpace(name), code),
			MaximumCapacity: capD,
			DeadStock:       deadD,
			ProductID:       pid,
			IsActive:        active,
			HasData:         hasData,
			CreatedByID:     adminID,
			DjangoID:        uint(id),
		}
		if err := dest.Create(&row).Error; err != nil {
			fmt.Printf("  skip tank %s: %v\n", code, err)
			continue
		}
		n++
	}
	if n > 0 {
		fmt.Printf("  + %d storage tanks from %s\n", n, table)
	}
	return rows.Err()
}

func storageTankTable(ctx context.Context, pg *pgx.Conn) string {
	for _, name := range []string{"PipelineDeliveryTank", "PipelienDeliveryTank"} {
		var n int
		if err := pg.QueryRow(ctx, `SELECT COUNT(*) FROM "`+name+`"`).Scan(&n); err == nil {
			return name
		}
	}
	return ""
}

func pgColumns(ctx context.Context, pg *pgx.Conn, table string) map[string]bool {
	out := map[string]bool{}
	rows, err := pg.Query(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_name = $1`, table)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			out[strings.ToLower(name)] = true
		}
	}
	return out
}

func firstCol(cols map[string]bool, names ...string) string {
	for _, n := range names {
		if cols[n] {
			return n
		}
	}
	return ""
}

func coalesceBoolSQL(col string) string {
	if col == "" {
		return "true"
	}
	return "COALESCE(" + col + ", true)"
}

func coalesceBoolInternal(col string) string {
	if col == "" {
		return "false"
	}
	return "COALESCE(" + col + ", false)"
}

func coalesceStrSQL(col string) string {
	if col == "" {
		return "''"
	}
	return "COALESCE(" + col + ",'')"
}

func coalesceNumSQL(col string) string {
	if col == "" {
		return "0"
	}
	return "COALESCE(" + col + ", 0)"
}

func nullIntSQL(col string) string {
	if col == "" {
		return "NULL"
	}
	return col
}

func derefUint(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

func copyCalibrations(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.TankCalibration{})
	warmDjangoIDs(dest, &models.TruckTank{})
	seen := map[string]uint{}
	var have []models.TankCalibration
	dest.Select("ID", "TankID", "ValidTo", "DjangoID").Find(&have)
	for _, r := range have {
		seen[fmt.Sprintf("%d:%s", r.TankID, r.ValidTo.Format("2006-01-02"))] = r.ID
		rememberDjangoUint(&models.TankCalibration{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, tank_id, COALESCE(is_active,true), COALESCE(certificate_end, CURRENT_DATE), created_at
		FROM "GantryTankCalibration" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("tank calibrations", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.TankCalibration) {
		for i := range batch {
			rememberDjangoUint(&models.TankCalibration{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, tankDjango int64
		var active bool
		var validTo, created time.Time
		if err := rows.Scan(&id, &tankDjango, &active, &validTo, &created); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.TankCalibration{}, id) != 0 {
			continue
		}
		tankID := idByDjango(dest, &models.TruckTank{}, tankDjango)
		if tankID == 0 {
			continue
		}
		key := fmt.Sprintf("%d:%s", tankID, validTo.Format("2006-01-02"))
		if existID, ok := seen[key]; ok {
			stampDjangoID(dest, &models.TankCalibration{}, existID, id)
			continue
		}
		seen[key] = 1
		w.add(models.TankCalibration{
			TankID: tankID, ValidFrom: created, ValidTo: validTo, IsActive: active, DjangoID: uint(id),
		})
	}
	w.close()
	return copyCompartments(ctx, pg, dest)
}

func copyCompartments(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.TankCompartment{})
	warmDjangoIDs(dest, &models.TankCalibration{})
	seen := map[string]struct{}{}
	var existing []models.TankCompartment
	dest.Select("CalibrationID", "[Index]").Find(&existing)
	for _, r := range existing {
		seen[fmt.Sprintf("%d:%d", r.CalibrationID, r.Index)] = struct{}{}
	}

	rows, err := pg.Query(ctx, `
		SELECT id, calibration_id, COALESCE(index,1), COALESCE(capacity,0)
		FROM "GantryCompartment"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("tank compartments", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.TankCompartment) {
		for i := range batch {
			rememberDjangoUint(&models.TankCompartment{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, calDjango int64
		var idx int
		var cap float64
		if err := rows.Scan(&id, &calDjango, &idx, &cap); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.TankCompartment{}, id) != 0 {
			continue
		}
		calID := idByDjango(dest, &models.TankCalibration{}, calDjango)
		if calID == 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", calID, idx)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		w.add(models.TankCompartment{
			CalibrationID: calID, Index: idx, Capacity: decimal.NewFromFloat(cap), DjangoID: uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func copyILO(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.GantryLoadingLine{})
	rows, err := pg.Query(ctx, `
		SELECT id, order_id, COALESCE(loaded,0), loaded_date, COALESCE(loading_status,1),
		       COALESCE(plate_number,''), COALESCE(destination_name,''), COALESCE(district_name,'')
		FROM "GantryIlo"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	type iloPatch struct {
		ID          uint
		LoadedQty   decimal.Decimal
		LoadedAt    *time.Time
		Status      types.OrderStatus
		HorsePlate  string
		Destination string
		District    string
	}
	w := newBatchWriter("ILO loaded qty", insertWorkers, insertBatch, func(batch []iloPatch) {
		for i := range batch {
			_ = dest.Model(&models.GantryLoadingLine{}).Where("ID = ?", batch[i].ID).Updates(map[string]any{
				"LoadedQty":   batch[i].LoadedQty,
				"LoadedAt":    batch[i].LoadedAt,
				"Status":      batch[i].Status,
				"HorsePlate":  batch[i].HorsePlate,
				"Destination": batch[i].Destination,
				"District":    batch[i].District,
			}).Error
		}
	})
	for rows.Next() {
		var id, orderDjango int64
		var loaded float64
		var loadedAt *time.Time
		var status int
		var horse, destName, district string
		if err := rows.Scan(&id, &orderDjango, &loaded, &loadedAt, &status, &horse, &destName, &district); err != nil {
			w.close()
			return err
		}
		lineID := idByDjango(dest, &models.GantryLoadingLine{}, orderDjango)
		if lineID == 0 {
			continue
		}
		w.add(iloPatch{
			ID: lineID, LoadedQty: decimal.NewFromFloat(loaded), LoadedAt: loadedAt,
			Status: mapLoadingStatus(status), HorsePlate: horse, Destination: destName, District: district,
		})
	}
	w.close()
	return rows.Err()
}

func copyCompartmentalizations(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	// Django GantryCompartmentalization → GantryCompartmentalization.
	// Snapshots (customer/product/status codes) are the DFMS master values, not
	// Django integer codes. ILO is GantryLoadingLine (order_id).
	//
	// One pgx reader maps rows (cache hits, NextTransactionID for zeros).
	// Bounded workers insert batches. Stages stay serial so parents exist.
	warmGantryParents(dest)
	var djangoMax uint
	_ = pg.QueryRow(ctx, `SELECT COALESCE(MAX(transaction_id),0) FROM "GantryCompartmentalization"`).Scan(&djangoMax)
	ensureTransactionFloor(dest, djangoMax)

	existing := map[string]uint{}
	var have []struct {
		ID             uint
		DocumentNumber string
		IloID          uint
	}
	dest.Model(&models.Compartmentalization{}).Select("ID", "DocumentNumber", "IloID").Scan(&have)
	for _, r := range have {
		existing[compDocKey(r.DocumentNumber, r.IloID)] = r.ID
	}

	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(transaction_id,0), COALESCE(loading_status,2),
		       COALESCE(printed,false), get_pass_date, badge_id, COALESCE(badge_code,0),
		       COALESCE(amended,false), COALESCE(is_active,true),
		       order_id, request_id, COALESCE(batch_number,''), COALESCE(doc_number,''),
		       COALESCE(customer_order_number,''), customer_id, COALESCE(customer_code,0),
		       COALESCE(customer_name,''), product_id, COALESCE(product_number,''),
		       COALESCE(product_description,''), COALESCE(quantity,0),
		       by_product_id, COALESCE(by_product_number,''), COALESCE(by_product_description,''),
		       COALESCE(by_product_quantity,0), product_status_id, COALESCE(product_status_code,0),
		       COALESCE(product_status_description,''), transporter_id, COALESCE(transporter_name,''),
		       driver_id, COALESCE(driver_name,''), COALESCE(driver_licence,''),
		       truck_id, COALESCE(plate_number,''), COALESCE(horse_plate_number,''),
		       trailer_one_id, COALESCE(trailer_one_plate_number,''),
		       trailer_two_id, COALESCE(trailer_two_plate_number,''),
		       order_date, loaded_date, COALESCE(loaded,0), COALESCE(by_product_loaded,0),
		       COALESCE(ewura_licence,''), COALESCE(destination_name,''), COALESCE(district_name,''),
		       expiration_date, created_at, created_by_id, sent_at, COALESCE(sent_to_ewura,false),
		       ewura_sent_at, COALESCE(file_name,'')
		FROM "GantryCompartmentalization" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("compartmentalizations", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.Compartmentalization) {
		for i := range batch {
			rememberDjangoUint(&models.Compartmentalization{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	skipped := 0
	for rows.Next() {
		var (
			id, transID, loadingStatus, badgeCode, custCode, statusCode, createdBy int64
			printed, amended, active, npgisSent                                    bool
			passAt, loadedAt, createdAt, almaSentAt, npgisAt                       *time.Time
			badgeDjango, byProdDjango, trailerTwoDjango                            *int64
			orderDjango, requestDjango, customerDjango, productDjango              int64
			statusDjango, transporterDjango, driverDjango, truckDjango             int64
			trailerOneDjango                                                       int64
			batch, doc, custOrder, custName, prodNo, prodName                      string
			byNo, byName, statusName, transporterName, driverName, driverLic       string
			plate, horse, t1, t2, ewuraLic, destName, district, almaFile           string
			qty, byQty, loaded, byLoaded                                           float64
			orderDate, expDate                                                     time.Time
		)
		if err := rows.Scan(&id, &transID, &loadingStatus, &printed, &passAt, &badgeDjango, &badgeCode,
			&amended, &active, &orderDjango, &requestDjango, &batch, &doc, &custOrder,
			&customerDjango, &custCode, &custName, &productDjango, &prodNo, &prodName, &qty,
			&byProdDjango, &byNo, &byName, &byQty, &statusDjango, &statusCode, &statusName,
			&transporterDjango, &transporterName, &driverDjango, &driverName, &driverLic,
			&truckDjango, &plate, &horse, &trailerOneDjango, &t1, &trailerTwoDjango, &t2,
			&orderDate, &loadedAt, &loaded, &byLoaded, &ewuraLic, &destName, &district,
			&expDate, &createdAt, &createdBy, &almaSentAt, &npgisSent, &npgisAt, &almaFile); err != nil {
			w.close()
			return err
		}
		if doc == "" || idByDjango(dest, &models.Compartmentalization{}, id) != 0 {
			continue
		}
		lineID := idByDjango(dest, &models.GantryLoadingLine{}, orderDjango)
		reqID := idByDjango(dest, &models.GantryLoadingRequest{}, requestDjango)
		custID, custCodeSnap, custNameSnap := lookupCustomer(dest, customerDjango, strconv.FormatInt(custCode, 10), custName)
		prodID, prodCodeSnap, prodNameSnap := lookupProduct(dest, productDjango, prodNo, prodName)
		statusID, statusCodeSnap, statusNameSnap := lookupStatus(dest, statusDjango, statusName)
		if lineID == 0 || reqID == 0 || custID == 0 || prodID == 0 || statusID == 0 {
			skipped++
			continue
		}
		key := compDocKey(doc, lineID)
		if existID, ok := existing[key]; ok {
			stampDjangoID(dest, &models.Compartmentalization{}, existID, id)
			continue
		}
		tid := uint(transID)
		if tid == 0 {
			n, err := models.NextTransactionID(dest)
			if err != nil {
				w.close()
				return err
			}
			tid = n
		}
		exp := expDate
		row := models.Compartmentalization{
			TransactionID:       tid,
			DocumentNumber:      clip(doc, 40),
			CustomerOrderNumber: clip(custOrder, 40),
			BatchNumber:         clip(batch, 30),
			IloID:               lineID,
			RequestID:           reqID,
			CustomerID:          custID,
			CustomerCode:        custCodeSnap,
			CustomerName:        custNameSnap,
			ProductID:           prodID,
			ProductCode:         prodCodeSnap,
			ProductName:         prodNameSnap,
			RequestedQty:        decimal.NewFromFloat(qty),
			ByProductQuantity:   decimal.NewFromFloat(byQty),
			StockStatusID:       statusID,
			StockStatusCode:     statusCodeSnap,
			StockStatusName:     statusNameSnap,
			TransporterID:       uintPtr(idByDjango(dest, &models.Transporter{}, transporterDjango)),
			TransporterName:     clip(transporterName, 180),
			DriverID:            uintPtr(idByDjango(dest, &models.Driver{}, driverDjango)),
			DriverName:          clip(driverName, 160),
			DriverLicense:       clip(driverLic, 40),
			TruckID:             uintPtr(idByDjango(dest, &models.Truck{}, truckDjango)),
			PlateNumber:         normPlate(plate),
			HorsePlate:          normPlate(horse),
			TrailerOneID:        uintPtr(idByDjango(dest, &models.TruckTank{}, trailerOneDjango)),
			TrailerOnePlate:     normPlate(t1),
			TrailerTwoID:        uintPtr(idByDjangoPtr(dest, &models.TruckTank{}, trailerTwoDjango)),
			TrailerTwoPlate:     normPlate(t2),
			BadgeID:             uintPtr(idByDjangoPtr(dest, &models.RfidBadge{}, badgeDjango)),
			BadgeCode:           clip(strconv.FormatInt(badgeCode, 10), 40),
			OrderDate:           orderDate,
			ExpirationDate:      &exp,
			EwuraLicense:        clip(ewuraLic, 40),
			Destination:         clip(destName, 160),
			District:            clip(district, 80),
			Printed:             printed,
			PrintedAt:           passAt,
			LoadedQty:           decimal.NewFromFloat(loaded),
			ByProductLoaded:     decimal.NewFromFloat(byLoaded),
			LoadedAt:            loadedAt,
			AlmaFileName:        clip(almaFile, 80),
			AlmaSentAt:          almaSentAt,
			NpgisSent:           npgisSent,
			NpgisSentAt:         npgisAt,
			Amended:             amended,
			IsActive:            active,
			Status:              mapLoadingStatus(int(loadingStatus)),
			CreatedByID:         userByDjango(dest, adminID, createdBy),
			DjangoID:            uint(id),
		}
		if byProdDjango != nil {
			if bid, code, name := lookupProduct(dest, *byProdDjango, byNo, byName); bid != 0 {
				row.ByProductID = &bid
				row.ByProductCode = code
				row.ByProductName = name
			}
		}
		if createdAt != nil {
			row.CreatedAt = *createdAt
			row.UpdatedAt = *createdAt
		}
		existing[key] = 1
		w.add(row)
	}
	w.close()
	if skipped > 0 {
		fmt.Printf("  skip %d compartmentalizations (parent django ids not imported)\n", skipped)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	bumpTransactionSequence(dest)
	if err := copyCompLines(ctx, pg, dest); err != nil {
		return err
	}
	return copyCompAttachments(ctx, pg, dest, adminID)
}

func compDocKey(doc string, iloID uint) string {
	return fmt.Sprintf("%s\x00%d", doc, iloID)
}

func copyCompLines(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.CompartmentalizationLine{})
	warmDjangoIDs(dest, &models.Compartmentalization{})
	warmDjangoIDs(dest, &models.TruckTank{})
	warmDjangoIDs(dest, &models.TankCalibration{})
	cells := map[string]struct{}{}
	var have []models.CompartmentalizationLine
	dest.Select("CompartmentalizationID", "TankID", "[Index]").Find(&have)
	for _, r := range have {
		cells[compCellKey(r.CompartmentalizationID, r.TankID, r.Index)] = struct{}{}
	}

	rows, err := pg.Query(ctx, `
		SELECT id, comp_id, tank_id, calibration_id, COALESCE(index,1),
		       COALESCE(capacity,0), COALESCE(quantity,0),
		       COALESCE(top_seal,''), COALESCE(dip_seal,''), COALESCE(bottom_seal,''),
		       product_id, COALESCE(product_number,''), COALESCE(product_description,''),
		       COALESCE(tank_plate_number,'')
		FROM "GantryCompartmentalizationLine"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("comp lines", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.CompartmentalizationLine) {
		for i := range batch {
			rememberDjangoUint(&models.CompartmentalizationLine{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, compDjango, tankDjango, calDjango int64
		var productDjango *int64
		var idx int
		var cap, qty float64
		var top, dip, bottom, plate, prodNo, prodName string
		if err := rows.Scan(&id, &compDjango, &tankDjango, &calDjango, &idx, &cap, &qty, &top, &dip, &bottom, &productDjango, &prodNo, &prodName, &plate); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.CompartmentalizationLine{}, id) != 0 {
			continue
		}
		compID := idByDjango(dest, &models.Compartmentalization{}, compDjango)
		tankID := idByDjango(dest, &models.TruckTank{}, tankDjango)
		calID := idByDjango(dest, &models.TankCalibration{}, calDjango)
		if compID == 0 || tankID == 0 || calID == 0 {
			continue
		}
		ck := compCellKey(compID, tankID, idx)
		if _, ok := cells[ck]; ok {
			continue
		}
		var pid *uint
		prodCode, prodNameSnap := prodNo, prodName
		if productDjango != nil {
			if mapped, code, name := lookupProduct(dest, *productDjango, prodNo, prodName); mapped != 0 {
				pid = &mapped
				prodCode, prodNameSnap = code, name
			}
		}
		cells[ck] = struct{}{}
		w.add(models.CompartmentalizationLine{
			CompartmentalizationID: compID,
			CalibrationID:          calID,
			TankID:                 tankID,
			TankPlate:              normPlate(plate),
			Index:                  idx,
			Capacity:               decimal.NewFromFloat(cap),
			ProductID:              pid,
			ProductCode:            prodCode,
			ProductName:            prodNameSnap,
			Quantity:               decimal.NewFromFloat(qty),
			TopSeal:                normSeal(top),
			DipSeal:                normSeal(dip),
			BottomSeal:             normSeal(bottom),
			DjangoID:               uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func compCellKey(compID, tankID uint, idx int) string {
	return fmt.Sprintf("%d:%d:%d", compID, tankID, idx)
}

func copyLoadings(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmGantryParents(dest)
	warmDjangoIDs(dest, &models.GantryLoading{})
	haveComp := map[uint]uint{}
	var have []struct {
		ID                     uint
		CompartmentalizationID uint
	}
	dest.Model(&models.GantryLoading{}).Select("ID", "CompartmentalizationID").Scan(&have)
	for _, r := range have {
		haveComp[r.CompartmentalizationID] = r.ID
	}

	rows, err := pg.Query(ctx, `
		SELECT id, comp_id, order_id, request_id, loaded_date, COALESCE(doc_number,''),
		       COALESCE(customer_order_number,''), COALESCE(batch_number,''), order_date,
		       COALESCE(requested_quantity,0), badge_id, COALESCE(badge_code,0),
		       customer_id, COALESCE(customer_code,0), COALESCE(customer_name,''),
		       product_status_id, COALESCE(product_status_code,0), COALESCE(product_status_description,''),
		       transporter_id, COALESCE(transporter_name,''), driver_id, COALESCE(driver_name,''),
		       COALESCE(driver_licence,''), truck_id, COALESCE(plate_number,''),
		       COALESCE(ewura_licence,''), COALESCE(destination_name,''), COALESCE(district_name,''),
		       expiration_date,
		       ago_id, COALESCE(ago_number,''), COALESCE(ago_description,''),
		       COALESCE(ago_quantity,0), COALESCE(ago_standard_volume,0),
		       COALESCE(ago_temperature,0), COALESCE(ago_density,0), COALESCE(ago_wcf,0), COALESCE(ago_weight,0),
		       mogas_id, COALESCE(mogas_number,''), COALESCE(mogas_description,''),
		       COALESCE(mogas_quantity,0), COALESCE(mogas_standard_volume,0),
		       COALESCE(mogas_temperature,0), COALESCE(mogas_density,0), COALESCE(mogas_wcf,0), COALESCE(mogas_weight,0)
		FROM "GantryLoading" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("gantry loadings", insertWorkers, insertBatch, insertLoadings(dest))
	for rows.Next() {
		var (
			id, compDjango, orderDjango, requestDjango, custDjango, statusDjango int64
			transporterDjango, driverDjango, truckDjango                         int64
			badgeDjango, agoDjango, pmsDjango                                    *int64
			loadedAt, orderDate, expDate                                         time.Time
			doc, custOrder, batch, custName, statusName, transporterName         string
			driverName, driverLic, plate, ewuraLic, destName, district           string
			agoNo, agoName, pmsNo, pmsName                                       string
			badgeCode, custCode, statusCode                                      int64
			reqQty, agoQty, agoStd, agoTemp, agoDen, agoWcf, agoWt               float64
			pmsQty, pmsStd, pmsTemp, pmsDen, pmsWcf, pmsWt                       float64
		)
		if err := rows.Scan(&id, &compDjango, &orderDjango, &requestDjango, &loadedAt, &doc, &custOrder, &batch,
			&orderDate, &reqQty, &badgeDjango, &badgeCode, &custDjango, &custCode, &custName,
			&statusDjango, &statusCode, &statusName, &transporterDjango, &transporterName,
			&driverDjango, &driverName, &driverLic, &truckDjango, &plate, &ewuraLic, &destName, &district, &expDate,
			&agoDjango, &agoNo, &agoName, &agoQty, &agoStd, &agoTemp, &agoDen, &agoWcf, &agoWt,
			&pmsDjango, &pmsNo, &pmsName, &pmsQty, &pmsStd, &pmsTemp, &pmsDen, &pmsWcf, &pmsWt); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.GantryLoading{}, id) != 0 {
			continue
		}
		compID := idByDjango(dest, &models.Compartmentalization{}, compDjango)
		lineID := idByDjango(dest, &models.GantryLoadingLine{}, orderDjango)
		reqID := idByDjango(dest, &models.GantryLoadingRequest{}, requestDjango)
		custID, custCodeSnap, custNameSnap := lookupCustomer(dest, custDjango, strconv.FormatInt(custCode, 10), custName)
		statusID, statusCodeSnap, statusNameSnap := lookupStatus(dest, statusDjango, statusName)
		if compID == 0 || lineID == 0 || reqID == 0 || custID == 0 || statusID == 0 {
			continue
		}
		if existID, ok := haveComp[compID]; ok {
			stampDjangoID(dest, &models.GantryLoading{}, existID, id)
			continue
		}
		exp := expDate
		load := models.GantryLoading{
			CompartmentalizationID: compID,
			IloID:                  lineID,
			RequestID:              reqID,
			DocumentNumber:         clip(doc, 40),
			CustomerOrderNumber:    clip(custOrder, 40),
			BatchNumber:            clip(batch, 30),
			OrderDate:              orderDate,
			LoadedAt:               loadedAt,
			Year:                   loadedAt.Year(),
			Month:                  int(loadedAt.Month()),
			RequestedQty:           decimal.NewFromFloat(reqQty),
			BadgeID:                uintPtr(idByDjangoPtr(dest, &models.RfidBadge{}, badgeDjango)),
			BadgeCode:              clip(intCode(badgeCode), 40),
			CustomerID:             custID,
			CustomerCode:           custCodeSnap,
			CustomerName:           custNameSnap,
			StockStatusID:          statusID,
			StockStatusCode:        statusCodeSnap,
			StockStatusName:        statusNameSnap,
			TransporterID:          uintPtr(idByDjango(dest, &models.Transporter{}, transporterDjango)),
			TransporterName:        clip(transporterName, 180),
			DriverID:               uintPtr(idByDjango(dest, &models.Driver{}, driverDjango)),
			DriverName:             clip(driverName, 160),
			DriverLicense:          clip(driverLic, 40),
			TruckID:                uintPtr(idByDjango(dest, &models.Truck{}, truckDjango)),
			PlateNumber:            normPlate(plate),
			EwuraLicense:           clip(ewuraLic, 40),
			Destination:            clip(destName, 160),
			District:               clip(district, 80),
			ExpirationDate:         &exp,
			DjangoID:               uint(id),
		}
		item := loadingInsert{header: load}
		item.prods = appendLoadProduct(item.prods, dest, agoDjango, agoNo, agoName, agoQty, agoStd, agoTemp, agoDen, agoWcf, agoWt)
		item.prods = appendLoadProduct(item.prods, dest, pmsDjango, pmsNo, pmsName, pmsQty, pmsStd, pmsTemp, pmsDen, pmsWcf, pmsWt)
		haveComp[compID] = 1
		w.add(item)
	}
	w.close()
	return rows.Err()
}

type loadingInsert struct {
	header models.GantryLoading
	prods  []models.GantryLoadingProduct
}

func insertLoadings(dest *gorm.DB) func([]loadingInsert) {
	return func(batch []loadingInsert) {
		if dest == nil || len(batch) == 0 {
			return
		}
		headers := make([]models.GantryLoading, len(batch))
		for i := range batch {
			headers[i] = batch[i].header
			headers[i].Products = nil
		}
		if err := dest.CreateInBatches(headers, len(headers)).Error; err != nil {
			for i := range batch {
				row := batch[i].header
				row.Products = batch[i].prods
				if err := dest.Create(&row).Error; err != nil {
					continue
				}
				rememberDjangoUint(&models.GantryLoading{}, row.DjangoID, row.ID)
			}
			return
		}
		var prods []models.GantryLoadingProduct
		for i := range headers {
			rememberDjangoUint(&models.GantryLoading{}, headers[i].DjangoID, headers[i].ID)
			for _, p := range batch[i].prods {
				p.LoadingID = headers[i].ID
				prods = append(prods, p)
			}
		}
		if len(prods) == 0 {
			return
		}
		if err := dest.CreateInBatches(prods, len(prods)).Error; err != nil {
			for i := range prods {
				_ = dest.Create(&prods[i]).Error
			}
		}
	}
}

func appendLoadProduct(dst []models.GantryLoadingProduct, dest *gorm.DB, djangoID *int64, code, name string, obs, std, temp, den, wcf, wt float64) []models.GantryLoadingProduct {
	if std == 0 && obs == 0 {
		return dst
	}
	var django int64
	if djangoID != nil {
		django = *djangoID
	}
	pid, prodCode, prodName := lookupProduct(dest, django, code, name)
	if pid == 0 {
		return dst
	}
	dens := decimal.NewFromFloat(den)
	w := decimal.NewFromFloat(wcf)
	if w.IsZero() {
		w = models.LoadingWCF(dens)
	}
	weight := decimal.NewFromFloat(wt)
	stdD := decimal.NewFromFloat(std)
	if weight.IsZero() {
		weight = models.LoadingWeight(w, stdD)
	}
	return append(dst, models.GantryLoadingProduct{
		ProductID:      pid,
		ProductCode:    prodCode,
		ProductName:    prodName,
		ObservedVolume: decimal.NewFromFloat(obs),
		StandardVolume: stdD,
		Temperature:    decimal.NewFromFloat(temp),
		Density:        dens,
		WCF:            w,
		Weight:         weight,
	})
}

func copyVesselLoadings(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmGantryParents(dest)
	warmDjangoIDs(dest, &models.GantryVesselLoading{})
	rows, err := pg.Query(ctx, `
		SELECT id, loading_vessel_id, vessel_id, vessel_date, COALESCE(vessel_description,''),
		       product_id, COALESCE(product_number,''), COALESCE(product_description,''),
		       COALESCE(quantity,0), COALESCE(density,0), COALESCE(temperature,0),
		       COALESCE(standard_volume,0), COALESCE(wcf,0), COALESCE(weight,0),
		       COALESCE(financial_hold,false), product_status_id, COALESCE(product_status_code,0),
		       COALESCE(product_status_description,''), comp_id, order_id, request_id, doc_date,
		       badge_id, COALESCE(badge_code,0), COALESCE(doc_number,''), COALESCE(customer_order_number,''),
		       COALESCE(batch_number,''), customer_id, COALESCE(customer_code,0), COALESCE(customer_name,''),
		       transporter_id, COALESCE(transporter_name,''), driver_id, COALESCE(driver_name,''),
		       COALESCE(driver_licence,''), truck_id, COALESCE(plate_number,''), order_date,
		       COALESCE(ewura_licence,''), COALESCE(destination_name,''), COALESCE(district_name,''),
		       expiration_date
		FROM "GantryVesselLoading" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := newBatchWriter("vessel loadings", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.GantryVesselLoading) {
		for i := range batch {
			rememberDjangoUint(&models.GantryVesselLoading{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var (
			id, parcelDjango, vesselDjango, productDjango, statusDjango, compDjango int64
			orderDjango, requestDjango, custDjango, transporterDjango, driverDjango int64
			truckDjango                                                             int64
			badgeDjango                                                             *int64
			vesselDate, loadedAt, orderDate, expDate                                time.Time
			vesselName, prodNo, prodName, statusName, doc, custOrder, batch         string
			custName, transporterName, driverName, driverLic, plate                 string
			ewuraLic, destName, district                                            string
			qty, dens, temp, std, wcf, wt                                           float64
			hold                                                                    bool
			statusCode, badgeCode, custCode                                         int64
		)
		if err := rows.Scan(&id, &parcelDjango, &vesselDjango, &vesselDate, &vesselName, &productDjango, &prodNo, &prodName,
			&qty, &dens, &temp, &std, &wcf, &wt, &hold, &statusDjango, &statusCode, &statusName,
			&compDjango, &orderDjango, &requestDjango, &loadedAt, &badgeDjango, &badgeCode, &doc, &custOrder, &batch,
			&custDjango, &custCode, &custName, &transporterDjango, &transporterName, &driverDjango, &driverName, &driverLic,
			&truckDjango, &plate, &orderDate, &ewuraLic, &destName, &district, &expDate); err != nil {
			out.close()
			return err
		}
		if idByDjango(dest, &models.GantryVesselLoading{}, id) != 0 {
			continue
		}
		parcelID := idByDjango(dest, &models.GantryRequestVessel{}, parcelDjango)
		vesselID := idByDjango(dest, &models.Vessel{}, vesselDjango)
		prodID, prodCode, prodNameSnap := lookupProduct(dest, productDjango, prodNo, prodName)
		statusID, statusCodeSnap, statusNameSnap := lookupStatus(dest, statusDjango, statusName)
		compID := idByDjango(dest, &models.Compartmentalization{}, compDjango)
		lineID := idByDjango(dest, &models.GantryLoadingLine{}, orderDjango)
		reqID := idByDjango(dest, &models.GantryLoadingRequest{}, requestDjango)
		custID, custCodeSnap, custNameSnap := lookupCustomer(dest, custDjango, strconv.FormatInt(custCode, 10), custName)
		if parcelID == 0 || vesselID == 0 || prodID == 0 || statusID == 0 || compID == 0 || lineID == 0 || reqID == 0 || custID == 0 {
			continue
		}
		densD := decimal.NewFromFloat(dens)
		wcfD := decimal.NewFromFloat(wcf)
		if wcfD.IsZero() {
			wcfD = models.LoadingWCF(densD)
		}
		stdD := decimal.NewFromFloat(std)
		weight := decimal.NewFromFloat(wt)
		if weight.IsZero() {
			weight = models.LoadingWeight(wcfD, stdD)
		}
		exp := expDate
		row := models.GantryVesselLoading{
			RequestVesselID:        parcelID,
			VesselID:               vesselID,
			VesselDate:             vesselDate,
			VesselName:             clip(vesselName, 120),
			ProductID:              prodID,
			ProductCode:            prodCode,
			ProductName:            prodNameSnap,
			Quantity:               decimal.NewFromFloat(qty),
			Density:                densD,
			Temperature:            decimal.NewFromFloat(temp),
			StandardVolume:         stdD,
			WCF:                    wcfD,
			Weight:                 weight,
			FinancialHold:          hold,
			StockStatusID:          statusID,
			StockStatusCode:        statusCodeSnap,
			StockStatusName:        statusNameSnap,
			CompartmentalizationID: compID,
			IloID:                  lineID,
			RequestID:              reqID,
			LoadedAt:               loadedAt,
			BadgeID:                uintPtr(idByDjangoPtr(dest, &models.RfidBadge{}, badgeDjango)),
			BadgeCode:              clip(intCode(badgeCode), 40),
			DocumentNumber:         clip(doc, 40),
			CustomerOrderNumber:    clip(custOrder, 40),
			BatchNumber:            clip(batch, 30),
			CustomerID:             custID,
			CustomerCode:           custCodeSnap,
			CustomerName:           custNameSnap,
			TransporterID:          uintPtr(idByDjango(dest, &models.Transporter{}, transporterDjango)),
			TransporterName:        clip(transporterName, 180),
			DriverID:               uintPtr(idByDjango(dest, &models.Driver{}, driverDjango)),
			DriverName:             clip(driverName, 160),
			DriverLicense:          clip(driverLic, 40),
			TruckID:                uintPtr(idByDjango(dest, &models.Truck{}, truckDjango)),
			PlateNumber:            normPlate(plate),
			OrderDate:              orderDate,
			EwuraLicense:           ewuraLic,
			Destination:            destName,
			District:               district,
			ExpirationDate:         &exp,
			DjangoID:               uint(id),
		}
		out.add(row)
	}
	out.close()
	return rows.Err()
}

func copyCompAttachments(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	return copyNamedAttachments(ctx, pg, dest, adminID, "comp attachments",
		`SELECT id, comp_id, COALESCE(description,''), COALESCE(attachment,''),
		        COALESCE(extension,'pdf'), COALESCE(size,'0'), uploaded_by_id
		 FROM "GantryCompartmentalizationAttachment" ORDER BY id`,
		"django-comp-%d", "django-comp-%", types.CompartmentalizationContent, &models.Compartmentalization{})
}

func copyNamedAttachments(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint, label, query, storedFmt, like string, entityType types.ContentType, parent any) error {
	seen := warmStoredNames(dest, like)
	warmDjangoIDs(dest, parent)
	warmDjangoIDs(dest, &models.User{})
	rows, err := pg.Query(ctx, query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter(label, insertWorkers, insertBatch, gormInsert(dest, func(batch []models.Attachment) {
		for i := range batch {
			rememberDjangoUint(&models.Attachment{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, parentDjango, uploadedBy int64
		var desc, path, ext, size string
		if err := rows.Scan(&id, &parentDjango, &desc, &path, &ext, &size, &uploadedBy); err != nil {
			w.close()
			return err
		}
		stored := fmt.Sprintf(storedFmt, id)
		if _, ok := seen[stored]; ok {
			continue
		}
		parentID := idByDjango(dest, parent, parentDjango)
		if parentID == 0 {
			continue
		}
		seen[stored] = struct{}{}
		w.add(models.Attachment{
			OriginalName: firstNonEmpty(desc, path, stored),
			StoredName:   stored,
			FilePath:     path,
			EntityID:     parentID,
			EntityType:   entityType,
			Extension:    ext,
			ByteSize:     size,
			UploadedByID: userByDjango(dest, adminID, uploadedBy),
			DjangoID:     uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func bumpTransactionSequence(dest *gorm.DB) {
	var max uint
	dest.Model(&models.Compartmentalization{}).Select("COALESCE(MAX(TransactionID),0)").Scan(&max)
	ensureTransactionFloor(dest, max)
}

func ensureTransactionFloor(dest *gorm.DB, floor uint) {
	if dest == nil || floor == 0 {
		return
	}
	var row models.TransactionSequence
	_ = dest.FirstOrCreate(&row, models.TransactionSequence{ID: 1}).Error
	if floor > row.LastValue {
		_ = dest.Model(&models.TransactionSequence{}).Where("ID = 1").Update("LastValue", floor).Error
	}
}

func uintPtr(id uint) *uint {
	if id == 0 {
		return nil
	}
	return &id
}

func intCode(n int64) string {
	if n == 0 {
		return ""
	}
	return strconv.FormatInt(n, 10)
}

func copyAmendments(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	warmDjangoIDs(dest, &models.OrderAmendment{})
	warmDjangoIDs(dest, &models.GantryLoadingLine{})
	byDoc := map[string]uint{}
	var have []models.OrderAmendment
	dest.Select("ID", "DocumentNumber", "DjangoID").Find(&have)
	for _, r := range have {
		byDoc[r.DocumentNumber] = r.ID
		rememberDjangoUint(&models.OrderAmendment{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, order_id, COALESCE(doc_number,''), COALESCE(amendment,1), COALESCE(quantity,0),
		       product_id, expiration_date, COALESCE(destination_name,''), COALESCE(district_name,''),
		       COALESCE(plate_number,''), COALESCE(transporter_name,''), COALESCE(driver_name,''),
		       created_by_id
		FROM "GantryAmendment" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("amendments", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.OrderAmendment) {
		for i := range batch {
			rememberDjangoUint(&models.OrderAmendment{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, orderDjango, createdBy int64
		var productDjango *int64
		var doc string
		var kind int
		var qty float64
		var exp *time.Time
		var destName, district, plate, transporter, driver string
		if err := rows.Scan(&id, &orderDjango, &doc, &kind, &qty, &productDjango, &exp, &destName, &district, &plate, &transporter, &driver, &createdBy); err != nil {
			w.close()
			return err
		}
		if doc == "" || idByDjango(dest, &models.OrderAmendment{}, id) != 0 {
			continue
		}
		lineID := idByDjango(dest, &models.GantryLoadingLine{}, orderDjango)
		if lineID == 0 {
			continue
		}
		if existID, ok := byDoc[doc]; ok {
			stampDjangoID(dest, &models.OrderAmendment{}, existID, id)
			continue
		}
		var pid *uint
		if productDjango != nil {
			if mapped := idByDjango(dest, &models.Product{}, *productDjango); mapped != 0 {
				pid = &mapped
			}
		}
		byDoc[doc] = 1
		w.add(models.OrderAmendment{
			DocumentNumber:  doc,
			Kind:            mapAmendmentKind(kind),
			IloID:           lineID,
			RequestedQty:    decimal.NewFromFloat(qty),
			ProductID:       pid,
			ExpirationDate:  exp,
			Destination:     destName,
			District:        district,
			TruckPlate:      plate,
			TransporterName: transporter,
			DriverName:      driver,
			Status:          types.OrderApproved,
			CreatedByID:     userByDjango(dest, adminID, createdBy),
			DjangoID:        uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func copyILRAttachments(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	return copyNamedAttachments(ctx, pg, dest, adminID, "ILR attachments",
		`SELECT id, request_id, COALESCE(description,''), COALESCE(attachment,''),
		        COALESCE(extension,'pdf'), COALESCE(size,'0'), uploaded_by_id
		 FROM "GantryIlrAttachment" ORDER BY id`,
		"django-ilr-%d", "django-ilr-%", types.GantryLoadingRequestContent, &models.GantryLoadingRequest{})
}

func mapAmendmentKind(n int) types.AmendmentKind {
	switch n {
	case 2:
		return types.AmendQtyIncrease
	case 3:
		return types.AmendQtyDecrease
	case 4:
		return types.AmendProduct
	case 5:
		return types.AmendCancel
	case 6:
		return types.AmendBatchCancel
	case 7:
		return types.AmendExtend
	default:
		return types.AmendNormal
	}
}

func classifyDjangoStatus(name string) (transit, local, mining, proration bool) {
	transit, local, mining, proration = models.ClassifyStockStatus(name)
	if transit || mining || proration {
		return
	}
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "CONGO", "DRC", "DR CONGO", "CONGO DRC", "RWANDA", "BURUNDI",
		"MALAWI", "ZAMBIA", "UGANDA", "KENYA", "SOUTH SUDAN", "TRANSIT":
		return true, false, false, false
	default:
		return
	}
}

func copyTransporters(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.Transporter{})
	byName := map[string]uint{}
	var have []models.Transporter
	dest.Select("ID", "Name", "DjangoID").Find(&have)
	for _, r := range have {
		byName[r.Name] = r.ID
		rememberDjangoUint(&models.Transporter{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, name, COALESCE(phone,''), COALESCE(email,''), COALESCE(contact_person,''),
		       COALESCE(tin_no,''), COALESCE(vrn_no,''), COALESCE(country_name,''),
		       COALESCE(address1,''), COALESCE(address2,''), COALESCE(address3,''),
		       COALESCE(address4,''), COALESCE(address5,''),
		       aeo_end_date, COALESCE(is_active,true), COALESCE(have_data,false)
		FROM "GantryTransporter"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("transporters", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.Transporter) {
		for i := range batch {
			rememberDjangoUint(&models.Transporter{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id int64
		var name, phone, email, contact, tin, vrn, country, a1, a2, a3, a4, a5 string
		var aeo *time.Time
		var active, hasData bool
		if err := rows.Scan(&id, &name, &phone, &email, &contact, &tin, &vrn, &country, &a1, &a2, &a3, &a4, &a5, &aeo, &active, &hasData); err != nil {
			w.close()
			return err
		}
		name = strings.TrimSpace(name)
		if name == "" || idByDjango(dest, &models.Transporter{}, id) != 0 {
			continue
		}
		if existID, ok := byName[name]; ok {
			stampDjangoID(dest, &models.Transporter{}, existID, id)
			continue
		}
		byName[name] = 1
		w.add(models.Transporter{
			Name: name, Phone: strings.TrimSpace(phone), Email: strings.TrimSpace(email),
			ContactPerson: strings.TrimSpace(contact), TinNumber: strings.TrimSpace(tin),
			VrnNumber: strings.TrimSpace(vrn), Address: strings.TrimSpace(a1), Address2: joinNonEmpty(a2, a3, a4, a5),
			CountryCode: countryCodeByName(dest, country), AeoEndDate: aeo,
			IsActive: active, HasData: hasData, DjangoID: uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func joinNonEmpty(parts ...string) string {
	var out []string
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, ", ")
}

func copyDrivers(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.Driver{})
	byLic := map[string]uint{}
	var have []models.Driver
	dest.Select("ID", "LicenseNumber", "DjangoID").Find(&have)
	for _, r := range have {
		byLic[r.LicenseNumber] = r.ID
		rememberDjangoUint(&models.Driver{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, name, COALESCE(driver_licence,''), licence_expire_date,
		       COALESCE(phone_number,''), COALESCE(email,''), COALESCE(is_active,true),
		       COALESCE(have_data,false)
		FROM "GantryDriver"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("drivers", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.Driver) {
		for i := range batch {
			rememberDjangoUint(&models.Driver{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id int64
		var name, lic, phone, email string
		var exp *time.Time
		var active, hasData bool
		if err := rows.Scan(&id, &name, &lic, &exp, &phone, &email, &active, &hasData); err != nil {
			w.close()
			return err
		}
		name = strings.TrimSpace(name)
		lic = strings.TrimSpace(lic)
		if name == "" || idByDjango(dest, &models.Driver{}, id) != 0 {
			continue
		}
		if lic == "" {
			lic = fmt.Sprintf("DRV-%d", id)
		}
		if existID, ok := byLic[lic]; ok {
			stampDjangoID(dest, &models.Driver{}, existID, id)
			continue
		}
		byLic[lic] = 1
		w.add(models.Driver{
			Name: name, LicenseNumber: lic, LicenseExpires: exp,
			Phone: strings.TrimSpace(phone), Email: strings.TrimSpace(email),
			IsActive: active, HasData: hasData, DjangoID: uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func copyTrucks(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.Truck{})
	byPlate := map[string]uint{}
	var have []models.Truck
	dest.Select("ID", "PlateNumber", "DjangoID").Find(&have)
	for _, r := range have {
		byPlate[r.PlateNumber] = r.ID
		rememberDjangoUint(&models.Truck{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(horse_plate_number,''), COALESCE(tank_one_plate_number,''),
		       COALESCE(tank_two_plate_number,''), COALESCE(vehicle_type,0), COALESCE(loading_type,1),
		       COALESCE(lng_cng,false), COALESCE(mplw,0), COALESCE(gcwr,0), COALESCE(tw,0),
		       COALESCE(is_active,true), COALESCE(have_data,false)
		FROM "GantryTruck"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("trucks", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.Truck) {
		for i := range batch {
			putTruck(batch[i])
		}
	}))
	for rows.Next() {
		var id int64
		var plate, trailer, trailerTwo string
		var vtype, loadType int
		var lng bool
		var mplw, gcwr, tw float64
		var active, hasData bool
		if err := rows.Scan(&id, &plate, &trailer, &trailerTwo, &vtype, &loadType, &lng, &mplw, &gcwr, &tw, &active, &hasData); err != nil {
			w.close()
			return err
		}
		plate = strings.TrimSpace(plate)
		if plate == "" || idByDjango(dest, &models.Truck{}, id) != 0 {
			continue
		}
		if existID, ok := byPlate[plate]; ok {
			stampDjangoID(dest, &models.Truck{}, existID, id)
			continue
		}
		kind := types.VehicleStraight
		switch vtype {
		case 2:
			kind = types.VehicleSemi
		case 3:
			kind = types.VehiclePulling
		default:
			if trailer != "" && trailerTwo != "" {
				kind = types.VehiclePulling
			} else if trailer != "" {
				kind = types.VehicleSemi
			}
		}
		load := types.LoadingTop
		if loadType == 2 {
			load = types.LoadingBottom
		}
		byPlate[plate] = 1
		w.add(models.Truck{
			PlateNumber: plate, Trailer: trailer, TrailerTwo: trailerTwo,
			VehicleType: kind, LoadingType: load, LngCng: lng,
			Mplw: decimal.NewFromFloat(mplw), Gcwr: decimal.NewFromFloat(gcwr),
			TareWeight: decimal.NewFromFloat(tw), IsActive: active, HasData: hasData, DjangoID: uint(id),
		})
	}
	w.close()
	return rows.Err()
}
