package reports

import (
	"time"

	"dfms/pkg/db"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

func yearlyLoading(c fiber.Ctx) error {
	year := queryYear(c)
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(year, 12, 31, 23, 59, 59, 0, time.Local)
	type row struct {
		Month      int
		AgoLocal   decimal.Decimal
		AgoTransit decimal.Decimal
		PmsLocal   decimal.Decimal
		PmsTransit decimal.Decimal
		Trucks     int64
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT MONTH(r.LoadedAt) AS Month,
			SUM(CASE WHEN st.IsTransit = 0 AND (p.Code IN ('1002','AGO')) THEN lp.StandardVolume ELSE 0 END) AS AgoLocal,
			SUM(CASE WHEN st.IsTransit = 1 AND (p.Code IN ('1002','AGO')) THEN lp.StandardVolume ELSE 0 END) AS AgoTransit,
			SUM(CASE WHEN st.IsTransit = 0 AND (p.Code IN ('1001','PMS','MOGAS')) THEN lp.StandardVolume ELSE 0 END) AS PmsLocal,
			SUM(CASE WHEN st.IsTransit = 1 AND (p.Code IN ('1001','PMS','MOGAS')) THEN lp.StandardVolume ELSE 0 END) AS PmsTransit,
			COUNT(DISTINCT r.ID) AS Trucks
		FROM GantryLoading r
		JOIN GantryLoadingProduct lp ON lp.LoadingID = r.ID
		JOIN GantryLoadingLine l ON l.ID = r.IloID
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Product p ON p.ID = lp.ProductID
		JOIN StockStatus st ON st.ID = req.StockStatusID
		WHERE r.LoadedAt >= ? AND r.LoadedAt <= ?
		GROUP BY MONTH(r.LoadedAt)
		ORDER BY Month`, from, to).Scan(&rows).Error
	return serveAny(c, "Yearly loading summary", "yearly_loading", rows)
}

func monthlyLoading(c fiber.Ctx) error {
	from, to := monthBounds(queryYear(c), queryMonth(c))
	type row struct {
		LoadedAt       time.Time
		DocumentNumber string
		CustomerName   string
		ProductCode    string
		StatusName     string
		Destination    string
		TruckPlate     string
		ObservedVolume decimal.Decimal
		StandardVolume decimal.Decimal
		Weight         decimal.Decimal
		IsTransit      bool
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT r.LoadedAt, l.DocumentNumber, c.Name AS CustomerName, p.Code AS ProductCode,
			st.Name AS StatusName, l.Destination, l.TruckPlate,
			lp.ObservedVolume, lp.StandardVolume, lp.Weight, st.IsTransit
		FROM GantryLoading r
		JOIN GantryLoadingProduct lp ON lp.LoadingID = r.ID
		JOIN GantryLoadingLine l ON l.ID = r.IloID
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = lp.ProductID
		JOIN StockStatus st ON st.ID = req.StockStatusID
		WHERE r.LoadedAt >= ? AND r.LoadedAt <= ?
		ORDER BY r.LoadedAt, c.Name`, from, to).Scan(&rows).Error
	return serveAny(c, "Monthly loading summary", "monthly_loading", rows)
}

func dailyLoading(c fiber.Ctx) error {
	from, to := dayBounds(queryDate(c))
	type row struct {
		CustomerName string
		Trucks       int64
		AgoObserved  decimal.Decimal
		AgoStandard  decimal.Decimal
		PmsObserved  decimal.Decimal
		PmsStandard  decimal.Decimal
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, COUNT(DISTINCT r.ID) AS Trucks,
			SUM(CASE WHEN p.Code IN ('1002','AGO') THEN lp.ObservedVolume ELSE 0 END) AS AgoObserved,
			SUM(CASE WHEN p.Code IN ('1002','AGO') THEN lp.StandardVolume ELSE 0 END) AS AgoStandard,
			SUM(CASE WHEN p.Code IN ('1001','PMS','MOGAS') THEN lp.ObservedVolume ELSE 0 END) AS PmsObserved,
			SUM(CASE WHEN p.Code IN ('1001','PMS','MOGAS') THEN lp.StandardVolume ELSE 0 END) AS PmsStandard
		FROM GantryLoading r
		JOIN GantryLoadingProduct lp ON lp.LoadingID = r.ID
		JOIN GantryLoadingLine l ON l.ID = r.IloID
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = lp.ProductID
		WHERE r.LoadedAt >= ? AND r.LoadedAt <= ?
		GROUP BY c.Name
		ORDER BY c.Name`, from, to).Scan(&rows).Error
	return serveRegister(c, "Daily loaded trucks", "daily_loading", rows)
}

func loadingPlan(c fiber.Ctx) error {
	asOf := queryDate(c)
	type row struct {
		CustomerName  string
		RequestNumber string
		StatusName    string
		Trucks        int64
		AgoQuantity   decimal.Decimal
		PmsQuantity   decimal.Decimal
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, req.DocumentNumber AS RequestNumber, st.Name AS StatusName,
			COUNT(*) AS Trucks,
			SUM(CASE WHEN p.Code IN ('1002','AGO') THEN l.RequestedQty ELSE 0 END) AS AgoQuantity,
			SUM(CASE WHEN p.Code IN ('1001','PMS','MOGAS') THEN l.RequestedQty ELSE 0 END) AS PmsQuantity
		FROM GantryLoadingLine l
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = req.ProductID
		JOIN StockStatus st ON st.ID = req.StockStatusID
		WHERE l.Amended = 0 AND l.IsActive = 1
			AND l.Status NOT IN (?, ?, ?, ?)
			AND req.OrderDate <= ?
			AND (l.LoadedAt IS NULL OR l.LoadedAt > ?)
		GROUP BY c.Name, req.DocumentNumber, st.Name
		ORDER BY c.Name, req.DocumentNumber`,
		types.OrderLoaded, types.OrderClosed, types.OrderCancelled, types.OrderRejected,
		asOf, asOf).Scan(&rows).Error
	return serveRegister(c, "Loading plan", "loading_plan", rows)
}

func transitDestination(c fiber.Ctx) error {
	from, to := monthBounds(queryYear(c), queryMonth(c))
	type row struct {
		CustomerName string
		Destination  string
		AgoStandard  decimal.Decimal
		PmsStandard  decimal.Decimal
		Trucks       int64
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, l.Destination,
			SUM(CASE WHEN p.Code IN ('1002','AGO') THEN lp.StandardVolume ELSE 0 END) AS AgoStandard,
			SUM(CASE WHEN p.Code IN ('1001','PMS','MOGAS') THEN lp.StandardVolume ELSE 0 END) AS PmsStandard,
			COUNT(DISTINCT r.ID) AS Trucks
		FROM GantryLoading r
		JOIN GantryLoadingProduct lp ON lp.LoadingID = r.ID
		JOIN GantryLoadingLine l ON l.ID = r.IloID
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = lp.ProductID
		JOIN StockStatus st ON st.ID = req.StockStatusID
		WHERE st.IsTransit = 1 AND r.LoadedAt >= ? AND r.LoadedAt <= ?
		GROUP BY c.Name, l.Destination
		ORDER BY c.Name, l.Destination`, from, to).Scan(&rows).Error
	return serveAny(c, "Transit by destination", "transit_destination", rows)
}

func ewuraLoadedTrucks(c fiber.Ctx) error {
	from, to := dayBounds(queryDate(c))
	type row struct {
		CustomerName   string
		DocumentNumber string
		TruckPlate     string
		Destination    string
		ProductCode    string
		StandardVolume decimal.Decimal
		EwuraLicense   string
		LoadedAt       time.Time
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT c.Name AS CustomerName, l.DocumentNumber, l.TruckPlate, l.Destination,
			p.Code AS ProductCode, lp.StandardVolume, l.EwuraLicense, r.LoadedAt
		FROM GantryLoading r
		JOIN GantryLoadingProduct lp ON lp.LoadingID = r.ID
		JOIN GantryLoadingLine l ON l.ID = r.IloID
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = lp.ProductID
		WHERE r.LoadedAt >= ? AND r.LoadedAt <= ?
		ORDER BY c.Name, r.LoadedAt`, from, to).Scan(&rows).Error
	return serveRegister(c, "EWURA loaded trucks", "ewura_loaded_trucks", rows)
}

func markedFuel(c fiber.Ctx) error {
	year, month := queryYear(c), queryMonth(c)
	start, end := monthBounds(year, month)
	mid := time.Date(year, time.Month(month), 15, 0, 0, 0, 0, time.Local)
	type row struct {
		Period         string
		LoadedAt       time.Time
		CustomerName   string
		Destination    string
		ProductCode    string
		ObservedVolume decimal.Decimal
		StandardVolume decimal.Decimal
		Weight         decimal.Decimal
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT CASE WHEN r.LoadedAt < ? THEN '1-14' ELSE '15-end' END AS Period,
			r.LoadedAt, c.Name AS CustomerName, l.Destination, p.Code AS ProductCode,
			lp.ObservedVolume, lp.StandardVolume, lp.Weight
		FROM GantryLoading r
		JOIN GantryLoadingProduct lp ON lp.LoadingID = r.ID
		JOIN GantryLoadingLine l ON l.ID = r.IloID
		JOIN GantryLoadingRequest req ON req.ID = l.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = lp.ProductID
		JOIN StockStatus st ON st.ID = req.StockStatusID
		WHERE st.IsTransit = 0 AND r.LoadedAt >= ? AND r.LoadedAt <= ?
		ORDER BY r.LoadedAt, c.Name`, mid, start, end).Scan(&rows).Error
	return serveAny(c, "Marked / local fuel", "marked_fuel", rows)
}

func glrStatus(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		DocumentNumber string
		OrderDate      time.Time
		CustomerName   string
		ProductCode    string
		Quantity       decimal.Decimal
		Status         string
		Lines          int64
		LoadedLines    int64
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT req.DocumentNumber, req.OrderDate, c.Name AS CustomerName, p.Code AS ProductCode,
			req.Quantity, req.Status,
			COUNT(l.ID) AS Lines,
			SUM(CASE WHEN l.Status IN (?, ?) THEN 1 ELSE 0 END) AS LoadedLines
		FROM GantryLoadingRequest req
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = req.ProductID
		LEFT JOIN GantryLoadingLine l ON l.RequestID = req.ID AND l.Amended = 0
		WHERE req.OrderDate >= ? AND req.OrderDate <= ?
		GROUP BY req.DocumentNumber, req.OrderDate, c.Name, p.Code, req.Quantity, req.Status
		ORDER BY req.OrderDate`, types.OrderLoaded, types.OrderClosed, from, to).Scan(&rows).Error
	return serveRegister(c, "ILR loading status", "ilr_status", rows)
}

func glrApprovals(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		DocumentNumber string
		OrderDate      time.Time
		CustomerName   string
		ProductCode    string
		Quantity       decimal.Decimal
		Status         string
		BatchNumber    string
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT req.DocumentNumber, req.OrderDate, c.Name AS CustomerName, p.Code AS ProductCode,
			req.Quantity, req.Status, req.BatchNumber
		FROM GantryLoadingRequest req
		JOIN Customer c ON c.ID = req.CustomerID
		JOIN Product p ON p.ID = req.ProductID
		WHERE req.OrderDate >= ? AND req.OrderDate <= ?
			AND req.Status IN (?, ?, ?)
		ORDER BY req.OrderDate`, from, to, types.OrderApproved, types.OrderCompleted, types.OrderClosed).Scan(&rows).Error
	return serveRegister(c, "ILR approvals", "ilr_approvals", rows)
}

func truckSeals(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		LoadedAt       time.Time
		DocumentNumber string
		TruckPlate     string
		TankPlate      string
		CompIndex      int
		ProductCode    string
		Quantity       decimal.Decimal
		TopSeal        string
		DipSeal        string
		BottomSeal     string
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT ISNULL(r.LoadedAt, l.LoadedAt) AS LoadedAt, comp.DocumentNumber,
			ISNULL(l.TruckPlate, comp.HorsePlate) AS TruckPlate,
			cl.TankPlate, cl.[Index] AS CompIndex, ISNULL(p.Code, '') AS ProductCode,
			cl.Quantity, cl.TopSeal, cl.DipSeal, cl.BottomSeal
		FROM GantryCompartmentalization comp
		JOIN GantryLoadingLine l ON l.ID = comp.IloID
		JOIN GantryCompartmentalizationLine cl ON cl.CompartmentalizationID = comp.ID
		LEFT JOIN Product p ON p.ID = cl.ProductID
		LEFT JOIN GantryLoading r ON r.CompartmentalizationID = comp.ID
		WHERE comp.Amended = 0 AND comp.Status IN (?, ?)
			AND ISNULL(r.LoadedAt, l.LoadedAt) >= ? AND ISNULL(r.LoadedAt, l.LoadedAt) <= ?
		ORDER BY ISNULL(r.LoadedAt, l.LoadedAt), comp.DocumentNumber, cl.[Index]`,
		types.OrderCompleted, types.OrderClosed, from, to).Scan(&rows).Error
	return serveRegister(c, "Loaded truck seals", "truck_seals", rows)
}
