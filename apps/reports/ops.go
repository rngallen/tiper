package reports

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/db"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

// stockPositionPivot is the agent Stock Position sheet: customer × product,
// plus tank dips, line content, gain/loss, and ullage.
func stockPositionPivot(c fiber.Ctx) error {
	type qty struct {
		CustomerName string
		ProductCode  string
		Quantity     decimal.Decimal
	}
	var books []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, SUM(b.Quantity) AS Quantity
		FROM StockBalance b
		JOIN Customer c ON c.ID = b.CustomerID
		JOIN Product p ON p.ID = b.ProductID
		WHERE b.IsProvision = 0
		GROUP BY c.Name, p.Code`).Scan(&books).Error

	products := productCodes()
	customers := map[string]map[string]decimal.Decimal{}
	for _, r := range books {
		if customers[r.CustomerName] == nil {
			customers[r.CustomerName] = map[string]decimal.Decimal{}
		}
		customers[r.CustomerName][r.ProductCode] = r.Quantity
	}
	names := sortedKeys(customers)
	rows := make([]fiber.Map, 0, len(names)+8)
	totals := map[string]decimal.Decimal{}
	for _, name := range names {
		row := fiber.Map{"row": name, "kind": "customer"}
		sum := decimal.Zero
		for _, p := range products {
			v := customers[name][p]
			row[p] = v
			sum = sum.Add(v)
			totals[p] = totals[p].Add(v)
		}
		row["TOTAL"] = sum
		rows = append(rows, row)
	}
	a := summaryRow("A Customers total", "summary", products, totals)
	rows = append(rows, a)

	tankBy := map[string]decimal.Decimal{}
	type tankRow struct {
		ProductCode string
		Quantity    decimal.Decimal
	}
	var tanks []tankRow
	_ = db.Db.Raw(`
		SELECT p.Code AS ProductCode, SUM(d.At20) AS Quantity
		FROM PhysicalDip d
		JOIN Tank t ON t.ID = d.TankID
		JOIN Product p ON p.ID = t.ProductID
		WHERE d.DipDate = (SELECT MAX(DipDate) FROM PhysicalDip)
		GROUP BY p.Code`).Scan(&tanks).Error
	for _, t := range tanks {
		tankBy[t.ProductCode] = t.Quantity
	}
	rows = append(rows, summaryRow("B Stocks in TIPER tanks", "summary", products, tankBy))

	type lineRow struct {
		ProductCode string
		Internal    decimal.Decimal
		External    decimal.Decimal
	}
	var lines []lineRow
	_ = db.Db.Raw(`
		SELECT p.Code AS ProductCode, l.InternalVolume AS Internal, l.ExternalVolume AS External
		FROM LineContent l
		JOIN Product p ON p.ID = l.ProductID
		WHERE l.ContentDate = (SELECT MAX(ContentDate) FROM LineContent)`).Scan(&lines).Error
	internal, external, lineTot := map[string]decimal.Decimal{}, map[string]decimal.Decimal{}, map[string]decimal.Decimal{}
	for _, l := range lines {
		internal[l.ProductCode] = l.Internal
		external[l.ProductCode] = l.External
		lineTot[l.ProductCode] = l.Internal.Add(l.External)
	}
	rows = append(rows, summaryRow("C Line content (D+E)", "summary", products, lineTot))
	rows = append(rows, summaryRow("D Internal lines", "input", products, internal))
	rows = append(rows, summaryRow("E External lines", "input", products, external))

	physical := map[string]decimal.Decimal{}
	gain := map[string]decimal.Decimal{}
	ullage := map[string]decimal.Decimal{}
	type capRow struct {
		ProductCode string
		Capacity    decimal.Decimal
	}
	var caps []capRow
	_ = db.Db.Raw(`
		SELECT p.Code AS ProductCode, SUM(t.MaximumCapacity - t.DeadStock) AS Capacity
		FROM Tank t JOIN Product p ON p.ID = t.ProductID
		WHERE t.IsActive = 1
		GROUP BY p.Code`).Scan(&caps).Error
	capBy := map[string]decimal.Decimal{}
	for _, x := range caps {
		capBy[x.ProductCode] = x.Capacity
	}
	for _, p := range products {
		physical[p] = tankBy[p].Add(lineTot[p])
		gain[p] = physical[p].Sub(totals[p])
		ullage[p] = capBy[p].Sub(tankBy[p])
	}
	rows = append(rows, summaryRow("F Total physical stock (B+C)", "summary", products, physical))
	rows = append(rows, summaryRow("G Stock gain/(loss) (F-A)", "variance", products, gain))
	rows = append(rows, summaryRow("Available ullage", "ullage", products, ullage))
	return serveAny(c, "Stock position", "stock_position", rows)
}

func eomStockPivot(c fiber.Ctx) error {
	asOf := queryDate(c)
	type qty struct {
		CustomerName string
		ProductCode  string
		Quantity     decimal.Decimal
	}
	var raw []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, SUM(d.ClosingQty) AS Quantity
		FROM (`+latestClosingSQL+`) d
		JOIN Customer c ON c.ID = d.CustomerID
		JOIN Product p ON p.ID = d.ProductID
		GROUP BY c.Name, p.Code`, asOf).Scan(&raw).Error
	pairs := make([]custProdQty, len(raw))
	for i, r := range raw {
		pairs[i] = custProdQty{r.CustomerName, r.ProductCode, r.Quantity}
	}
	return serveAny(c, fmt.Sprintf("Stock balance end of month (%s)", asOf.Format("02.01.2006")),
		"eom_stock", pivotCustomerProduct(pairs))
}

func balancesStatusPivot(c fiber.Ctx) error {
	type qty struct {
		CustomerName string
		ProductCode  string
		StatusName   string
		Quantity     decimal.Decimal
	}
	var raw []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, st.Name AS StatusName, SUM(b.Quantity) AS Quantity
		FROM StockBalance b
		JOIN Customer c ON c.ID = b.CustomerID
		JOIN Product p ON p.ID = b.ProductID
		JOIN StockStatus st ON st.ID = b.StockStatusID
		WHERE b.IsProvision = 0
		GROUP BY c.Name, p.Code, st.Name`).Scan(&raw).Error
	rows := make([]fiber.Map, 0, len(raw))
	for _, r := range raw {
		rows = append(rows, fiber.Map{
			"customer": r.CustomerName,
			"product":  r.ProductCode,
			"status":   r.StatusName,
			"quantity": r.Quantity,
		})
	}
	return serveAny(c, "Customer stock balances with status", "balances_status", rows)
}

func customerStatement(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		CustomerName string
		ProductCode  string
		Opening      decimal.Decimal
		Reception    decimal.Decimal
		Loading      decimal.Decimal
		PumpOver     decimal.Decimal
		ITT          decimal.Decimal
		Adjustment   decimal.Decimal
		Closing      decimal.Decimal
	}
	openAsOf := from.AddDate(0, 0, -1)
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode,
			ISNULL(op.ClosingQty, 0) AS Opening,
			ISNULL(per.ReceivedQty, 0) AS Reception,
			ISNULL(per.LoadingQty, 0) AS Loading,
			ISNULL(per.PumpOverQty, 0) AS PumpOver,
			ISNULL(per.ITTQty, 0) AS ITT,
			ISNULL(per.AdjustmentQty, 0) AS Adjustment,
			ISNULL(cl.ClosingQty, 0) AS Closing
		FROM Customer c
		JOIN Product p ON 1 = 1
		LEFT JOIN (`+latestClosingSQL+`) op ON op.CustomerID = c.ID AND op.ProductID = p.ID
		LEFT JOIN (`+periodFlowSQL+`) per ON per.CustomerID = c.ID AND per.ProductID = p.ID
		LEFT JOIN (`+latestClosingSQL+`) cl ON cl.CustomerID = c.ID AND cl.ProductID = p.ID
		WHERE p.IsActive = 1
			AND (ISNULL(op.ClosingQty, 0) <> 0 OR ISNULL(per.ReceivedQty, 0) <> 0
				OR ISNULL(per.OutflowQty, 0) <> 0 OR ISNULL(cl.ClosingQty, 0) <> 0)
		ORDER BY c.Name, p.Code`,
		openAsOf, from, to, to,
	).Scan(&rows).Error
	return serveAny(c, "Customer stock report", "customer_statement", rows)
}

func stockMovementSubsidiary(c fiber.Ctx) error {
	from, to := monthBounds(queryYear(c), queryMonth(c))
	type qty struct {
		CustomerName string
		ProductCode  string
		Opening      decimal.Decimal
		Receipt      decimal.Decimal
		Loading      decimal.Decimal
		PumpOver     decimal.Decimal
		ITT          decimal.Decimal
		Adjustment   decimal.Decimal
		Closing      decimal.Decimal
		Physical     decimal.Decimal
		GainLoss     decimal.Decimal
	}
	openAsOf := from.AddDate(0, 0, -1)
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode,
			ISNULL(op.ClosingQty, 0) AS Opening,
			ISNULL(per.ReceivedQty, 0) AS Receipt,
			ISNULL(per.LoadingQty, 0) AS Loading,
			ISNULL(per.PumpOverQty, 0) AS PumpOver,
			ISNULL(per.ITTQty, 0) AS ITT,
			ISNULL(per.AdjustmentQty, 0) AS Adjustment,
			ISNULL(cl.ClosingQty, 0) AS Closing
		FROM Customer c
		JOIN Product p ON 1 = 1
		LEFT JOIN (`+latestClosingSQL+`) op ON op.CustomerID = c.ID AND op.ProductID = p.ID
		LEFT JOIN (`+periodFlowSQL+`) per ON per.CustomerID = c.ID AND per.ProductID = p.ID
		LEFT JOIN (`+latestClosingSQL+`) cl ON cl.CustomerID = c.ID AND cl.ProductID = p.ID
		WHERE p.IsActive = 1
			AND (ISNULL(op.ClosingQty, 0) <> 0 OR ISNULL(per.ReceivedQty, 0) <> 0
				OR ISNULL(per.OutflowQty, 0) <> 0 OR ISNULL(cl.ClosingQty, 0) <> 0)
		ORDER BY p.Code, c.Name`,
		openAsOf, from, to, to,
	).Scan(&rows).Error
	for i := range rows {
		rows[i].Physical = rows[i].Closing
		rows[i].GainLoss = decimal.Zero
	}
	return serveAny(c, "Stock movement subsidiary", "stock_movement", rows)
}

func vesselParcelBilling(c fiber.Ctx) error {
	type qty struct {
		VesselCode   string
		VesselDate   time.Time
		CustomerName string
		ProductCode  string
		StatusName   string
		Quantity     decimal.Decimal
	}
	var raw []qty
	_ = db.Db.Raw(`
		SELECT v.Code AS VesselCode, b.VesselDate, c.Name AS CustomerName, p.Code AS ProductCode,
			st.Name AS StatusName, SUM(b.Quantity) AS Quantity
		FROM StockBalance b
		JOIN Vessel v ON v.ID = b.VesselID
		JOIN Customer c ON c.ID = b.CustomerID
		JOIN Product p ON p.ID = b.ProductID
		JOIN StockStatus st ON st.ID = b.StockStatusID
		WHERE b.IsProvision = 0
		GROUP BY v.Code, b.VesselDate, c.Name, p.Code, st.Name
		HAVING SUM(b.Quantity) <> 0
		ORDER BY b.VesselDate DESC, v.Code, c.Name`).Scan(&raw).Error
	rows := make([]fiber.Map, 0, len(raw))
	for _, r := range raw {
		rows = append(rows, fiber.Map{
			"vessel":   r.VesselCode,
			"date":     r.VesselDate.Format("02/01/2006"),
			"customer": r.CustomerName,
			"product":  r.ProductCode,
			"status":   r.StatusName,
			"quantity": r.Quantity,
		})
	}
	return serveAny(c, "Customer stock movement per vessel / parcel", "vessel_parcel", rows)
}

func receptionMonthly(route string) fiber.Handler {
	return func(c fiber.Ctx) error {
		from, to := monthBounds(queryYear(c), queryMonth(c))
		type qty struct {
			DepotName  string
			VesselName string
			Quantity   decimal.Decimal
		}
		var raw []qty
		_ = db.Db.Raw(`
			SELECT DepotName, VesselName, SUM(CubicMeter) AS Quantity
			FROM ReceptionFact
			WHERE RouteCode = ? AND Date >= ? AND Date <= ?
			GROUP BY DepotName, VesselName
			ORDER BY DepotName, VesselName`,
			route, from, to).Scan(&raw).Error
		depotTot := map[string]decimal.Decimal{}
		grand := decimal.Zero
		for _, r := range raw {
			depotTot[r.DepotName] = depotTot[r.DepotName].Add(r.Quantity)
			grand = grand.Add(r.Quantity)
		}
		rows := make([]fiber.Map, 0, len(raw)+1)
		for _, r := range raw {
			pct := decimal.Zero
			if !grand.IsZero() {
				pct = r.Quantity.Mul(decimal.NewFromInt(100)).Div(grand).Round(1)
			}
			rows = append(rows, fiber.Map{
				"depot":      r.DepotName,
				"vessel":     r.VesselName,
				"volume":     r.Quantity,
				"depotTotal": depotTot[r.DepotName],
				"sharePct":   pct,
			})
		}
		title := fmt.Sprintf("%s vessel reception %s %d", route, time.Month(queryMonth(c)).String(), queryYear(c))
		return serveAny(c, title, strings.ToLower(route)+"_monthly", rows)
	}
}

func marketShareShips(c fiber.Ctx) error {
	return writeMarketShareShips(c, strings.ToUpper(strings.TrimSpace(c.Query("route"))))
}

func writeMarketShareShips(c fiber.Ctx, route string) error {
	from, to := parseRange(c)
	type qty struct {
		VesselDate time.Time
		VesselName string
		Tiper      decimal.Decimal
		Others     decimal.Decimal
	}
	q := `
		SELECT Date AS VesselDate, VesselName,
			SUM(CASE WHEN ReceiptType = 'internal' THEN CubicMeter ELSE 0 END) AS Tiper,
			SUM(CASE WHEN ReceiptType <> 'internal' THEN CubicMeter ELSE 0 END) AS Others
		FROM ReceptionFact
		WHERE Date >= ? AND Date <= ?`
	args := []any{from, to}
	if route != "" {
		q += ` AND RouteCode = ?`
		args = append(args, route)
	}
	q += ` GROUP BY Date, VesselName ORDER BY Date, VesselName`
	var raw []qty
	_ = db.Db.Raw(q, args...).Scan(&raw).Error
	rows := make([]fiber.Map, 0, len(raw))
	for _, r := range raw {
		total := r.Tiper.Add(r.Others)
		tiperPct, othersPct := decimal.Zero, decimal.Zero
		if !total.IsZero() {
			tiperPct = r.Tiper.Mul(decimal.NewFromInt(100)).Div(total).Round(0)
			othersPct = r.Others.Mul(decimal.NewFromInt(100)).Div(total).Round(0)
		}
		rows = append(rows, fiber.Map{
			"date":  r.VesselDate.Format("02-Jan-06"),
			"ship":  r.VesselName,
			"total": total, "tiper": r.Tiper, "tiperPct": tiperPct,
			"others": r.Others, "othersPct": othersPct,
		})
	}
	title := "Received ships market share"
	if route != "" {
		title = route + " received ships market share"
	}
	return serveAny(c, title, "market_share_ships", rows)
}

func marketShareMonthly(c fiber.Ctx) error {
	return writeMarketShareMonthly(c, strings.ToUpper(strings.TrimSpace(c.Query("route"))))
}

func writeMarketShareMonthly(c fiber.Ctx, route string) error {
	year := queryYear(c)
	type qty struct {
		Month  int
		Tiper  decimal.Decimal
		Others decimal.Decimal
		Hold   decimal.Decimal
	}
	q := `
		SELECT MONTH(Date) AS Month,
			SUM(CASE WHEN ReceiptType = 'internal' THEN CubicMeter ELSE 0 END) AS Tiper,
			SUM(CASE WHEN ReceiptType <> 'internal' THEN CubicMeter ELSE 0 END) AS Others,
			SUM(CASE WHEN ReceiptType = 'internal' AND FinancialHold = 1 THEN CubicMeter ELSE 0 END) AS Hold
		FROM ReceptionFact
		WHERE YEAR(Date) = ?`
	args := []any{year}
	if route != "" {
		q += ` AND RouteCode = ?`
		args = append(args, route)
	}
	q += ` GROUP BY MONTH(Date) ORDER BY Month`
	var raw []qty
	_ = db.Db.Raw(q, args...).Scan(&raw).Error
	byMonth := map[int]qty{}
	for _, r := range raw {
		byMonth[r.Month] = r
	}
	rows := make([]fiber.Map, 0, 12)
	yTiper, yOthers, yHold := decimal.Zero, decimal.Zero, decimal.Zero
	for m := 1; m <= 12; m++ {
		r := byMonth[m]
		total := r.Tiper.Add(r.Others)
		tiperPct, othersPct, holdPct := decimal.Zero, decimal.Zero, decimal.Zero
		if !total.IsZero() {
			tiperPct = r.Tiper.Mul(decimal.NewFromInt(100)).Div(total).Round(0)
			othersPct = r.Others.Mul(decimal.NewFromInt(100)).Div(total).Round(0)
		}
		if !r.Tiper.IsZero() {
			holdPct = r.Hold.Mul(decimal.NewFromInt(100)).Div(r.Tiper).Round(0)
		}
		yTiper, yOthers, yHold = yTiper.Add(r.Tiper), yOthers.Add(r.Others), yHold.Add(r.Hold)
		rows = append(rows, fiber.Map{
			"month": time.Month(m).String(), "total": total,
			"tiper": r.Tiper, "tiperPct": tiperPct,
			"others": r.Others, "othersPct": othersPct,
			"finHold": r.Hold, "finHoldPct": holdPct,
		})
	}
	yTotal := yTiper.Add(yOthers)
	rows = append(rows, fiber.Map{
		"month": fmt.Sprintf("TOTAL %d", year), "total": yTotal,
		"tiper": yTiper, "tiperPct": pct(yTiper, yTotal),
		"others": yOthers, "othersPct": pct(yOthers, yTotal),
		"finHold": yHold, "finHoldPct": pct(yHold, yTiper),
	})
	title := fmt.Sprintf("Received ships market share per month %d", year)
	if route != "" {
		title = route + " " + title
	}
	return serveAny(c, title, "market_share_monthly", rows)
}

func pumpOverOMC(c fiber.Ctx) error {
	from, to := monthBounds(queryYear(c), queryMonth(c))
	type qty struct {
		Name       string
		AgoTransit decimal.Decimal
		AgoLocal   decimal.Decimal
		PmsTransit decimal.Decimal
		PmsLocal   decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS Name,
			SUM(CASE WHEN st.IsTransit = 1 AND (p.Code IN ('1002','AGO')) THEN d.LoadingQty + d.PumpOverQty ELSE 0 END) AS AgoTransit,
			SUM(CASE WHEN st.IsTransit = 0 AND (p.Code IN ('1002','AGO')) THEN d.LoadingQty + d.PumpOverQty ELSE 0 END) AS AgoLocal,
			SUM(CASE WHEN st.IsTransit = 1 AND (p.Code IN ('1001','PMS','MOGAS')) THEN d.LoadingQty + d.PumpOverQty ELSE 0 END) AS PmsTransit,
			SUM(CASE WHEN st.IsTransit = 0 AND (p.Code IN ('1001','PMS','MOGAS')) THEN d.LoadingQty + d.PumpOverQty ELSE 0 END) AS PmsLocal
		FROM StockDailyPosition d
		JOIN Customer c ON c.ID = d.CustomerID
		JOIN Product p ON p.ID = d.ProductID
		JOIN StockStatus st ON st.ID = d.StockStatusID
		WHERE d.PositionDate >= ? AND d.PositionDate <= ?
		GROUP BY c.Name
		HAVING SUM(d.LoadingQty + d.PumpOverQty) <> 0
		ORDER BY c.Name`, from, to).Scan(&rows).Error
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		total := r.AgoTransit.Add(r.AgoLocal).Add(r.PmsTransit).Add(r.PmsLocal)
		out = append(out, fiber.Map{
			"omc": r.Name, "agoTransit": r.AgoTransit, "agoLocal": r.AgoLocal,
			"pmsTransit": r.PmsTransit, "pmsLocal": r.PmsLocal, "total": total,
		})
	}
	return serveAny(c, fmt.Sprintf("Pump-over & loading to OMCs %s %d", time.Month(queryMonth(c)).String(), queryYear(c)),
		"pump_over_omc", out)
}

func billingSummary(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		FeeCode      string
		CurrencyCode string
		Runs         int64
		Quantity     decimal.Decimal
		Amount       decimal.Decimal
		Status       string
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT FeeCode, CurrencyCode, Status, COUNT(*) AS Runs, SUM(Quantity) AS Quantity, SUM(Amount) AS Amount
		FROM BillingRun
		WHERE PeriodStart >= ? AND PeriodStart <= ?
		GROUP BY FeeCode, CurrencyCode, Status
		ORDER BY FeeCode, CurrencyCode, Status`, from, to).Scan(&rows).Error
	return serveAny(c, "Billing summary", "billing_summary", rows)
}

func tankUllage(c fiber.Ctx) error {
	type qty struct {
		TankCode    string
		TankName    string
		ProductCode string
		ProductName string
		Capacity    string
		DeadStock   string
		Dip         string
		Ullage      string
		DipDate     *time.Time
	}
	var rows []qty
	clause, args := activeClause("t.IsActive", queryActive(c))
	_ = db.Db.Raw(`
		SELECT t.Code AS TankCode, t.Name AS TankName,
			ISNULL(p.Code, '') AS ProductCode, ISNULL(p.Name, '') AS ProductName,
			CONVERT(varchar(48), t.MaximumCapacity) AS Capacity,
			CONVERT(varchar(48), t.DeadStock) AS DeadStock,
			CONVERT(varchar(48), ISNULL(d.At20, 0)) AS Dip,
			CONVERT(varchar(48), t.MaximumCapacity - t.DeadStock - ISNULL(d.At20, 0)) AS Ullage,
			d.DipDate
		FROM Tank t
		LEFT JOIN Product p ON p.ID = t.ProductID
		LEFT JOIN PhysicalDip d ON d.TankID = t.ID
			AND d.DipDate = (SELECT MAX(x.DipDate) FROM PhysicalDip x WHERE x.TankID = t.ID)
		WHERE 1 = 1`+clause+`
		ORDER BY t.Code`, args...).Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{
			r.TankCode, r.TankName, fmtProduct(r.ProductCode, r.ProductName),
			fmtQtyStr(r.Capacity), fmtQtyStr(r.DeadStock), fmtQtyStr(r.Dip), fmtQtyStr(r.Ullage), fmtDateOnly(r.DipDate),
		})
	}
	return serveTable(c, titled("Tank ullage", activeNote(c)), "tank_ullage",
		[]string{"S/N", "Tank", "Name", "Product", "Capacity (L)", "Dead stock (L)", "Dip (L)", "Ullage (L)", "Dip date"},
		withSerial(out))
}

func openOrders(c fiber.Ctx) error {
	type qty struct {
		Kind           string
		DocumentNumber string
		CustomerName   string
		ProductCode    string
		Quantity       decimal.Decimal
		Status         string
		OrderDate      time.Time
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT * FROM (
		SELECT 'ILR' AS Kind, r.DocumentNumber, c.Name AS CustomerName, p.Code AS ProductCode,
			r.Quantity, r.Status, r.OrderDate
		FROM GantryLoadingRequest r
		JOIN Customer c ON c.ID = r.CustomerID
		JOIN Product p ON p.ID = r.ProductID
		WHERE r.Status NOT IN ('completed','cancelled','rejected','closed')
		UNION ALL
		SELECT 'PDO', r.DocumentNumber, c.Name, p.Code, r.Quantity, r.Status, r.OrderDate
		FROM PumpOverRequest r
		JOIN Customer c ON c.ID = r.CustomerID
		JOIN Product p ON p.ID = r.ProductID
		WHERE r.Status NOT IN ('completed','cancelled','rejected','closed')
	) x ORDER BY OrderDate DESC`).Scan(&rows).Error
	return serveRegister(c, "Open orders", "open_orders", rows)
}

func licenseExpiry(c fiber.Ctx) error {
	type qty struct {
		Licensee      string
		LicenseNumber string
		Class         string
		ExpiryDate    *time.Time
		DaysLeft      int
		IsActive      bool
	}
	clause, args := activeClause("IsActive", queryActive(c))
	classSQL := ""
	if classes := queryClasses(c); len(classes) > 0 {
		classSQL = " AND LicenseClass IN ?"
		args = append(args, classes)
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT Licensee, LicenseNumber, LicenseClass AS Class, ExpiryDate, IsActive,
			CASE WHEN ExpiryDate IS NULL THEN 0 ELSE DATEDIFF(day, GETDATE(), ExpiryDate) END AS DaysLeft
		FROM EwuraPetroleumLicense
		WHERE 1 = 1`+clause+classSQL+`
		ORDER BY ExpiryDate, Licensee`, args...).Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		days := ""
		if r.ExpiryDate != nil && !r.ExpiryDate.IsZero() {
			days = strconv.Itoa(r.DaysLeft)
		}
		out = append(out, []string{
			r.Licensee, r.LicenseNumber, r.Class, fmtDateOnly(r.ExpiryDate), days, yesNo(r.IsActive),
		})
	}
	classNote := ""
	if classes := queryClasses(c); len(classes) > 0 {
		classNote = strings.Join(classes, ", ")
	}
	return serveTable(c, titled("EWURA license expiry", activeNote(c), classNote), "license_expiry",
		[]string{"S/N", "Licensee", "License number", "Class", "Expiry", "Days left", "Active"},
		withSerial(out))
}

func workflowAging(c fiber.Ctx) error {
	type qty struct {
		ProcessName string
		DocumentNo  string
		NodeName    string
		Status      string
		DaysOpen    int
		CreatedAt   time.Time
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT p.Name AS ProcessName, i.No AS DocumentNo, ISNULL(n.Name, '') AS NodeName,
			i.Status, DATEDIFF(day, i.CreatedAt, GETDATE()) AS DaysOpen, i.CreatedAt
		FROM ProcessInstance i
		JOIN Process p ON p.ID = i.ProcessID
		LEFT JOIN Node n ON n.ID = i.CurNodeID
		WHERE i.Status IN ('draft','running')
		ORDER BY DaysOpen DESC, i.CreatedAt`).Scan(&rows).Error
	return serveRegister(c, "Open workflow aging", "workflow_aging", rows)
}

func stockAging(c fiber.Ctx) error {
	type qty struct {
		CustomerName string
		ProductCode  string
		VesselCode   string
		VesselDate   time.Time
		Quantity     decimal.Decimal
		DaysOnHand   int
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, v.Code AS VesselCode, b.VesselDate,
			SUM(b.Quantity) AS Quantity, DATEDIFF(day, b.VesselDate, GETDATE()) AS DaysOnHand
		FROM StockBalance b
		JOIN Customer c ON c.ID = b.CustomerID
		JOIN Product p ON p.ID = b.ProductID
		JOIN Vessel v ON v.ID = b.VesselID
		WHERE b.IsProvision = 0
		GROUP BY c.Name, p.Code, v.Code, b.VesselDate
		HAVING SUM(b.Quantity) <> 0
		ORDER BY DaysOnHand DESC`).Scan(&rows).Error
	return serveRegister(c, "Stock aging by parcel", "stock_aging", rows)
}

func dailyThroughput(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		Day      time.Time
		Receipts decimal.Decimal
		Loading  decimal.Decimal
		PumpOver decimal.Decimal
		ITT      decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT d.PositionDate AS Day,
			SUM(d.ReceivedQty) AS Receipts,
			SUM(d.LoadingQty) AS Loading,
			SUM(d.PumpOverQty) AS PumpOver,
			SUM(d.ITTQty) AS ITT
		FROM StockDailyPosition d
		WHERE d.PositionDate >= ? AND d.PositionDate <= ?
		GROUP BY d.PositionDate
		ORDER BY Day`, from, to).Scan(&rows).Error
	return serveAny(c, "Daily throughput", "daily_throughput", rows)
}

func productCodes() []string {
	var codes []string
	_ = db.Db.Model(&models.Product{}).Where("IsActive = 1").Order("Code").Pluck("Code", &codes).Error
	if len(codes) == 0 {
		return []string{"AGO", "PMS"}
	}
	return codes
}

func summaryRow(label, kind string, products []string, vals map[string]decimal.Decimal) fiber.Map {
	row := fiber.Map{"row": label, "kind": kind}
	sum := decimal.Zero
	for _, p := range products {
		v := vals[p]
		row[p] = v
		sum = sum.Add(v)
	}
	row["TOTAL"] = sum
	return row
}

type custProdQty struct {
	CustomerName string
	ProductCode  string
	Quantity     decimal.Decimal
}

func pivotCustomerProduct(raw []custProdQty) []fiber.Map {
	products := productCodes()
	by := map[string]map[string]decimal.Decimal{}
	for _, r := range raw {
		if by[r.CustomerName] == nil {
			by[r.CustomerName] = map[string]decimal.Decimal{}
		}
		by[r.CustomerName][r.ProductCode] = r.Quantity
	}
	names := sortedKeys(by)
	out := make([]fiber.Map, 0, len(names))
	for _, name := range names {
		row := fiber.Map{"customer": name}
		sum := decimal.Zero
		for _, p := range products {
			v := by[name][p]
			row[p] = v
			sum = sum.Add(v)
		}
		row["TOTAL"] = sum
		out = append(out, row)
	}
	return out
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func pct(part, total decimal.Decimal) decimal.Decimal {
	if total.IsZero() {
		return decimal.Zero
	}
	return part.Mul(decimal.NewFromInt(100)).Div(total).Round(0)
}
