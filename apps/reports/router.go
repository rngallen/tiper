package reports

import (
	"strings"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/permissions"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

func Router(app *fiber.App) {
	BindLetterhead()
	g := app.Group("/api/v1/reports", middleware.PasetoMiddleware(), middleware.SessionVersionMiddleware())
	read := middleware.PermissionMiddleware(permissions.ReportsRead)

	g.Get("/registry", read, func(c fiber.Ctx) error {
		return response.OkDetail(c, registry())
	})
	g.Get("/filter-options", read, filterOptions)

	register := func(code string, h fiber.Handler) {
		g.Get("/"+code, read, h)
		g.Get("/"+code+".pdf", read, h)
	}
	g.Get("/charts/catalog", read, chartCatalog)
	g.Get("/charts/:source<string>", read, chartData)

	register("dashboard", dashboard)
	register("stock-position", stockPositionPivot)
	register("eom-stock", eomStockPivot)
	register("balances-status", balancesStatusPivot)
	register("customer-range", customerRange)
	register("customer-statement", customerStatement)
	register("cargo-tracking", cargoTracking)
	register("vessel-parcel", vesselParcelBilling)
	register("stock-movement", stockMovementSubsidiary)
	register("sbm-reception", reception("SBM"))
	register("koj-reception", reception("KOJ"))
	register("sbm-monthly", receptionMonthly("SBM"))
	register("koj-monthly", receptionMonthly("KOJ"))
	register("market-share", marketShare)
	register("market-share-ships", marketShareShips)
	register("market-share-monthly", marketShareMonthly)
	register("sbm-ships-share", func(c fiber.Ctx) error { return writeMarketShareShips(c, "SBM") })
	register("koj-ships-share", func(c fiber.Ctx) error { return writeMarketShareShips(c, "KOJ") })
	register("sbm-monthly-share", func(c fiber.Ctx) error { return writeMarketShareMonthly(c, "SBM") })
	register("koj-monthly-share", func(c fiber.Ctx) error { return writeMarketShareMonthly(c, "KOJ") })
	register("sbm-receipts", routeReceiptsShare("SBM"))
	register("koj-receipts", routeReceiptsShare("KOJ"))
	register("pump-over-loading", pumpOverLoading)
	register("pump-over-omc", pumpOverOMC)
	register("pump-over-status", pumpOverStatus)

	register("periodic-stock-status", periodicStockStatus)
	register("gain-loss", gainLoss)
	register("ewura-weekly", ewuraWeekly)
	register("pbpa-weekly", pbpaWeekly)
	register("customer-stock", individualCustomerStock)
	register("daily-mass-balance", dailyMassBalance)
	register("monthly-confirmation", monthlyStockConfirmation)
	register("itt-summary", ittSummary)
	register("srt-vessels", srtVessels)
	register("hold-register", holdRegister)
	register("reservation-exposure", reservationExposure)

	register("billing-list", billingList)
	register("billing-by-trade", billingByTrade)
	register("billing-by-product", billingByProduct)
	register("billing-exceptions", billingExceptions)

	register("customers", customersList)
	register("products", productsList)
	register("vessels", vesselsList)
	register("drivers", driversList)
	register("trucks", trucksList)
	register("transporters", transportersList)
	register("depots", depotsList)
	register("tanks", tanksList)
	register("itts", ittsList)
	register("pump-over-requests", pumpOverRequestsList)
	register("pump-over-reports", pumpOverReportsList)

	register("yearly-loading", yearlyLoading)
	register("monthly-loading", monthlyLoading)
	register("daily-loading", dailyLoading)
	register("loading-plan", loadingPlan)
	register("transit-destination", transitDestination)
	register("ewura-loaded-trucks", ewuraLoadedTrucks)
	register("marked-fuel", markedFuel)
	register("glr-status", glrStatus)
	register("glr-approvals", glrApprovals)
	register("truck-seals", truckSeals)

	register("glr-document", glrDocument)
	register("delivery-note", deliveryNote)
	register("gate-in", gateIn)
	register("gate-out", gateOut)
	register("pump-over-document", pumpOverDocument)
	register("pump-over-report", pumpOverReportDoc)
	register("itt-document", ittDocument)
	register("receipt-document", receiptDocument)
	register("zerolization-document", zerolizationDocument)
	register("hold-release-document", holdReleaseDocument)
	register("miloss-document", miLossDocument)

	register("users", usersReport)
	register("roles", rolesReport)
	register("audit-activity", auditActivity)

	register("billing-summary", billingSummary)
	register("tank-ullage", tankUllage)
	register("open-orders", openOrders)
	register("license-expiry", licenseExpiry)
	register("workflow-aging", workflowAging)
	register("stock-aging", stockAging)
	register("daily-throughput", dailyThroughput)
}

func dashboard(c fiber.Ctx) error {
	var receipts, openRuns, licenses, movements int64
	_ = db.Db.Model(&models.Receipt{}).Count(&receipts).Error
	_ = db.Db.Model(&models.BillingRun{}).Where("Status IN ?", []string{"draft", "submitted"}).Count(&openRuns).Error
	_ = db.Db.Model(&models.EwuraPetroleumLicense{}).Count(&licenses).Error
	_ = db.Db.Model(&models.StockMovement{}).Count(&movements).Error
	type bal struct {
		Free decimal.Decimal
	}
	var free bal
	_ = db.Db.Raw(`
		SELECT ISNULL(SUM(b.Quantity), 0)
			- ISNULL((SELECT SUM(Quantity) FROM StockReservation WHERE Status = 'open'), 0) AS Free
		FROM StockBalance b
		WHERE b.IsProvision = 0`).Scan(&free).Error
	payload := fiber.Map{
		"receipts":    receipts,
		"openRuns":    openRuns,
		"licenses":    licenses,
		"movements":   movements,
		"bookStock":   free.Free,
		"generatedAt": time.Now(),
	}
	if wantsExcel(c) || wantsPDF(c) {
		return serveAny(c, "Operations snapshot", "dashboard", []fiber.Map{payload})
	}
	return response.OkDetail(c, payload)
}

func customerRange(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		CustomerName string
		ProductCode  string
		InQty        decimal.Decimal
		OutQty       decimal.Decimal
		NetQty       decimal.Decimal
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode,
			SUM(d.ReceivedQty) AS InQty,
			-SUM(d.OutflowQty) AS OutQty,
			SUM(d.ReceivedQty - d.OutflowQty) AS NetQty
		FROM StockDailyPosition d
		JOIN Customer c ON c.ID = d.CustomerID
		JOIN Product p ON p.ID = d.ProductID
		WHERE d.PositionDate >= ? AND d.PositionDate <= ?
		GROUP BY c.Name, p.Code`, from, to).Scan(&rows).Error
	return serveRegister(c, "Customer range", "customer_range", rows)
}

func cargoTracking(c fiber.Ctx) error {
	type row struct {
		CustomerName string
		VesselCode   string
		VesselDate   time.Time
		ProductCode  string
		Quantity     decimal.Decimal
		StatusName   string
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, v.Code AS VesselCode, b.VesselDate, p.Code AS ProductCode,
			SUM(b.Quantity) AS Quantity, st.Name AS StatusName
		FROM StockBalance b
		JOIN Customer c ON c.ID = b.CustomerID
		JOIN Vessel v ON v.ID = b.VesselID
		JOIN Product p ON p.ID = b.ProductID
		JOIN StockStatus st ON st.ID = b.StockStatusID
		WHERE b.IsProvision = 0
		GROUP BY c.Name, v.Code, b.VesselDate, p.Code, st.Name
		HAVING SUM(b.Quantity) <> 0`).Scan(&rows).Error
	return serveRegister(c, "Cargo tracking", "cargo_tracking", rows)
}

func reception(route string) fiber.Handler {
	return func(c fiber.Ctx) error {
		from, to := parseRange(c)
		type row struct {
			VesselCode   string
			VesselDate   time.Time
			ProductCode  string
			CustomerName string
			Quantity     decimal.Decimal
		}
		var rows []row
		_ = db.Db.Raw(`
			SELECT VesselCode, VesselDate, ProductCode, CustomerName, CubicMeter AS Quantity
			FROM ReceptionFact
			WHERE RouteCode = ? AND Date >= ? AND Date <= ?`,
			route, from, to).Scan(&rows).Error
		return serveAny(c, route+" reception", strings.ToLower(route)+"_reception", rows)
	}
}

func marketShare(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		RouteCode    string
		CustomerName string
		Ships        int64
		Quantity     decimal.Decimal
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT RouteCode, CustomerName,
			COUNT(DISTINCT VesselID) AS Ships, SUM(CubicMeter) AS Quantity
		FROM ReceptionFact
		WHERE Date >= ? AND Date <= ?
		GROUP BY RouteCode, CustomerName`, from, to).Scan(&rows).Error
	return serveAny(c, "Market share", "market_share", rows)
}

func pumpOverLoading(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		Month        string
		EventType    string
		CustomerCode string
		Quantity     decimal.Decimal
		Trucks       int64
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT CONVERT(varchar(7), OccurredAt, 126) AS Month, EventType, CustomerCode,
			SUM(Quantity) AS Quantity, COUNT(*) AS Trucks
		FROM InventoryEventLog
		WHERE Posted = 1 AND OccurredAt >= ? AND OccurredAt <= ?
		GROUP BY CONVERT(varchar(7), OccurredAt, 126), EventType, CustomerCode
		ORDER BY Month DESC`, from, to).Scan(&rows).Error
	return serveAny(c, "Pump-over and gantry loading", "pump_over_loading", rows)
}

func parseRange(c fiber.Ctx) (time.Time, time.Time) {
	to := time.Now()
	from := to.AddDate(0, -1, 0)
	if d := c.Query("from"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			from = t
		}
	}
	if d := c.Query("to"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			to = t
		}
	}
	return from, to
}
