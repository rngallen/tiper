package reports

import (
	"time"

	"dfms/apps/models"
	"dfms/pkg/db"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
)

func customersList(c fiber.Ctx) error {
	type row struct {
		Code         string
		Name         string
		KycNumber    string
		VrnNumber    string
		EwuraLicense string
		TinNumber    string
		Email        string
		Phone        string
		IsActive     bool
	}
	var rows []row
	_ = applyActive(db.Db.Model(&models.Customer{}), c).
		Select("Code, Name, KycNumber, VrnNumber, EwuraLicense, TinNumber, Email, Phone, IsActive").
		Order("Code, Name").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{
			r.Code, r.Name, r.KycNumber, r.VrnNumber, r.EwuraLicense, r.TinNumber, r.Email, r.Phone, yesNo(r.IsActive),
		})
	}
	return serveTable(c, titled("Customers", activeNote(c)), "customers",
		[]string{"S/N", "Code", "Name", "KYC", "VRN", "EWURA license", "TIN", "Email", "Phone", "Active"},
		withSerial(out))
}

func productsList(c fiber.Ctx) error {
	type row struct {
		Code     string
		Name     string
		Unit     string
		Category string
		IsActive bool
	}
	q := applyProductCategory(db.Db.Table("Product p").
		Joins("JOIN StockCategory c ON c.ID = p.StockCategoryID"), c)
	switch queryActive(c) {
	case "active":
		q = q.Where("p.IsActive = ?", true)
	case "inactive":
		q = q.Where("p.IsActive = ?", false)
	}
	var rows []row
	_ = q.Select("p.Code, p.Name, p.Unit, c.Name AS Category, p.IsActive").
		Order("p.Code").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Code, r.Name, r.Unit, r.Category, yesNo(r.IsActive)})
	}
	return serveTable(c, titled("Products", activeNote(c), categoryTitle(queryCategory(c))), "products",
		[]string{"S/N", "Code", "Name", "Unit", "Category", "Active"},
		withSerial(out))
}

func vesselsList(c fiber.Ctx) error {
	type row struct {
		Code      string
		Name      string
		ImoNumber string
		IsActive  bool
	}
	var rows []row
	_ = applyActive(db.Db.Model(&models.Vessel{}), c).
		Select("Code, Name, ImoNumber, IsActive").
		Order("Code, Name").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Code, r.Name, r.ImoNumber, yesNo(r.IsActive)})
	}
	return serveTable(c, titled("Vessels", activeNote(c)), "vessels",
		[]string{"S/N", "Code", "Name", "IMO number", "Active"},
		withSerial(out))
}

func driversList(c fiber.Ctx) error {
	type row struct {
		LicenseNumber  string
		Name           string
		LicenseExpires *time.Time
		Phone          string
		IsActive       bool
	}
	var rows []row
	_ = applyActive(db.Db.Model(&models.Driver{}), c).
		Select("LicenseNumber, Name, LicenseExpires, Phone, IsActive").
		Order("Name").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{
			r.LicenseNumber, r.Name, fmtDateOnly(r.LicenseExpires), r.Phone, yesNo(r.IsActive),
		})
	}
	return serveTable(c, titled("Drivers", activeNote(c)), "drivers",
		[]string{"S/N", "Licence", "Name", "Expiry", "Phone", "Active"},
		withSerial(out))
}

func trucksList(c fiber.Ctx) error {
	type row struct {
		PlateNumber string
		Trailer     string
		TrailerTwo  string
		VehicleType string
		IsActive    bool
	}
	var rows []row
	_ = applyActive(db.Db.Model(&models.Truck{}), c).
		Select("PlateNumber, Trailer, TrailerTwo, VehicleType, IsActive").
		Order("PlateNumber").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{
			r.PlateNumber, r.Trailer, r.TrailerTwo, titleWord(r.VehicleType), yesNo(r.IsActive),
		})
	}
	return serveTable(c, titled("Trucks", activeNote(c)), "trucks",
		[]string{"S/N", "Horse plate", "Trailer", "Trailer two", "Vehicle type", "Active"},
		withSerial(out))
}

func transportersList(c fiber.Ctx) error {
	type row struct {
		Name      string
		TinNumber string
		Phone     string
		License   string
		IsActive  bool
	}
	var rows []row
	_ = applyActive(db.Db.Model(&models.Transporter{}), c).
		Select("Name, TinNumber, Phone, License, IsActive").
		Order("Name").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Name, r.TinNumber, r.Phone, r.License, yesNo(r.IsActive)})
	}
	return serveTable(c, titled("Haulers", activeNote(c)), "transporters",
		[]string{"S/N", "Name", "TIN", "Phone", "License", "Active"},
		withSerial(out))
}

func depotsList(c fiber.Ctx) error {
	type row struct {
		Code         string
		Name         string
		EwuraLicense string
		IsInternal   bool
		IsActive     bool
	}
	var rows []row
	_ = applyActive(db.Db.Model(&models.Depot{}), c).
		Select("Code, Name, EwuraLicense, IsInternal, IsActive").
		Order("Code, Name").Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{r.Code, r.Name, r.EwuraLicense, yesNo(r.IsInternal), yesNo(r.IsActive)})
	}
	return serveTable(c, titled("Depots", activeNote(c)), "depots",
		[]string{"S/N", "Code", "Name", "EWURA license", "Internal", "Active"},
		withSerial(out))
}

func tanksList(c fiber.Ctx) error {
	type row struct {
		Code            string
		Name            string
		ProductCode     string
		ProductName     string
		MaximumCapacity string
		DeadStock       string
		IsActive        bool
	}
	clause, args := activeClause("t.IsActive", queryActive(c))
	var rows []row
	_ = db.Db.Raw(`
		SELECT t.Code, t.Name,
			ISNULL(p.Code, '') AS ProductCode, ISNULL(p.Name, '') AS ProductName,
			CONVERT(varchar(48), t.MaximumCapacity) AS MaximumCapacity,
			CONVERT(varchar(48), t.DeadStock) AS DeadStock, t.IsActive
		FROM Tank t
		LEFT JOIN Product p ON p.ID = t.ProductID
		WHERE 1 = 1`+clause+`
		ORDER BY t.Code`, args...).Scan(&rows).Error
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, []string{
			r.Code, r.Name, fmtProduct(r.ProductCode, r.ProductName),
			fmtQtyStr(r.MaximumCapacity), fmtQtyStr(r.DeadStock), yesNo(r.IsActive),
		})
	}
	return serveTable(c, titled("Tanks", activeNote(c)), "tanks",
		[]string{"S/N", "Code", "Name", "Product", "Maximum capacity (L)", "Dead stock (L)", "Active"},
		withSerial(out))
}

func ittsList(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		DocumentNumber string
		TransferDate   time.Time
		FromCustomer   string
		ToCustomer     string
		ProductCode    string
		VesselCode     string
		Quantity       decimal.Decimal `gorm:"type:decimal"`
		Status         string
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT t.DocumentNumber, t.TransferDate,
			fc.Name AS FromCustomer, tc.Name AS ToCustomer,
			p.Code AS ProductCode, v.Code AS VesselCode,
			t.Quantity, t.Status
		FROM IttTransfer t
		JOIN Customer fc ON fc.ID = t.FromCustomerID
		JOIN Customer tc ON tc.ID = t.ToCustomerID
		JOIN Product p ON p.ID = t.ProductID
		JOIN Vessel v ON v.ID = t.VesselID
		WHERE t.TransferDate >= ? AND t.TransferDate <= ?
		ORDER BY t.TransferDate DESC, t.DocumentNumber`, from, to).Scan(&rows).Error
	return serveRegister(c, "In-tank transfers", "itts", rows)
}

func pumpOverRequestsList(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		DocumentNumber string
		OrderDate      time.Time
		CustomerName   string
		ProductCode    string
		DepotName      string
		Quantity       decimal.Decimal `gorm:"type:decimal"`
		Status         string
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT r.DocumentNumber, r.OrderDate, c.Name AS CustomerName,
			p.Code AS ProductCode, d.Name AS DepotName, r.Quantity, r.Status
		FROM PumpOverRequest r
		JOIN Customer c ON c.ID = r.CustomerID
		JOIN Product p ON p.ID = r.ProductID
		JOIN Depot d ON d.ID = r.DepotID
		WHERE r.OrderDate >= ? AND r.OrderDate <= ?
		ORDER BY r.OrderDate DESC, r.DocumentNumber`, from, to).Scan(&rows).Error
	return serveRegister(c, "Pump-over requests", "pump_over_requests", rows)
}

func pumpOverReportsList(c fiber.Ctx) error {
	from, to := parseRange(c)
	type row struct {
		DocumentNumber  string
		ReportDate      time.Time
		RequestNumber   string
		CustomerName    string
		ActualDelivered decimal.Decimal `gorm:"type:decimal"`
		ActualReceived  decimal.Decimal `gorm:"type:decimal"`
		Variance        decimal.Decimal `gorm:"type:decimal"`
		Status          string
	}
	var rows []row
	_ = db.Db.Raw(`
		SELECT r.DocumentNumber, r.ReportDate, req.DocumentNumber AS RequestNumber,
			c.Name AS CustomerName, r.ActualDelivered, r.ActualReceived, r.Variance, r.Status
		FROM PumpOverReport r
		JOIN PumpOverRequest req ON req.ID = r.RequestID
		JOIN Customer c ON c.ID = req.CustomerID
		WHERE r.ReportDate >= ? AND r.ReportDate <= ?
		ORDER BY r.ReportDate DESC, r.DocumentNumber`, from, to).Scan(&rows).Error
	return serveRegister(c, "Pump-over reports", "pump_over_reports", rows)
}
