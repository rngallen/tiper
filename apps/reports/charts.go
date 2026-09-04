package reports

import (
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

type chartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type chartPayload struct {
	Source string       `json:"source"`
	Title  string       `json:"title"`
	Unit   string       `json:"unit,omitempty"`
	Points []chartPoint `json:"points"`
}

func chartCatalog(c fiber.Ctx) error {
	return response.OkDetail(c, []fiber.Map{
		{"code": "kpis", "name": "Operations snapshot", "types": []string{"kpi"}, "description": "Receipts, open billing, licenses, and book stock."},
		{"code": "stock-by-product", "name": "Book stock by product", "types": []string{"bar", "doughnut", "pie", "table"}, "description": "Net ledger quantity per product."},
		{"code": "receipts-by-route", "name": "Receipts by route", "types": []string{"bar", "doughnut", "pie"}, "description": "Approved SBM vs KOJ volume this year."},
		{"code": "market-share-tiper", "name": "TIPER vs others (monthly)", "types": []string{"bar", "line"}, "description": "Internal vs external reception by month."},
		{"code": "throughput-daily", "name": "Daily throughput", "types": []string{"line", "bar"}, "description": "Receipts, loading, and pump-over for the last 30 days."},
		{"code": "pump-over-omc", "name": "Pump-over & loading by OMC", "types": []string{"doughnut", "pie", "bar", "table"}, "description": "Current month outturn by customer."},
		{"code": "tank-ullage", "name": "Tank ullage", "types": []string{"bar", "table"}, "description": "Remaining capacity on the latest dip."},
		{"code": "open-orders", "name": "Open orders", "types": []string{"bar", "kpi", "table"}, "description": "ILR and pump-over documents still in flight."},
		{"code": "billing-open", "name": "Open billing runs", "types": []string{"doughnut", "pie", "bar", "kpi"}, "description": "Draft and submitted billing by fee."},
		{"code": "customer-outstanding", "name": "Customer outstanding", "types": []string{"bar", "pie", "doughnut", "table"}, "description": "Open billing amounts by customer."},
		{"code": "workflow-aging", "name": "Approvals waiting", "types": []string{"bar", "table", "kpi"}, "description": "Open workflow instances by process."},
		{"code": "license-expiry", "name": "Licenses expiring", "types": []string{"bar", "table"}, "description": "EWURA licenses due in the next 90 days."},
		{"code": "stock-aging", "name": "Oldest parcels", "types": []string{"bar", "table"}, "description": "Open stock by days since vessel date."},
	})
}

func chartData(c fiber.Ctx) error {
	source := strings.TrimSpace(c.Params("source"))
	switch source {
	case "kpis":
		return chartKPIs(c)
	case "stock-by-product":
		return chartPairs(c, source, "Book stock by product", "m³", `
			SELECT p.Code AS Label, SUM(b.Quantity) AS Value
			FROM StockBalance b JOIN Product p ON p.ID = b.ProductID
			WHERE b.IsProvision = 0 GROUP BY p.Code ORDER BY p.Code`)
	case "receipts-by-route":
		year := queryYear(c)
		return chartPairs(c, source, "Receipts by route", "m³", `
			SELECT RouteCode AS Label, SUM(CubicMeter) AS Value
			FROM ReceptionFact
			WHERE YEAR(Date) = ?
			GROUP BY RouteCode`, year)
	case "market-share-tiper":
		year := queryYear(c)
		type row struct {
			Month int
			Tiper decimal.Decimal
			Other decimal.Decimal
		}
		var raw []row
		_ = db.Db.Raw(`
			SELECT MONTH(Date) AS Month,
				SUM(CASE WHEN ReceiptType = 'internal' THEN CubicMeter ELSE 0 END) AS Tiper,
				SUM(CASE WHEN ReceiptType <> 'internal' THEN CubicMeter ELSE 0 END) AS Other
			FROM ReceptionFact
			WHERE YEAR(Date) = ?
			GROUP BY MONTH(Date) ORDER BY Month`, year).Scan(&raw).Error
		points := make([]chartPoint, 0, len(raw)*2)
		for _, r := range raw {
			label := time.Month(r.Month).String()[:3]
			points = append(points, chartPoint{Label: label + " TIPER", Value: r.Tiper.InexactFloat64()})
			points = append(points, chartPoint{Label: label + " others", Value: r.Other.InexactFloat64()})
		}
		return response.OkDetail(c, chartPayload{Source: source, Title: "TIPER vs others", Unit: "m³", Points: points})
	case "throughput-daily":
		from := time.Now().AddDate(0, 0, -30)
		return chartPairs(c, source, "Daily throughput (30 days)", "m³", `
			SELECT CONVERT(varchar(10), d.PositionDate, 23) AS Label,
				SUM(d.ReceivedQty + d.LoadingQty + d.PumpOverQty) AS Value
			FROM StockDailyPosition d
			WHERE d.PositionDate >= ?
			GROUP BY CONVERT(varchar(10), d.PositionDate, 23)
			ORDER BY CONVERT(varchar(10), d.PositionDate, 23)`, from)
	case "pump-over-omc":
		from, to := monthBounds(time.Now().Year(), int(time.Now().Month()))
		return chartPairs(c, source, "Pump-over & loading this month", "m³", `
			SELECT TOP 12 c.Name AS Label, SUM(d.LoadingQty + d.PumpOverQty) AS Value
			FROM StockDailyPosition d JOIN Customer c ON c.ID = d.CustomerID
			WHERE d.PositionDate >= ? AND d.PositionDate <= ?
			GROUP BY c.Name
			HAVING SUM(d.LoadingQty + d.PumpOverQty) <> 0
			ORDER BY SUM(d.LoadingQty + d.PumpOverQty) DESC`, from, to)
	case "tank-ullage":
		return chartPairs(c, source, "Tank ullage", "m³", `
			SELECT t.Code AS Label,
				t.MaximumCapacity - t.DeadStock - ISNULL(d.At20, 0) AS Value
			FROM Tank t
			LEFT JOIN PhysicalDip d ON d.TankID = t.ID
				AND d.DipDate = (SELECT MAX(x.DipDate) FROM PhysicalDip x WHERE x.TankID = t.ID)
			WHERE t.IsActive = 1
			ORDER BY t.Code`)
	case "open-orders":
		return chartPairs(c, source, "Open orders", "", `
			SELECT Status AS Label, COUNT(*) AS Value FROM (
				SELECT Status FROM GantryLoadingRequest WHERE Status NOT IN ('completed','cancelled','rejected','closed','expired')
				UNION ALL
				SELECT Status FROM PumpOverRequest WHERE Status NOT IN ('completed','cancelled','rejected','closed','expired')
			) x GROUP BY Status`)
	case "billing-open":
		return chartPairs(c, source, "Open billing runs", "", `
			SELECT FeeCode AS Label, COUNT(*) AS Value
			FROM BillingRun WHERE Status IN ('draft','submitted')
			GROUP BY FeeCode`)
	case "customer-outstanding":
		return chartPairs(c, source, "Customer outstanding", "amount", `
			SELECT TOP 12 c.Name AS Label, SUM(r.Amount) AS Value
			FROM BillingRun r JOIN Customer c ON c.ID = r.CustomerID
			WHERE r.Status IN ('draft','submitted','approved')
			GROUP BY c.Name
			HAVING SUM(r.Amount) <> 0
			ORDER BY SUM(r.Amount) DESC`)
	case "workflow-aging":
		return chartPairs(c, source, "Open approvals", "", `
			SELECT p.Name AS Label, COUNT(*) AS Value
			FROM ProcessInstance i JOIN Process p ON p.ID = i.ProcessID
			WHERE i.Status IN ('draft','running')
			GROUP BY p.Name`)
	case "license-expiry":
		return chartPairs(c, source, "Licenses expiring (90 days)", "", `
			SELECT TOP 12 Licensee AS Label, 1 AS Value
			FROM EwuraPetroleumLicense
			WHERE ExpiryDate IS NOT NULL AND ExpiryDate <= DATEADD(day, 90, GETDATE())
			ORDER BY ExpiryDate`)
	case "stock-aging":
		return chartPairs(c, source, "Oldest open parcels", "m³", `
			SELECT TOP 10 c.Name + ' ' + p.Code AS Label, SUM(b.Quantity) AS Value
			FROM StockBalance b
			JOIN Customer c ON c.ID = b.CustomerID
			JOIN Product p ON p.ID = b.ProductID
			WHERE b.IsProvision = 0
			GROUP BY c.Name, p.Code, b.VesselDate
			HAVING SUM(b.Quantity) <> 0
			ORDER BY MIN(b.VesselDate)`)
	default:
		return response.NotFound(c, "chart source")
	}
}

func chartKPIs(c fiber.Ctx) error {
	var receipts, openRuns, licenses, movements, openWF int64
	_ = db.Db.Model(&models.Receipt{}).Count(&receipts).Error
	_ = db.Db.Model(&models.BillingRun{}).Where("Status IN ?", []string{"draft", "submitted"}).Count(&openRuns).Error
	_ = db.Db.Model(&models.EwuraPetroleumLicense{}).Count(&licenses).Error
	_ = db.Db.Model(&models.StockMovement{}).Count(&movements).Error
	_ = db.Db.Model(&models.ProcessInstance{}).Where("Status IN ?", []string{"draft", "running"}).Count(&openWF).Error
	var book decimal.Decimal
	_ = db.Db.Raw(`SELECT ISNULL(SUM(Quantity), 0) FROM StockBalance WHERE IsProvision = 0`).Scan(&book).Error
	return response.OkDetail(c, chartPayload{
		Source: "kpis",
		Title:  "Operations snapshot",
		Points: []chartPoint{
			{Label: "Vessel receipts", Value: float64(receipts)},
			{Label: "Open billing", Value: float64(openRuns)},
			{Label: "EWURA licenses", Value: float64(licenses)},
			{Label: "Ledger lines", Value: float64(movements)},
			{Label: "Open approvals", Value: float64(openWF)},
			{Label: "Book stock", Value: book.InexactFloat64()},
		},
	})
}

func chartPairs(c fiber.Ctx, source, title, unit, sql string, args ...any) error {
	type row struct {
		Label string
		Value decimal.Decimal
	}
	var raw []row
	_ = db.Db.Raw(sql, args...).Scan(&raw).Error
	points := make([]chartPoint, 0, len(raw))
	for _, r := range raw {
		points = append(points, chartPoint{Label: r.Label, Value: r.Value.InexactFloat64()})
	}
	return response.OkDetail(c, chartPayload{Source: source, Title: title, Unit: unit, Points: points})
}
