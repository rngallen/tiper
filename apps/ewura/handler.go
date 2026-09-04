package ewura

import (
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/export"
	"dfms/pkg/logs"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func listLicenses(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequestBind(c, err)
	}

	allowed := map[string]string{
		"licenseNumber": "LicenseNumber",
		"licensee":      "Licensee",
		"licenseClass":  "LicenseClass",
		"licenseType":   "LicenseType",
		"regionName":    "RegionName",
		"districtName":  "DistrictName",
		"tinNumber":     "TinNumber",
		"expiryDate":    "ExpiryDate",
		"issueDate":     "IssueDate",
		"isActive":      "IsActive",
	}
	sortCol, ok := allowed[search.OrderBy]
	if !ok {
		sortCol = "Licensee"
	}

	q := applyLicenseFilters(db.Db.WithContext(c.Context()).Model(&models.EwuraPetroleumLicense{}), search, c)
	q = q.Order(licenseOrder(sortCol, search.SortDirection))

	headers := []any{"License", "Licensee", "Class", "Type", "Sector", "Zone", "Region", "District", "TIN", "Phone", "Email", "Issued", "Expires", "Active"}
	mapRow := func(r models.EwuraPetroleumLicense) []any {
		return []any{
			r.LicenseNumber, r.Licensee, r.LicenseClass, r.LicenseType, r.Sector,
			r.ZoneName, r.RegionName, r.DistrictName, r.TinNumber, r.Phone, r.Email,
			fmtDate(r.IssueDate), fmtDate(r.ExpiryDate), r.IsActive,
		}
	}

	if search.Export {
		return export.Query(c, q, "EWURA licenses", "ewura_licenses", headers, mapRow)
	}
	if wantsPDF(c) {
		var rows []models.EwuraPetroleumLicense
		if err := q.Find(&rows).Error; err != nil {
			logs.Error(err)
			return response.InternalServerError(c)
		}
		out := make([][]string, 0, len(rows))
		for _, r := range rows {
			active := "No"
			if r.IsActive {
				active = "Yes"
			}
			out = append(out, []string{
				r.LicenseNumber, r.Licensee, r.LicenseClass, r.LicenseType,
				r.RegionName, r.TinNumber, fmtDate(r.ExpiryDate), active,
			})
		}
		return export.TablePDF(c, "EWURA petroleum licenses", "ewura_licenses",
			[]string{"License", "Licensee", "Class", "Type", "Region", "TIN", "Expires", "Active"}, out)
	}

	var items []models.EwuraPetroleumLicense
	page, err := response.NewPaginator(c, q, search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, page)
}

func licenseOptions(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	query := db.Db.WithContext(c.Context()).Model(&models.EwuraPetroleumLicense{}).
		Where("IsActive = ?", true).
		Order("Licensee ASC, LicenseNumber ASC")
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"LicenseNumber LIKE ? OR Licensee LIKE ? OR TinNumber LIKE ?",
			like, like, like,
		)
	}
	var rows []models.EwuraPetroleumLicense
	if err := query.Limit(80).Find(&rows).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, rows)
}

func applyLicenseFilters(q *gorm.DB, search response.SearchOutput, c fiber.Ctx) *gorm.DB {
	if search.HasSearch() {
		q = q.Where(
			"LicenseNumber LIKE ? OR Licensee LIKE ? OR TinNumber LIKE ? OR LicenseClass LIKE ? OR RegionName LIKE ? OR DistrictName LIKE ? OR Email LIKE ?",
			search.Search, search.Search, search.Search, search.Search, search.Search, search.Search, search.Search,
		)
	}
	if search.IsActive != nil {
		q = q.Where("IsActive = ?", *search.IsActive)
	}
	if cls := strings.TrimSpace(c.Query("licenseClass")); cls != "" {
		q = q.Where("LicenseClass = ?", cls)
	}
	return q
}

func licenseOrder(col, dir string) string {
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	col = strings.TrimSpace(col)
	if col == "" {
		col = "Licensee"
	}
	if strings.EqualFold(col, "LicenseNumber") {
		return "LicenseNumber " + dir
	}
	return col + " " + dir + ", LicenseNumber ASC"
}

func wantsPDF(c fiber.Ctx) bool {
	return strings.HasSuffix(strings.ToLower(c.Path()), ".pdf") ||
		strings.EqualFold(c.Query("format"), "pdf")
}

func fmtDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("02/01/2006")
}
