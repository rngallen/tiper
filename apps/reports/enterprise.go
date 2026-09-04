package reports

import (
	"fmt"
	"strings"
	"time"

	"dfms/pkg/db"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

// periodicStockStatus is stock at TIPER for a period: on-hand by status
// (transit / local / mining — mining is a local subclass), receipts in the
// window, financial hold, and volume received under SRT contracts.
func periodicStockStatus(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		CustomerCode string
		CustomerName string
		ProductCode  string
		OnHand       decimal.Decimal
		Received     decimal.Decimal
		Transit      decimal.Decimal
		Local        decimal.Decimal
		Mining       decimal.Decimal
		Hold         decimal.Decimal
		SRTReceived  decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Code AS CustomerCode, c.Name AS CustomerName, p.Code AS ProductCode,
			SUM(CASE WHEN l.rn = 1 THEN l.ClosingQty ELSE 0 END) AS OnHand,
			ISNULL(per.ReceivedQty, 0) AS Received,
			SUM(CASE WHEN l.rn = 1 AND st.IsTransit = 1 THEN l.ClosingQty ELSE 0 END) AS Transit,
			SUM(CASE WHEN l.rn = 1 AND st.IsLocal = 1 AND st.IsMining = 0 THEN l.ClosingQty ELSE 0 END) AS Local,
			SUM(CASE WHEN l.rn = 1 AND st.IsMining = 1 THEN l.ClosingQty ELSE 0 END) AS Mining,
			SUM(CASE WHEN l.rn = 1 THEN l.HoldQty ELSE 0 END) AS Hold,
			ISNULL(per.SRTReceivedQty, 0) AS SRTReceived
		FROM (
			SELECT d.*, ROW_NUMBER() OVER (
				PARTITION BY d.CustomerID, d.ProductID, d.StockStatusID
				ORDER BY d.PositionDate DESC) AS rn
			FROM StockDailyPosition d
			WHERE d.PositionDate <= ?
		) l
		JOIN Customer c ON c.ID = l.CustomerID
		JOIN Product p ON p.ID = l.ProductID
		JOIN StockStatus st ON st.ID = l.StockStatusID
		LEFT JOIN (
			SELECT CustomerID, ProductID,
				SUM(ReceivedQty) AS ReceivedQty, SUM(SRTReceivedQty) AS SRTReceivedQty
			FROM StockDailyPosition
			WHERE PositionDate >= ? AND PositionDate <= ?
			GROUP BY CustomerID, ProductID
		) per ON per.CustomerID = l.CustomerID AND per.ProductID = l.ProductID
		WHERE l.rn = 1
		GROUP BY c.Code, c.Name, p.Code, per.ReceivedQty, per.SRTReceivedQty
		ORDER BY c.Name, p.Code`, to, from, to).Scan(&rows).Error
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		out = append(out, fiber.Map{
			"customerCode": r.CustomerCode, "customer": r.CustomerName, "product": r.ProductCode,
			"onHand": r.OnHand, "received": r.Received,
			"transit": r.Transit, "local": r.Local, "mining": r.Mining,
			"localIncludingMining": r.Local.Add(r.Mining),
			"financialHold":        r.Hold, "srtReceived": r.SRTReceived,
		})
	}
	return serveAny(c, "Periodic stock status", "periodic_stock_status", out)
}

// gainLoss compares book stock to physical dips for the selected period.
func gainLoss(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		ProductCode     string
		OpeningBook     decimal.Decimal
		ClosingBook     decimal.Decimal
		OpeningPhysical decimal.Decimal
		ClosingPhysical decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT p.Code AS ProductCode,
			ISNULL((
				SELECT TOP 1 b.BookQty FROM ProductDailyBalance b
				WHERE b.ProductID = p.ID AND b.BalanceDate < ?
				ORDER BY b.BalanceDate DESC
			), 0) AS OpeningBook,
			ISNULL((
				SELECT TOP 1 b.BookQty FROM ProductDailyBalance b
				WHERE b.ProductID = p.ID AND b.BalanceDate <= ?
				ORDER BY b.BalanceDate DESC
			), 0) AS ClosingBook,
			ISNULL((
				SELECT TOP 1 b.PhysicalQty + b.LineQty FROM ProductDailyBalance b
				WHERE b.ProductID = p.ID AND b.BalanceDate < ?
				ORDER BY b.BalanceDate DESC
			), 0) AS OpeningPhysical,
			ISNULL((
				SELECT TOP 1 b.PhysicalQty + b.LineQty FROM ProductDailyBalance b
				WHERE b.ProductID = p.ID AND b.BalanceDate <= ?
				ORDER BY b.BalanceDate DESC
			), 0) AS ClosingPhysical
		FROM Product p
		WHERE p.IsActive = 1
		ORDER BY p.Code`, from, to, from, to).Scan(&rows).Error
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		bookMove := r.ClosingBook.Sub(r.OpeningBook)
		physMove := r.ClosingPhysical.Sub(r.OpeningPhysical)
		out = append(out, fiber.Map{
			"product":     r.ProductCode,
			"openingBook": r.OpeningBook, "closingBook": r.ClosingBook, "bookMovement": bookMove,
			"openingPhysical": r.OpeningPhysical, "closingPhysical": r.ClosingPhysical,
			"physicalMovement": physMove,
			"closingVariance":  r.ClosingPhysical.Sub(r.ClosingBook),
			"periodGainLoss":   physMove.Sub(bookMove),
		})
	}
	return serveAny(c, "Gain / loss (book vs dip)", "gain_loss", out)
}

// ewuraWeekly is the weekly customer stock return for the regulator.
func ewuraWeekly(c fiber.Ctx) error {
	_, to := parseRange(c)
	type qty struct {
		CustomerCode string
		CustomerName string
		ProductCode  string
		Quantity     decimal.Decimal
		Transit      decimal.Decimal
		Local        decimal.Decimal
		Mining       decimal.Decimal
		Hold         decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Code AS CustomerCode, c.Name AS CustomerName, p.Code AS ProductCode,
			SUM(l.ClosingQty) AS Quantity,
			SUM(CASE WHEN st.IsTransit = 1 THEN l.ClosingQty ELSE 0 END) AS Transit,
			SUM(CASE WHEN st.IsLocal = 1 AND st.IsMining = 0 THEN l.ClosingQty ELSE 0 END) AS Local,
			SUM(CASE WHEN st.IsMining = 1 THEN l.ClosingQty ELSE 0 END) AS Mining,
			SUM(l.HoldQty) AS Hold
		FROM (
			SELECT d.*, ROW_NUMBER() OVER (
				PARTITION BY d.CustomerID, d.ProductID, d.StockStatusID
				ORDER BY d.PositionDate DESC) AS rn
			FROM StockDailyPosition d WHERE d.PositionDate <= ?
		) l
		JOIN Customer c ON c.ID = l.CustomerID
		JOIN Product p ON p.ID = l.ProductID
		JOIN StockStatus st ON st.ID = l.StockStatusID
		WHERE l.rn = 1
		GROUP BY c.Code, c.Name, p.Code
		HAVING SUM(l.ClosingQty) <> 0
		ORDER BY c.Name, p.Code`, to).Scan(&rows).Error
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		out = append(out, fiber.Map{
			"customerCode": r.CustomerCode, "customer": r.CustomerName, "product": r.ProductCode,
			"quantityStored": r.Quantity, "transit": r.Transit, "local": r.Local,
			"mining": r.Mining, "financialHold": r.Hold,
		})
	}
	return serveAny(c, "EWURA weekly stock return", "ewura_weekly", out)
}

// pbpaWeekly is the PBPA system book-stock overview by product and status.
func pbpaWeekly(c fiber.Ctx) error {
	_, to := parseRange(c)
	type qty struct {
		ProductCode string
		Transit     decimal.Decimal
		Local       decimal.Decimal
		Mining      decimal.Decimal
		Proration   decimal.Decimal
		Hold        decimal.Decimal
		Total       decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT p.Code AS ProductCode,
			SUM(CASE WHEN st.IsTransit = 1 THEN l.ClosingQty ELSE 0 END) AS Transit,
			SUM(CASE WHEN st.IsLocal = 1 AND st.IsMining = 0 THEN l.ClosingQty ELSE 0 END) AS Local,
			SUM(CASE WHEN st.IsMining = 1 THEN l.ClosingQty ELSE 0 END) AS Mining,
			SUM(CASE WHEN st.IsProration = 1 THEN l.ClosingQty ELSE 0 END) AS Proration,
			SUM(l.HoldQty) AS Hold,
			SUM(l.ClosingQty) AS Total
		FROM (
			SELECT d.*, ROW_NUMBER() OVER (
				PARTITION BY d.CustomerID, d.ProductID, d.StockStatusID
				ORDER BY d.PositionDate DESC) AS rn
			FROM StockDailyPosition d WHERE d.PositionDate <= ?
		) l
		JOIN Product p ON p.ID = l.ProductID
		JOIN StockStatus st ON st.ID = l.StockStatusID
		WHERE l.rn = 1
		GROUP BY p.Code
		ORDER BY p.Code`, to).Scan(&rows).Error
	out := make([]fiber.Map, 0, len(rows)+1)
	var tT, tL, tM, tP, tH, tAll decimal.Decimal
	for _, r := range rows {
		tT, tL, tM = tT.Add(r.Transit), tL.Add(r.Local), tM.Add(r.Mining)
		tP, tH, tAll = tP.Add(r.Proration), tH.Add(r.Hold), tAll.Add(r.Total)
		out = append(out, fiber.Map{
			"product": r.ProductCode, "transit": r.Transit, "local": r.Local,
			"mining": r.Mining, "proration": r.Proration, "financialHold": r.Hold, "total": r.Total,
		})
	}
	out = append(out, fiber.Map{
		"product": "TOTAL", "transit": tT, "local": tL, "mining": tM,
		"proration": tP, "financialHold": tH, "total": tAll,
	})
	return serveAny(c, "PBPA weekly book stock", "pbpa_weekly", out)
}

// individualCustomerStock is per-customer balances, flows, hold, and transit
// destinations (Congo, Malawi, Zambia, …) from gantry loadings in the period.
func individualCustomerStock(c fiber.Ctx) error {
	from, to := parseRange(c)
	cust := strings.TrimSpace(c.Query("customer"))
	if cust == "" {
		cust = strings.TrimSpace(c.Query("id"))
	}
	type qty struct {
		CustomerCode string
		CustomerName string
		ProductCode  string
		Opening      decimal.Decimal
		Inflow       decimal.Decimal
		Outflow      decimal.Decimal
		Closing      decimal.Decimal
		Transit      decimal.Decimal
		Local        decimal.Decimal
		Hold         decimal.Decimal
	}
	q := `
		SELECT c.Code AS CustomerCode, c.Name AS CustomerName, p.Code AS ProductCode,
			ISNULL(op.ClosingQty, 0) AS Opening,
			ISNULL(per.Inflow, 0) AS Inflow,
			ISNULL(per.Outflow, 0) AS Outflow,
			ISNULL(cl.ClosingQty, 0) AS Closing,
			ISNULL(cl.Transit, 0) AS Transit,
			ISNULL(cl.LocalQty, 0) AS Local,
			ISNULL(cl.HoldQty, 0) AS Hold
		FROM Customer c
		JOIN Product p ON 1 = 1
		LEFT JOIN (
			SELECT d.CustomerID, d.ProductID, SUM(d.ClosingQty) AS ClosingQty
			FROM (
				SELECT x.*, ROW_NUMBER() OVER (PARTITION BY x.CustomerID, x.ProductID, x.StockStatusID ORDER BY x.PositionDate DESC) rn
				FROM StockDailyPosition x WHERE x.PositionDate < ?
			) d WHERE d.rn = 1
			GROUP BY d.CustomerID, d.ProductID
		) op ON op.CustomerID = c.ID AND op.ProductID = p.ID
		LEFT JOIN (
			SELECT CustomerID, ProductID, SUM(ReceivedQty) AS Inflow, SUM(OutflowQty) AS Outflow
			FROM StockDailyPosition WHERE PositionDate >= ? AND PositionDate <= ?
			GROUP BY CustomerID, ProductID
		) per ON per.CustomerID = c.ID AND per.ProductID = p.ID
		LEFT JOIN (
			SELECT d.CustomerID, d.ProductID, SUM(d.ClosingQty) AS ClosingQty,
				SUM(CASE WHEN st.IsTransit = 1 THEN d.ClosingQty ELSE 0 END) AS Transit,
				SUM(CASE WHEN st.IsLocal = 1 THEN d.ClosingQty ELSE 0 END) AS LocalQty,
				SUM(d.HoldQty) AS HoldQty
			FROM (
				SELECT x.*, ROW_NUMBER() OVER (PARTITION BY x.CustomerID, x.ProductID, x.StockStatusID ORDER BY x.PositionDate DESC) rn
				FROM StockDailyPosition x WHERE x.PositionDate <= ?
			) d
			JOIN StockStatus st ON st.ID = d.StockStatusID
			WHERE d.rn = 1
			GROUP BY d.CustomerID, d.ProductID
		) cl ON cl.CustomerID = c.ID AND cl.ProductID = p.ID
		WHERE p.IsActive = 1`
	args := []any{from, from, to, to}
	if cust != "" {
		q += ` AND (c.UID = ? OR c.Code = ?)`
		args = append(args, cust, cust)
	}
	q += ` AND (ISNULL(op.ClosingQty,0) <> 0 OR ISNULL(per.Inflow,0) <> 0 OR ISNULL(per.Outflow,0) <> 0 OR ISNULL(cl.ClosingQty,0) <> 0)
		ORDER BY c.Name, p.Code`
	var rows []qty
	_ = db.Db.Raw(q, args...).Scan(&rows).Error

	type dest struct {
		CustomerCode string
		ProductCode  string
		Destination  string
		Quantity     decimal.Decimal
	}
	var dests []dest
	dq := `
		SELECT c.Code AS CustomerCode, p.Code AS ProductCode,
			ISNULL(NULLIF(LTRIM(RTRIM(l.Destination)), ''), 'Unspecified') AS Destination,
			SUM(l.LoadedQty) AS Quantity
		FROM GantryLoadingLine l
		JOIN GantryLoadingRequest r ON r.ID = l.RequestID
		JOIN Customer c ON c.ID = r.CustomerID
		JOIN Product p ON p.ID = r.ProductID
		JOIN StockStatus st ON st.ID = r.StockStatusID
		WHERE st.IsTransit = 1 AND l.LoadedQty <> 0
			AND ISNULL(l.LoadedAt, r.OrderDate) >= ? AND ISNULL(l.LoadedAt, r.OrderDate) <= ?`
	dargs := []any{from, to}
	if cust != "" {
		dq += ` AND (c.UID = ? OR c.Code = ?)`
		dargs = append(dargs, cust, cust)
	}
	dq += ` GROUP BY c.Code, p.Code, ISNULL(NULLIF(LTRIM(RTRIM(l.Destination)), ''), 'Unspecified')`
	_ = db.Db.Raw(dq, dargs...).Scan(&dests).Error
	destMap := map[string][]fiber.Map{}
	for _, d := range dests {
		key := d.CustomerCode + "|" + d.ProductCode
		destMap[key] = append(destMap[key], fiber.Map{"destination": d.Destination, "quantity": d.Quantity})
	}

	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		out = append(out, fiber.Map{
			"customerCode": r.CustomerCode, "customer": r.CustomerName, "product": r.ProductCode,
			"opening": r.Opening, "inflow": r.Inflow, "outflow": r.Outflow, "closing": r.Closing,
			"transit": r.Transit, "local": r.Local, "financialHold": r.Hold,
			"transitDestinations": destMap[r.CustomerCode+"|"+r.ProductCode],
		})
	}
	return serveAny(c, "Individual customer stock", "customer_stock", out)
}

// dailyMassBalance is EOD book vs physical dip by product, with cumulative gain/loss.
func dailyMassBalance(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		Day         time.Time
		ProductCode string
		Book        decimal.Decimal
		Physical    decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT b.BalanceDate AS Day, p.Code AS ProductCode, b.BookQty AS Book,
			b.PhysicalQty + b.LineQty AS Physical
		FROM ProductDailyBalance b
		JOIN Product p ON p.ID = b.ProductID
		WHERE b.BalanceDate >= ? AND b.BalanceDate <= ?
		ORDER BY b.BalanceDate, p.Code`, from, to).Scan(&rows).Error
	cum := map[string]decimal.Decimal{}
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		variance := r.Physical.Sub(r.Book)
		cum[r.ProductCode] = cum[r.ProductCode].Add(variance)
		out = append(out, fiber.Map{
			"date": r.Day.Format("02/01/2006"), "product": r.ProductCode,
			"book": r.Book, "physical": r.Physical,
			"gainLoss": variance, "cumulativeGainLoss": cum[r.ProductCode],
		})
	}
	return serveAny(c, "Daily book stock mass balance", "daily_mass_balance", out)
}

// monthlyStockConfirmation is opening / receipts / adjustments / pump-over / closing.
func monthlyStockConfirmation(c fiber.Ctx) error {
	from, to := monthBounds(queryYear(c), queryMonth(c))
	type qty struct {
		CustomerCode string
		CustomerName string
		ProductCode  string
		Opening      decimal.Decimal
		Receipts     decimal.Decimal
		Adjustments  decimal.Decimal
		PumpOvers    decimal.Decimal
		Loading      decimal.Decimal
		ITT          decimal.Decimal
		Closing      decimal.Decimal
	}
	openAsOf := from.AddDate(0, 0, -1)
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Code AS CustomerCode, c.Name AS CustomerName, p.Code AS ProductCode,
			ISNULL(op.ClosingQty, 0) AS Opening,
			ISNULL(per.ReceivedQty, 0) AS Receipts,
			ISNULL(per.AdjustmentQty, 0) AS Adjustments,
			ISNULL(per.PumpOverQty, 0) AS PumpOvers,
			ISNULL(per.LoadingQty, 0) AS Loading,
			ISNULL(per.ITTQty, 0) AS ITT,
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
	return serveAny(c, fmt.Sprintf("Monthly stock confirmation %s %d", time.Month(queryMonth(c)).String(), queryYear(c)),
		"monthly_stock_confirmation", rows)
}

// ittSummary lists inter-customer transfers in the period.
func ittSummary(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		DocumentNumber string
		TransferDate   time.Time
		FromCustomer   string
		ToCustomer     string
		ProductCode    string
		Quantity       decimal.Decimal
		Status         string
		ApprovalRef    string
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT t.DocumentNumber, t.TransferDate, f.Name AS FromCustomer, x.Name AS ToCustomer,
			p.Code AS ProductCode, t.Quantity, t.Status,
			ISNULL(i.No, t.DocumentNumber) AS ApprovalRef
		FROM IttTransfer t
		JOIN Customer f ON f.ID = t.FromCustomerID
		JOIN Customer x ON x.ID = t.ToCustomerID
		JOIN Product p ON p.ID = t.ProductID
		LEFT JOIN ProcessInstance i ON i.DocContentType = ? AND i.ObjectID = t.ID
		WHERE t.TransferDate >= ? AND t.TransferDate <= ?
		ORDER BY t.TransferDate, t.DocumentNumber`, types.IttTransferContent, from, to).Scan(&rows).Error
	return serveRegister(c, "ITT summary", "itt_summary", rows)
}

// srtVessels lists Single Receiving Terminal vessels and parcel status.
func srtVessels(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		VesselCode   string
		VesselDate   time.Time
		ProductCode  string
		CustomerName string
		Quantity     decimal.Decimal
		Transit      decimal.Decimal
		Local        decimal.Decimal
		Mining       decimal.Decimal
		Hold         decimal.Decimal
		Status       string
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT v.Code AS VesselCode, r.VesselDate, p.Code AS ProductCode, c.Name AS CustomerName,
			d.CubicMeter AS Quantity,
			CASE WHEN st.IsTransit = 1 THEN d.CubicMeter ELSE 0 END AS Transit,
			CASE WHEN st.IsLocal = 1 AND st.IsMining = 0 THEN d.CubicMeter ELSE 0 END AS Local,
			CASE WHEN st.IsMining = 1 THEN d.CubicMeter ELSE 0 END AS Mining,
			CASE WHEN d.FinancialHold = 1 THEN d.CubicMeter ELSE 0 END AS Hold,
			r.Status
		FROM ReceiptDetail d
		JOIN Receipt r ON r.ID = d.ReceiptID
		JOIN Vessel v ON v.ID = r.VesselID
		JOIN Product p ON p.ID = r.ProductID
		JOIN Customer c ON c.ID = d.CustomerID
		JOIN StockStatus st ON st.ID = d.StockStatusID
		JOIN ImportTenderType tender ON tender.Code = r.TenderCode AND tender.IsSingleReceiving = 1
		WHERE r.Date >= ? AND r.Date <= ? AND d.OriginDetailID IS NULL
		ORDER BY r.VesselDate, v.Code, c.Name`, from, to).Scan(&rows).Error
	return serveRegister(c, "SRT vessel report", "srt_vessels", rows)
}

// routeReceiptsShare is SBM or KOJ receipts with TIPER vs other-terminal
// share per vessel for the period and on a cumulative (year-to-date) basis.
func routeReceiptsShare(route string) fiber.Handler {
	return func(c fiber.Ctx) error {
		from, to := parseRange(c)
		type qty struct {
			VesselDate       time.Time
			VesselName       string
			PeriodTiper      decimal.Decimal
			PeriodOthers     decimal.Decimal
			CumulativeTiper  decimal.Decimal
			CumulativeOthers decimal.Decimal
		}
		var rows []qty
		_ = db.Db.Raw(`
			SELECT Date AS VesselDate, VesselName,
				SUM(CASE WHEN Date >= ? AND Date <= ? AND ReceiptType = 'internal' THEN CubicMeter ELSE 0 END) AS PeriodTiper,
				SUM(CASE WHEN Date >= ? AND Date <= ? AND ReceiptType <> 'internal' THEN CubicMeter ELSE 0 END) AS PeriodOthers,
				SUM(CASE WHEN ReceiptType = 'internal' THEN CubicMeter ELSE 0 END) AS CumulativeTiper,
				SUM(CASE WHEN ReceiptType <> 'internal' THEN CubicMeter ELSE 0 END) AS CumulativeOthers
			FROM ReceptionFact
			WHERE RouteCode = ? AND Date <= ?
			GROUP BY Date, VesselName
			HAVING SUM(CASE WHEN Date >= ? AND Date <= ? THEN CubicMeter ELSE 0 END) <> 0
			ORDER BY Date, VesselName`,
			from, to, from, to, route, to, from, to,
		).Scan(&rows).Error
		out := make([]fiber.Map, 0, len(rows))
		for _, r := range rows {
			pTot := r.PeriodTiper.Add(r.PeriodOthers)
			cTot := r.CumulativeTiper.Add(r.CumulativeOthers)
			out = append(out, fiber.Map{
				"date": r.VesselDate.Format("02-Jan-06"), "vessel": r.VesselName,
				"periodTiper": r.PeriodTiper, "periodOthers": r.PeriodOthers, "periodTotal": pTot,
				"periodTiperPct": pct(r.PeriodTiper, pTot), "periodOthersPct": pct(r.PeriodOthers, pTot),
				"cumulativeTiper": r.CumulativeTiper, "cumulativeOthers": r.CumulativeOthers, "cumulativeTotal": cTot,
				"cumulativeTiperPct": pct(r.CumulativeTiper, cTot), "cumulativeOthersPct": pct(r.CumulativeOthers, cTot),
			})
		}
		title := route + " receipts — TIPER vs other terminals"
		if route == "SBM" {
			title = "SBM (Single Buoy Mooring) receipts — TIPER vs other terminals"
		} else if route == "KOJ" {
			title = "KOJ (Kurasini Oil Jetty) receipts — TIPER vs other terminals"
		}
		return serveAny(c, title, strings.ToLower(route)+"_receipts_share", out)
	}
}

// pumpOverStatus is product moved from TIPER to other terminals by status.
func pumpOverStatus(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		CustomerName string
		ProductCode  string
		DepotName    string
		Transit      decimal.Decimal
		Local        decimal.Decimal
		Mining       decimal.Decimal
		Total        decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, '' AS DepotName,
			SUM(CASE WHEN st.IsTransit = 1 THEN d.PumpOverQty ELSE 0 END) AS Transit,
			SUM(CASE WHEN st.IsLocal = 1 AND st.IsMining = 0 THEN d.PumpOverQty ELSE 0 END) AS Local,
			SUM(CASE WHEN st.IsMining = 1 THEN d.PumpOverQty ELSE 0 END) AS Mining,
			SUM(d.PumpOverQty) AS Total
		FROM StockDailyPosition d
		JOIN Customer c ON c.ID = d.CustomerID
		JOIN Product p ON p.ID = d.ProductID
		JOIN StockStatus st ON st.ID = d.StockStatusID
		WHERE d.PositionDate >= ? AND d.PositionDate <= ?
		GROUP BY c.Name, p.Code
		HAVING SUM(d.PumpOverQty) <> 0
		ORDER BY c.Name, p.Code, DepotName`, from, to).Scan(&rows).Error
	return serveAny(c, "Pump-over to other terminals", "pump_over_status", rows)
}

func billingList(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		DocumentNumber string
		PeriodStart    time.Time
		CustomerName   string
		FeeCode        string
		CurrencyCode   string
		Quantity       decimal.Decimal
		Amount         decimal.Decimal
		Status         string
		Source         string
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT r.DocumentNumber, r.PeriodStart, ISNULL(c.Name, '') AS CustomerName,
			r.FeeCode, r.CurrencyCode, r.Quantity, r.Amount, r.Status, ISNULL(r.Source, '') AS Source
		FROM BillingRun r
		LEFT JOIN Customer c ON c.ID = r.CustomerID
		WHERE r.PeriodStart >= ? AND r.PeriodStart <= ?
		ORDER BY r.PeriodStart, r.DocumentNumber`, from, to).Scan(&rows).Error
	return serveRegister(c, "Billing / invoice list", "billing_list", rows)
}

func billingByTrade(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		ClassOfTrade string
		FeeCode      string
		Runs         int64
		Quantity     decimal.Decimal
		Amount       decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT ISNULL(rec.TenderCode, 'Unassigned') AS ClassOfTrade, r.FeeCode,
			COUNT(*) AS Runs, SUM(r.Quantity) AS Quantity, SUM(r.Amount) AS Amount
		FROM BillingRun r
		LEFT JOIN ReceiptDetail d ON d.ID = r.ReceiptDetailID
		LEFT JOIN Receipt rec ON rec.ID = d.ReceiptID
		WHERE r.PeriodStart >= ? AND r.PeriodStart <= ?
		GROUP BY ISNULL(rec.TenderCode, 'Unassigned'), r.FeeCode
		ORDER BY ClassOfTrade, r.FeeCode`, from, to).Scan(&rows).Error
	return serveAny(c, "Billing by class of trade", "billing_by_trade", rows)
}

func billingByProduct(c fiber.Ctx) error {
	from, to := parseRange(c)
	type qty struct {
		ProductCode string
		FeeCode     string
		Runs        int64
		Quantity    decimal.Decimal
		Amount      decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT ISNULL(p.Code, 'Unassigned') AS ProductCode, r.FeeCode,
			COUNT(*) AS Runs, SUM(r.Quantity) AS Quantity, SUM(r.Amount) AS Amount
		FROM BillingRun r
		LEFT JOIN ReceiptDetail d ON d.ID = r.ReceiptDetailID
		LEFT JOIN Receipt rec ON rec.ID = d.ReceiptID
		LEFT JOIN Product p ON p.ID = rec.ProductID
		WHERE r.PeriodStart >= ? AND r.PeriodStart <= ?
		GROUP BY ISNULL(p.Code, 'Unassigned'), r.FeeCode
		ORDER BY ProductCode, r.FeeCode`, from, to).Scan(&rows).Error
	return serveAny(c, "Billing by product", "billing_by_product", rows)
}

func holdRegister(c fiber.Ctx) error {
	type qty struct {
		CustomerName string
		ProductCode  string
		VesselCode   string
		VesselDate   time.Time
		Quantity     decimal.Decimal
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, v.Code AS VesselCode, b.VesselDate,
			SUM(b.Quantity) AS Quantity
		FROM StockBalance b
		JOIN Customer c ON c.ID = b.CustomerID
		JOIN Product p ON p.ID = b.ProductID
		JOIN Vessel v ON v.ID = b.VesselID
		WHERE b.IsProvision = 0 AND b.FinancialHold = 1
		GROUP BY c.Name, p.Code, v.Code, b.VesselDate
		HAVING SUM(b.Quantity) <> 0
		ORDER BY c.Name, p.Code`).Scan(&rows).Error
	return serveRegister(c, "Financial hold register", "hold_register", rows)
}

func reservationExposure(c fiber.Ctx) error {
	type qty struct {
		CustomerName string
		ProductCode  string
		OrderNumber  string
		Quantity     decimal.Decimal
		Status       string
		ExpiresAt    *time.Time
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, p.Code AS ProductCode, r.OrderNumber, r.Quantity, r.Status, r.ExpiresAt
		FROM StockReservation r
		JOIN Customer c ON c.ID = r.CustomerID
		JOIN Product p ON p.ID = r.ProductID
		WHERE r.Status = 'open'
		ORDER BY r.ExpiresAt, c.Name`).Scan(&rows).Error
	return serveRegister(c, "Open stock reservations", "reservation_exposure", rows)
}

func billingExceptions(c fiber.Ctx) error {
	type qty struct {
		Reason     string
		Status     string
		ValidUntil time.Time
		CreatedAt  time.Time
	}
	var rows []qty
	_ = db.Db.Raw(`
		SELECT Reason, Status, ValidUntil, CreatedAt
		FROM BillingException
		ORDER BY CreatedAt DESC`).Scan(&rows).Error
	return serveRegister(c, "Billing exceptions", "billing_exceptions", rows)
}
