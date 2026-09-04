// One-way copy from the retired Django fuel_delivery PostgreSQL database into
// DFMS MSSQL. This is not a live bridge — run it once in a change window, then
// operators work only in DFMS.
//
//	go run ./tools/migrate-fuel-delivery -src "postgres://user:pass@host:5432/stock"
//	go run ./tools/migrate-fuel-delivery -src "..." -dry-run=false
//
// Destination is the application database from .env (DFMS.DB.*). Each imported
// parent stores its Django primary key on DjangoID. Children resolve FKs by
// looking up that DjangoID and writing the new DFMS ID — Django ids are never
// copied into foreign-key columns. Re-runs are idempotent.
//
// Large tables (ILO lines, comp lines, loadings) are copied with a bounded
// worker pool (-workers, -batch). Stages stay serial so parents exist first.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/inventory"
	"dfms/pkg/config"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"dfms/pkg/migrate"
	"dfms/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	src := flag.String("src", "", "Django PostgreSQL DSN")
	dry := flag.Bool("dry-run", true, "print source counts without writing")
	workers := flag.Int("workers", 8, "parallel MSSQL insert workers for large tables")
	batch := flag.Int("batch", 200, "rows per insert batch")
	flag.Parse()
	if strings.TrimSpace(*src) == "" {
		fmt.Fprintln(os.Stderr, "usage: migrate-fuel-delivery -src postgres://... [-dry-run=false] [-workers=8] [-batch=200]")
		os.Exit(2)
	}
	insertWorkers, insertBatch = clampCopyPool(*workers, *batch)
	ctx := context.Background()
	if err := config.InitConfig(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pg, err := pgx.Connect(ctx, *src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres: %v\n", err)
		os.Exit(1)
	}
	defer pg.Close(ctx)

	fmt.Printf("fuel_delivery → DFMS (one-way)\n  source: %s\n  dry-run: %v\n  workers: %d  batch: %d\n\n", *src, *dry, insertWorkers, insertBatch)
	counts := []struct{ table, note string }{
		{"AuthUser", "staff (passwords reset — not copied)"},
		{"SageStatus", "→ StockStatus (Local / Transit / Congo / …)"},
		{"EwuraPetroleumLicences", "→ EwuraPetroleumLicense (not seeded on migrate up)"},
		{"SageCustomer", "OMC codes + EWURA license"},
		{"SageItem", "AGO / PMS"},
		{"SageVessel", "vessel names"},
		{"GantryTransporter", "hauliers"},
		{"GantryDriver", "drivers"},
		{"GantryTruck", "plates"},
		{"GantryTank", "truck tanks"},
		{"PipelineDeliveryTank", "storage tanks → Tank"},
		{"GantryTankCalibration", "calibrations"},
		{"GantryDestination", "destinations"},
		{"GantryIlr", "→ GantryLoadingRequest"},
		{"GantryIlrLine", "→ GantryLoadingLine (ILO)"},
		{"GantryIlrAttachment", "→ Attachment"},
		{"GantryIlo", "loaded qty onto ILO"},
		{"GantryRfid", "gantry badges"},
		{"GantryCompartmentalization", "→ Compartmentalization"},
		{"GantryCompartmentalizationLine", "→ GantryCompartmentalizationLine"},
		{"GantryLoading", "→ GantryLoading + GantryLoadingProduct"},
		{"GantryVesselLoading", "→ GantryVesselLoading"},
		{"GantryCompartmentalizationAttachment", "→ Attachment"},
		{"PipelineDeliveryRequest", "→ PumpOverRequest"},
		{"PipelineDeliveryReport", "→ PumpOverReport"},
		{"TransferItt", "→ IttTransfer"},
	}
	for _, t := range counts {
		n, err := countTable(ctx, pg, t.table)
		if err != nil {
			fmt.Printf("  %-32s  (missing)  %s\n", t.table, t.note)
			continue
		}
		fmt.Printf("  %-32s  %6d  %s\n", t.table, n, t.note)
	}
	if *dry {
		fmt.Println("\ndry-run: re-run with -dry-run=false after migrate up on an empty DFMS database.")
		return
	}

	if err := db.ConnectDatabase(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "dfms db: %v\n", err)
		os.Exit(1)
	}
	if err := migrate.RequireReady(db.Db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	adminID := uint(1)
	var admin models.User
	if err := db.Db.Order("ID ASC").First(&admin).Error; err == nil {
		adminID = admin.ID
	}

	if err := copyLive(ctx, pg, db.Db, adminID); err != nil {
		fmt.Fprintf(os.Stderr, "copy: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\ncopy finished — Django can stay offline. Run DFMS only.")
}

func countTable(ctx context.Context, pg *pgx.Conn, table string) (int, error) {
	var n int
	err := pg.QueryRow(ctx, `SELECT COUNT(*) FROM "`+table+`"`).Scan(&n)
	return n, err
}

func copyLive(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	dest = dest.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err := copyTitles(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyUsers(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyProducts(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copySageStatuses(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyLicenses(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyCustomers(ctx, pg, dest, adminID); err != nil {
		return err
	}
	warmMasters(dest)
	if err := copyVessels(ctx, pg, dest, adminID); err != nil {
		return err
	}
	warmPlaces(dest)
	if err := copyLogistics(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyExtra(ctx, pg, dest, adminID); err != nil {
		return err
	}
	warmPlaces(dest)
	warmTrucks(dest)
	if err := copyGLR(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyILO(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyPDO(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyPDOVessels(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyITT(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyCompartmentalizations(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyApprovalTrails(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyLoadings(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyVesselLoadings(ctx, pg, dest); err != nil {
		return err
	}
	if err := rebuildLoadingSummaries(dest); err != nil {
		return fmt.Errorf("rebuild loading summaries: %w", err)
	}
	if err := copyAmendments(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyCustomerDocs(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := copyILRAttachments(ctx, pg, dest, adminID); err != nil {
		return err
	}
	if err := inventory.RebuildSnapshots(dest); err != nil {
		return fmt.Errorf("rebuild snapshots: %w", err)
	}
	logs.Info("fuel_delivery copy complete")
	return nil
}

func copyVessels(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(value,''), COALESCE(CAST(code AS text),''), COALESCE(is_active,true)
		FROM "SageVessel" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name, code string
		var active bool
		if err := rows.Scan(&id, &name, &code, &active); err != nil {
			return err
		}
		if idByDjango(dest, &models.Vessel{}, id) != 0 {
			continue
		}
		code = strings.ToUpper(strings.TrimSpace(firstNonEmpty(code, name)))
		name = firstNonEmpty(name, code)
		if code == "" {
			continue
		}
		var existing models.Vessel
		if dest.Where("Code = ? OR Name = ?", code, name).First(&existing).Error == nil {
			stampDjangoID(dest, &models.Vessel{}, existing.ID, id)
			continue
		}
		row := models.Vessel{Code: code, Name: name, IsActive: active, CreatedByID: adminID, DjangoID: uint(id)}
		if err := dest.Create(&row).Error; err != nil {
			return err
		}
		fmt.Printf("  + vessel %s (django %d → %d)\n", code, id, row.ID)
	}
	return rows.Err()
}

func copyLogistics(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	if err := copyTransporters(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyDrivers(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyTrucks(ctx, pg, dest); err != nil {
		return err
	}
	return copyDestinations(ctx, pg, dest)
}

func copyDestinations(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.Destination{})
	byName := map[string]uint{}
	var have []models.Destination
	dest.Select("ID", "Name", "DjangoID").Find(&have)
	for _, r := range have {
		byName[strings.TrimSpace(r.Name)] = r.ID
		rememberDjangoUint(&models.Destination{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `SELECT id, name, is_country, COALESCE(is_active,true) FROM "GantryDestination"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("destinations", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.Destination) {
		for i := range batch {
			rememberDjangoUint(&models.Destination{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id int64
		var name string
		var country, active bool
		if err := rows.Scan(&id, &name, &country, &active); err != nil {
			w.close()
			return err
		}
		name = strings.TrimSpace(name)
		if name == "" || idByDjango(dest, &models.Destination{}, id) != 0 {
			continue
		}
		if existID, ok := byName[name]; ok {
			stampDjangoID(dest, &models.Destination{}, existID, id)
			continue
		}
		byName[name] = 1
		w.add(models.Destination{Name: name, IsCountry: country, IsActive: active, DjangoID: uint(id)})
	}
	w.close()
	return rows.Err()
}

func copyGLR(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	statusFallback := firstStatus(dest)
	warmMasters(dest)
	warmDjangoIDs(dest, &models.GantryLoadingRequest{})
	byDoc := map[string]uint{}
	var have []models.GantryLoadingRequest
	dest.Select("ID", "DocumentNumber", "DjangoID").Find(&have)
	for _, r := range have {
		byDoc[r.DocumentNumber] = r.ID
		rememberDjangoUint(&models.GantryLoadingRequest{}, r.DjangoID, r.ID)
	}

	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(doc_number,''), COALESCE(batch_number,''), ilr_date,
		       COALESCE(description,''), customer_id, product_id, product_status_id, created_by_id,
		       COALESCE(quantity,0), COALESCE(status,'completed'),
		       COALESCE(valid_contract_available,false), COALESCE(loading_order_available,false),
		       by_product_id, COALESCE(by_product_quantity,0)
		FROM "GantryIlr" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := newBatchWriter("ILR headers", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.GantryLoadingRequest) {
		for i := range batch {
			rememberDjangoUint(&models.GantryLoadingRequest{}, batch[i].DjangoID, batch[i].ID)
			putILR(batch[i].ID, batch[i].ProductID, batch[i].StockStatusID)
		}
	}))
	skipped := 0
	for rows.Next() {
		var id, customerID, productID, statusID, createdBy int64
		var byProductID *int64
		var doc, batch, status, desc string
		var ilrDate time.Time
		var qty, byQty float64
		var contract, loadingOrder bool
		if err := rows.Scan(&id, &doc, &batch, &ilrDate, &desc, &customerID, &productID, &statusID, &createdBy, &qty, &status, &contract, &loadingOrder, &byProductID, &byQty); err != nil {
			out.close()
			return err
		}
		if doc == "" {
			continue
		}
		if existing := idByDjango(dest, &models.GantryLoadingRequest{}, id); existing != 0 {
			continue
		}
		if existID, ok := byDoc[doc]; ok {
			stampDjangoID(dest, &models.GantryLoadingRequest{}, existID, id)
			continue
		}
		cid := idByDjango(dest, &models.Customer{}, customerID)
		pid := idByDjango(dest, &models.Product{}, productID)
		if cid == 0 || pid == 0 {
			skipped++
			continue
		}
		sid := idByDjango(dest, &models.StockStatus{}, statusID)
		if sid == 0 {
			sid = statusFallback
		}
		if sid == 0 {
			skipped++
			continue
		}
		if len(batch) > 8 {
			batch = batch[len(batch)-8:]
		}
		desc = strings.TrimSpace(desc)
		if desc == "" {
			desc = doc
		}
		row := models.GantryLoadingRequest{
			DocumentNumber:        doc,
			BatchNumber:           firstNonEmpty(batch, doc),
			OrderDate:             ilrDate,
			Description:           desc,
			CustomerID:            cid,
			ProductID:             pid,
			StockStatusID:         sid,
			Quantity:              decimal.NewFromFloat(qty),
			CubicMeter:            decimal.NewFromFloat(qty),
			ValidContract:         contract,
			LoadingOrderAvailable: loadingOrder,
			Status:                mapStatus(status),
			CreatedByID:           userByDjango(dest, adminID, createdBy),
			DjangoID:              uint(id),
		}
		if byProductID != nil && *byProductID > 0 {
			if bid := idByDjango(dest, &models.Product{}, *byProductID); bid != 0 && bid != pid {
				row.ByProductID = &bid
				row.ByProductQuantity = decimal.NewFromFloat(byQty)
			}
		}
		byDoc[doc] = 1
		out.add(row)
	}
	out.close()
	if skipped > 0 {
		fmt.Printf("  skip %d ILR headers (customer/product not imported)\n", skipped)
	}
	warmILR(dest)
	warmPlaces(dest)
	warmTrucks(dest)
	if err := copyGLO(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyILRVessels(ctx, pg, dest); err != nil {
		return err
	}
	if err := copyILRPositions(ctx, pg, dest); err != nil {
		return err
	}
	return copyILROutstanding(ctx, pg, dest)
}

func copyILRVessels(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.GantryRequestVessel{})
	warmDjangoIDs(dest, &models.GantryLoadingRequest{})
	warmDjangoIDs(dest, &models.Vessel{})
	seen := map[string]struct{}{}
	var have []models.GantryRequestVessel
	dest.Select("RequestID", "VesselID", "VesselDate", "ProductID", "StockStatusID").Find(&have)
	for _, r := range have {
		seen[fmt.Sprintf("%d:%d:%s:%d:%d", r.RequestID, r.VesselID, r.VesselDate.Format("2006-01-02"), r.ProductID, r.StockStatusID)] = struct{}{}
	}

	rows, err := pg.Query(ctx, `
		SELECT id, request_id, vessel_id, vessel_date, COALESCE(quantity,0),
		       product_id, product_status_id, COALESCE(quantity_loaded,0), COALESCE(financial_hold,false)
		FROM "GantryIlrVessel"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("ILR vessels", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.GantryRequestVessel) {
		for i := range batch {
			rememberDjangoUint(&models.GantryRequestVessel{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, reqID, vesselID, productID, statusID int64
		var dt time.Time
		var qty, loaded float64
		var hold bool
		if err := rows.Scan(&id, &reqID, &vesselID, &dt, &qty, &productID, &statusID, &loaded, &hold); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.GantryRequestVessel{}, id) != 0 {
			continue
		}
		glrID := idByDjango(dest, &models.GantryLoadingRequest{}, reqID)
		vid := idByDjango(dest, &models.Vessel{}, vesselID)
		if glrID == 0 || vid == 0 {
			continue
		}
		ilr, ok := ilrFields(glrID)
		if !ok {
			continue
		}
		pid := idByDjango(dest, &models.Product{}, productID)
		if pid == 0 {
			pid = ilr.ProductID
		}
		sid := idByDjango(dest, &models.StockStatus{}, statusID)
		if sid == 0 {
			sid = ilr.StockStatusID
		}
		if sid == 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d:%s:%d:%d", glrID, vid, dt.Format("2006-01-02"), pid, sid)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		w.add(models.GantryRequestVessel{
			RequestID: glrID, VesselID: vid, VesselDate: dt,
			ProductID: pid, StockStatusID: sid,
			Quantity: decimal.NewFromFloat(qty), CubicMeter: decimal.NewFromFloat(qty),
			LoadedQty: decimal.NewFromFloat(loaded), FinancialHold: hold, DjangoID: uint(id),
		})
	}
	w.close()
	return rows.Err()
}

func copyGLO(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.GantryLoadingLine{})
	warmDjangoIDs(dest, &models.GantryLoadingRequest{})
	warmDjangoIDs(dest, &models.Transporter{})
	warmDjangoIDs(dest, &models.Driver{})
	warmDjangoIDs(dest, &models.Product{})
	byDoc := map[string]uint{}
	var have []struct {
		ID             uint
		DocumentNumber string
	}
	dest.Model(&models.GantryLoadingLine{}).Select("ID", "DocumentNumber").Scan(&have)
	for _, r := range have {
		byDoc[r.DocumentNumber] = r.ID
	}

	rows, err := pg.Query(ctx, `
		SELECT id, request_id, COALESCE(doc_number,''), COALESCE(customer_order_number,''),
		       COALESCE(transporter_name,''), COALESCE(driver_name,''), COALESCE(plate_number,''),
		       COALESCE(destination_name,''), COALESCE(district_name,''), COALESCE(quantity,0),
		       COALESCE(loading_status,1), expiration_date, COALESCE(ewura_licence,''),
		       transporter_id, driver_id, truck_id, product_id, by_product_id, COALESCE(by_product_quantity,0)
		FROM "GantryIlrLine" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("ILO lines", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.GantryLoadingLine) {
		for i := range batch {
			rememberDjangoUint(&models.GantryLoadingLine{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	skipped := 0
	for rows.Next() {
		var id, reqID int64
		var transporterID, driverID, truckID, productID, byProductID *int64
		var doc, con, transporter, driver, plate, destName, district, lic string
		var qty, byQty float64
		var loadStatus int
		var exp *time.Time
		if err := rows.Scan(&id, &reqID, &doc, &con, &transporter, &driver, &plate, &destName, &district, &qty, &loadStatus, &exp, &lic,
			&transporterID, &driverID, &truckID, &productID, &byProductID, &byQty); err != nil {
			w.close()
			return err
		}
		if doc == "" {
			continue
		}
		if idByDjango(dest, &models.GantryLoadingLine{}, id) != 0 {
			continue
		}
		glrID := idByDjango(dest, &models.GantryLoadingRequest{}, reqID)
		if glrID == 0 {
			skipped++
			continue
		}
		if existID, ok := byDoc[doc]; ok {
			stampDjangoID(dest, &models.GantryLoadingLine{}, existID, id)
			continue
		}
		pid, ok := ilrProduct(glrID)
		if !ok {
			skipped++
			continue
		}
		if productID != nil {
			if mapped := idByDjango(dest, &models.Product{}, *productID); mapped != 0 {
				pid = mapped
			}
		}
		row := models.GantryLoadingLine{
			RequestID:           glrID,
			DocumentNumber:      doc,
			CustomerOrderNumber: con,
			ProductID:           pid,
			TransporterName:     transporter,
			DriverName:          driver,
			TruckPlate:          plate,
			HorsePlate:          plate,
			TrailerOnePlate:     plate,
			Destination:         destName,
			District:            district,
			EwuraLicense:        lic,
			RequestedQty:        decimal.NewFromFloat(qty),
			CubicMeter:          decimal.NewFromFloat(qty),
			ExpirationDate:      exp,
			IsActive:            loadStatus != 5,
			Status:              mapLoadingStatus(loadStatus),
			DjangoID:            uint(id),
		}
		if tid := idByDjangoPtr(dest, &models.Transporter{}, transporterID); tid != 0 {
			row.TransporterID = &tid
		}
		if did := idByDjangoPtr(dest, &models.Driver{}, driverID); did != 0 {
			row.DriverID = &did
		}
		if trid := idByDjangoPtr(dest, &models.Truck{}, truckID); trid != 0 {
			row.TruckID = &trid
			if truck, ok := truckPlates(trid); ok {
				row.TruckPlate = firstNonEmpty(row.TruckPlate, truck.PlateNumber)
				row.HorsePlate = truck.PlateNumber
				row.TrailerOnePlate = truck.Trailer
				row.TrailerTwoPlate = truck.TrailerTwo
			}
		}
		if destID := destIDByName(dest, destName); destID != 0 {
			row.DestinationID = &destID
			if distID := districtIDByName(dest, destID, district); distID != 0 {
				row.DistrictID = &distID
			}
		}
		if byProductID != nil {
			if bid := idByDjango(dest, &models.Product{}, *byProductID); bid != 0 && bid != pid {
				row.ByProductID = &bid
				row.ByProductQuantity = decimal.NewFromFloat(byQty)
			}
		}
		byDoc[doc] = 1
		w.add(row)
	}
	w.close()
	if skipped > 0 {
		fmt.Printf("  skip %d ILO lines (ILR not imported)\n", skipped)
	}
	return rows.Err()
}

func copyILRPositions(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.GantryStockPosition{})
	warmDjangoIDs(dest, &models.GantryLoadingRequest{})
	seen := map[string]struct{}{}
	var have []models.GantryStockPosition
	dest.Select("RequestID", "ProductID").Find(&have)
	for _, r := range have {
		seen[fmt.Sprintf("%d:%d", r.RequestID, r.ProductID)] = struct{}{}
	}

	rows, err := pg.Query(ctx, `
		SELECT id, request_id, product_id, COALESCE(total_balance,0), COALESCE(volume_fhold,0),
		       COALESCE(final_volume,0), COALESCE(price,0)
		FROM "GantryIlrStockPosition"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("ILR stock positions", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.GantryStockPosition) {
		for i := range batch {
			rememberDjangoUint(&models.GantryStockPosition{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, reqID, productID int64
		var total, hold, final, price float64
		if err := rows.Scan(&id, &reqID, &productID, &total, &hold, &final, &price); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.GantryStockPosition{}, id) != 0 {
			continue
		}
		glrID := idByDjango(dest, &models.GantryLoadingRequest{}, reqID)
		pid := idByDjango(dest, &models.Product{}, productID)
		if glrID == 0 || pid == 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", glrID, pid)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		w.add(models.GantryStockPosition{
			RequestID: glrID, ProductID: pid, DjangoID: uint(id),
			TotalBalance: decimal.NewFromFloat(total), HoldQty: decimal.NewFromFloat(hold),
			FreeQty: decimal.NewFromFloat(total - hold), FinalQty: decimal.NewFromFloat(final),
			Price: decimal.NewFromFloat(price),
		})
	}
	w.close()
	return rows.Err()
}

func copyILROutstanding(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	warmDjangoIDs(dest, &models.GantryCustomerOutstanding{})
	warmDjangoIDs(dest, &models.GantryLoadingRequest{})
	seen := map[uint]struct{}{}
	var have []models.GantryCustomerOutstanding
	dest.Select("RequestID").Find(&have)
	for _, r := range have {
		seen[r.RequestID] = struct{}{}
	}

	rows, err := pg.Query(ctx, `
		SELECT id, request_id, COALESCE(storage_tzs,0), COALESCE(storage_usd,0),
		       COALESCE(weight_and_measure_tzs,0), COALESCE(tbs_tzs,0)
		FROM "GantryIlrCustomerOutStanding"`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("ILR outstanding", insertWorkers, insertBatch, gormInsert(dest, func(batch []models.GantryCustomerOutstanding) {
		for i := range batch {
			rememberDjangoUint(&models.GantryCustomerOutstanding{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	for rows.Next() {
		var id, reqID int64
		var storageTZS, storageUSD, wm, tbs float64
		if err := rows.Scan(&id, &reqID, &storageTZS, &storageUSD, &wm, &tbs); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.GantryCustomerOutstanding{}, id) != 0 {
			continue
		}
		glrID := idByDjango(dest, &models.GantryLoadingRequest{}, reqID)
		if glrID == 0 {
			continue
		}
		if _, ok := seen[glrID]; ok {
			continue
		}
		seen[glrID] = struct{}{}
		w.add(models.GantryCustomerOutstanding{
			RequestID: glrID, DjangoID: uint(id),
			StorageTZS: decimal.NewFromFloat(storageTZS), StorageUSD: decimal.NewFromFloat(storageUSD),
			WeightMeasureTZS: decimal.NewFromFloat(wm), TbsTZS: decimal.NewFromFloat(tbs),
		})
	}
	w.close()
	return rows.Err()
}

func copyPDO(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	statusFallback := firstStatus(dest)
	if statusFallback == 0 {
		return fmt.Errorf("stock status master is empty; run migrate up before copy")
	}
	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(doc_number,''), request_date, customer_id, product_id, depot_id,
		       COALESCE(quantity,0), COALESCE(status,'completed'), created_by_id,
		       COALESCE(customer_order_number,'')
		FROM "PipelineDeliveryRequest" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id, customerID, productID, depotID, createdBy int64
		var doc, status, orderNo string
		var orderDate time.Time
		var qty float64
		if err := rows.Scan(&id, &doc, &orderDate, &customerID, &productID, &depotID, &qty, &status, &createdBy, &orderNo); err != nil {
			return err
		}
		if doc == "" || idByDjango(dest, &models.PumpOverRequest{}, id) != 0 {
			continue
		}
		var exists models.PumpOverRequest
		if dest.Where("DocumentNumber = ?", doc).First(&exists).Error == nil {
			stampDjangoID(dest, &models.PumpOverRequest{}, exists.ID, id)
			continue
		}
		cid := idByDjango(dest, &models.Customer{}, customerID)
		pid := idByDjango(dest, &models.Product{}, productID)
		did := idByDjango(dest, &models.Depot{}, depotID)
		if did == 0 {
			did = firstDepot(dest)
		}
		if cid == 0 || pid == 0 || did == 0 {
			fmt.Printf("  skip PDO %s (unmapped customer/product/depot django ids)\n", doc)
			continue
		}
		sid := vesselStatusID(ctx, pg, dest, "PipelineVessel", "request_id", id)
		if sid == 0 {
			sid = statusFallback
		}
		if sid == 0 {
			fmt.Printf("  skip PDO %s (no stock status)\n", doc)
			continue
		}
		row := models.PumpOverRequest{
			DocumentNumber:      doc,
			OrderDate:           orderDate,
			CustomerID:          cid,
			ProductID:           pid,
			StockStatusID:       sid,
			DepotID:             did,
			Quantity:            decimal.NewFromFloat(qty),
			CustomerOrderNumber: strings.TrimSpace(orderNo),
			Status:              mapStatus(status),
			CreatedByID:         userByDjango(dest, adminID, createdBy),
			DjangoID:            uint(id),
		}
		if err := dest.Omit("Customer", "Product", "StockStatus", "Depot", "Vessels").Create(&row).Error; err != nil {
			fmt.Printf("  skip PDO %s: %v\n", doc, err)
			continue
		}
		rememberDjangoUint(&models.PumpOverRequest{}, uint(id), row.ID)
		fmt.Printf("  + PDO %s (django %d → %d)\n", doc, id, row.ID)
	}
	return rows.Err()
}

func copyPDOVessels(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	statusFallback := firstStatus(dest)
	warmDjangoIDs(dest, &models.PumpOverRequest{})
	warmDjangoIDs(dest, &models.Vessel{})
	warmDjangoIDs(dest, &models.PumpOverVessel{})
	seen := map[string]struct{}{}
	var have []models.PumpOverVessel
	dest.Select("RequestID", "VesselID", "VesselDate", "StockStatusID", "DjangoID").Find(&have)
	for _, r := range have {
		seen[fmt.Sprintf("%d:%d:%s:%d", r.RequestID, r.VesselID, r.VesselDate.Format("2006-01-02"), r.StockStatusID)] = struct{}{}
		rememberDjangoUint(&models.PumpOverVessel{}, r.DjangoID, r.ID)
	}
	rows, err := pg.Query(ctx, `
		SELECT id, request_id, vessel_id, vessel_date, COALESCE(quantity,0), status_id
		FROM "PipelineVessel" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	w := newBatchWriter("PDO vessels", insertWorkers, insertBatch, gormInsert(dest.Omit("Vessel", "StockStatus"), func(batch []models.PumpOverVessel) {
		for i := range batch {
			rememberDjangoUint(&models.PumpOverVessel{}, batch[i].DjangoID, batch[i].ID)
		}
	}))
	skipped := 0
	for rows.Next() {
		var id, reqDjango, vesselDjango, statusDjango int64
		var vdate time.Time
		var qty float64
		if err := rows.Scan(&id, &reqDjango, &vesselDjango, &vdate, &qty, &statusDjango); err != nil {
			w.close()
			return err
		}
		if idByDjango(dest, &models.PumpOverVessel{}, id) != 0 {
			continue
		}
		reqID := idByDjango(dest, &models.PumpOverRequest{}, reqDjango)
		vid := idByDjango(dest, &models.Vessel{}, vesselDjango)
		sid := idByDjango(dest, &models.StockStatus{}, statusDjango)
		if sid == 0 {
			sid = statusFallback
		}
		if reqID == 0 || vid == 0 || sid == 0 {
			skipped++
			continue
		}
		key := fmt.Sprintf("%d:%d:%s:%d", reqID, vid, vdate.Format("2006-01-02"), sid)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		w.add(models.PumpOverVessel{
			RequestID: reqID, VesselID: vid, VesselDate: vdate, StockStatusID: sid,
			Quantity: decimal.NewFromFloat(qty), DjangoID: uint(id),
		})
	}
	w.close()
	if skipped > 0 {
		fmt.Printf("  skip %d PDO vessels (unmapped request/vessel/status)\n", skipped)
	}
	return rows.Err()
}

func copyITT(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, adminID uint) error {
	statusFallback := firstStatus(dest)
	if statusFallback == 0 {
		return fmt.Errorf("stock status master is empty; run migrate up before copy")
	}
	rows, err := pg.Query(ctx, `
		SELECT id, COALESCE(doc_number,''), itt_date, transferor_id, transferee_id, product_id,
		       COALESCE(quantity,0), created_by_id
		FROM "TransferItt" ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id, fromID, toID, productID, createdBy int64
		var doc string
		var dt time.Time
		var qty float64
		if err := rows.Scan(&id, &doc, &dt, &fromID, &toID, &productID, &qty, &createdBy); err != nil {
			return err
		}
		if doc == "" || idByDjango(dest, &models.IttTransfer{}, id) != 0 {
			continue
		}
		var exists models.IttTransfer
		if dest.Where("DocumentNumber = ?", doc).First(&exists).Error == nil {
			stampDjangoID(dest, &models.IttTransfer{}, exists.ID, id)
			continue
		}
		fid := idByDjango(dest, &models.Customer{}, fromID)
		tid := idByDjango(dest, &models.Customer{}, toID)
		pid := idByDjango(dest, &models.Product{}, productID)
		vid, vdate, sid := ittVessel(ctx, pg, dest, id, dt)
		if sid == 0 {
			sid = statusFallback
		}
		if fid == 0 || tid == 0 || pid == 0 || vid == 0 || sid == 0 {
			fmt.Printf("  skip ITT %s (unmapped django FKs)\n", doc)
			continue
		}
		row := models.IttTransfer{
			DocumentNumber: doc, TransferDate: dt, FromCustomerID: fid, ToCustomerID: tid,
			ProductID: pid, VesselID: vid, VesselDate: vdate, StockStatusID: sid,
			Quantity: decimal.NewFromFloat(qty), Status: "posted",
			CreatedByID: userByDjango(dest, adminID, createdBy), DjangoID: uint(id),
		}
		if err := dest.Omit("FromCustomer", "ToCustomer", "Product", "Vessel").Create(&row).Error; err != nil {
			fmt.Printf("  skip ITT %s: %v\n", doc, err)
			continue
		}
		fmt.Printf("  + ITT %s (django %d → %d)\n", doc, id, row.ID)
	}
	return rows.Err()
}

func ittVessel(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, ittDjangoID int64, fallback time.Time) (uint, time.Time, uint) {
	var vesselID, statusID int64
	var dt time.Time
	err := pg.QueryRow(ctx, `
		SELECT vessel_id, vessel_date, status_id FROM "TransferVessel" WHERE itt_id = $1 ORDER BY id LIMIT 1`, ittDjangoID).
		Scan(&vesselID, &dt, &statusID)
	if err != nil {
		return 0, fallback, 0
	}
	vid := idByDjango(dest, &models.Vessel{}, vesselID)
	if dt.IsZero() {
		dt = fallback
	}
	return vid, dt, idByDjango(dest, &models.StockStatus{}, statusID)
}

func vesselStatusID(ctx context.Context, pg *pgx.Conn, dest *gorm.DB, table, fkCol string, parentDjangoID int64) uint {
	var statusID int64
	q := `SELECT status_id FROM "` + table + `" WHERE ` + fkCol + ` = $1 ORDER BY id LIMIT 1`
	if err := pg.QueryRow(ctx, q, parentDjangoID).Scan(&statusID); err != nil || statusID == 0 {
		return 0
	}
	return idByDjango(dest, &models.StockStatus{}, statusID)
}

func firstStatus(dest *gorm.DB) uint {
	if dest == nil {
		return 0
	}
	var id uint
	if dest.Model(&models.StockStatus{}).Select("ID").Where("Code = ?", types.StockLocal).Limit(1).Scan(&id).Error == nil && id != 0 {
		return id
	}
	id = 0
	if dest.Model(&models.StockStatus{}).Select("ID").Order("ID ASC").Limit(1).Scan(&id).Error == nil {
		return id
	}
	return 0
}

func firstDepot(dest *gorm.DB) uint {
	var d models.Depot
	if dest.Order("ID ASC").First(&d).Error == nil {
		return d.ID
	}
	return 0
}

func mapStatus(s string) types.OrderStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "completed", "approved":
		return types.OrderApproved
	case "rejected":
		return types.OrderRejected
	case "running":
		return types.OrderSubmitted
	default:
		return types.OrderDraft
	}
}

func mapLoadingStatus(n int) types.OrderStatus {
	switch n {
	case 2:
		return types.OrderRunning
	case 3:
		return types.OrderInProgress
	case 4:
		return types.OrderLoaded
	case 5:
		return types.OrderClosed
	default:
		return types.OrderOpen
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
