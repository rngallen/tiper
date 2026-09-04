package reports

import (
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/export"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func serveTable(c fiber.Ctx, title, file string, headers []string, rows [][]string) error {
	if wantsPDF(c) {
		return export.TablePDF(c, title, file, headers, rows)
	}
	if wantsExcel(c) {
		heads := make([]any, len(headers))
		for i, h := range headers {
			heads[i] = h
		}
		data := make([][]any, 0, len(rows))
		for _, row := range rows {
			cells := make([]any, len(headers))
			for i := range headers {
				if i < len(row) {
					cells[i] = row[i]
				}
			}
			data = append(data, cells)
		}
		return export.Slice(c, title, file, heads, data)
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		m := fiber.Map{}
		for i, h := range headers {
			if i < len(row) {
				m[h] = row[i]
			}
		}
		out = append(out, m)
	}
	return response.OkDetail(c, out)
}

// serveRegister prints a tabular register: stable struct-field order, human
// headers, formatted cells, and a leading serial number.
func serveRegister(c fiber.Ctx, title, file string, rows any) error {
	headers, data, ok := tableFromSlice(rows)
	if !ok {
		return serveAny(c, title, file, rows)
	}
	return serveTable(c, title, file, append([]string{"S/N"}, humanizeHeaders(headers)...), withSerial(data))
}

func withSerial(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = append([]string{strconv.Itoa(i + 1)}, row...)
	}
	return out
}

func queryActive(c fiber.Ctx) string {
	v := strings.ToLower(strings.TrimSpace(c.Query("active")))
	switch v {
	case "active", "true", "1", "yes":
		return "active"
	case "inactive", "false", "0", "no":
		return "inactive"
	default:
		return "all"
	}
}

func applyActive(q *gorm.DB, c fiber.Ctx) *gorm.DB {
	switch queryActive(c) {
	case "active":
		return q.Where("IsActive = ?", true)
	case "inactive":
		return q.Where("IsActive = ?", false)
	default:
		return q
	}
}

func activeClause(col, flag string) (string, []any) {
	switch flag {
	case "active":
		return " AND " + col + " = ?", []any{true}
	case "inactive":
		return " AND " + col + " = ?", []any{false}
	default:
		return "", nil
	}
}

func queryClasses(c fiber.Ctx) []string {
	return parseClassList(c.Query("class"))
}

func parseClassList(raw string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func queryCategory(c fiber.Ctx) string {
	return strings.TrimSpace(c.Query("category"))
}

func applyProductCategory(q *gorm.DB, c fiber.Ctx) *gorm.DB {
	uid := queryCategory(c)
	if uid == "" {
		return q
	}
	return q.Where("EXISTS (SELECT 1 FROM StockCategory sc WHERE sc.ID = p.StockCategoryID AND sc.UID = ?)", uid)
}

func categoryTitle(uid string) string {
	if uid == "" {
		return ""
	}
	var name string
	_ = db.Db.Model(&models.StockCategory{}).Where("UID = ?", uid).Limit(1).Pluck("Name", &name).Error
	return strings.TrimSpace(name)
}

func titled(base string, notes ...string) string {
	var parts []string
	for _, n := range notes {
		if s := strings.TrimSpace(n); s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return base
	}
	return base + " — " + strings.Join(parts, " · ")
}

func activeNote(c fiber.Ctx) string {
	switch queryActive(c) {
	case "active":
		return "Active"
	case "inactive":
		return "Inactive"
	default:
		return ""
	}
}

func fmtProduct(code, name string) string {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	switch {
	case code != "" && name != "" && !strings.EqualFold(code, name):
		return code + " — " + name
	case name != "":
		return name
	default:
		return code
	}
}

func fmtDateOnly(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("02/01/2006")
}

func fmtTimeCell(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Format("02/01/2006")
	}
	return t.Format("02/01/2006 15:04")
}

func titleWord(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

func tableFromSlice(rows any) ([]string, [][]string, bool) {
	v := reflect.ValueOf(rows)
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return nil, nil, false
	}
	et := v.Type().Elem()
	if et.Kind() == reflect.Pointer {
		et = et.Elem()
	}
	if et.Kind() != reflect.Struct {
		return nil, nil, false
	}
	var fields []reflect.StructField
	for i := 0; i < et.NumField(); i++ {
		f := et.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := jsonFieldName(f)
		if name == "" {
			continue
		}
		fields = append(fields, f)
	}
	if len(fields) == 0 {
		return nil, nil, false
	}
	headers := make([]string, len(fields))
	for i, f := range fields {
		headers[i] = jsonFieldName(f)
	}
	data := make([][]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		ev := v.Index(i)
		if ev.Kind() == reflect.Pointer {
			if ev.IsNil() {
				data = append(data, make([]string, len(fields)))
				continue
			}
			ev = ev.Elem()
		}
		row := make([]string, len(fields))
		for j, f := range fields {
			row[j] = formatCell(ev.FieldByIndex(f.Index).Interface())
		}
		data = append(data, row)
	}
	return headers, data, true
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if tag != "" {
		name := strings.Split(tag, ",")[0]
		if name != "" {
			return name
		}
	}
	return f.Name
}

var headerAlias = map[string]string{
	"isActive":         "Active",
	"IsActive":         "Active",
	"active":           "Active",
	"isInternal":       "Internal",
	"IsInternal":       "Internal",
	"ewuraLicense":     "EWURA license",
	"EwuraLicense":     "EWURA license",
	"tinNumber":        "TIN",
	"TinNumber":        "TIN",
	"imoNumber":        "IMO number",
	"ImoNumber":        "IMO number",
	"productCode":      "Product",
	"ProductCode":      "Product",
	"licenseNumber":    "License number",
	"LicenseNumber":    "License number",
	"licenseExpires":   "Expiry",
	"LicenseExpires":   "Expiry",
	"expiryDate":       "Expiry",
	"ExpiryDate":       "Expiry",
	"maximumCapacity":  "Maximum capacity (L)",
	"MaximumCapacity":  "Maximum capacity (L)",
	"deadStock":        "Dead stock (L)",
	"DeadStock":        "Dead stock (L)",
	"Capacity":         "Capacity (L)",
	"Ullage":           "Ullage (L)",
	"Dip":              "Dip (L)",
	"documentNumber":   "Document",
	"DocumentNumber":   "Document",
	"customerName":     "Customer",
	"CustomerName":     "Customer",
	"vesselCode":       "Vessel",
	"VesselCode":       "Vessel",
	"vesselDate":       "Vessel date",
	"VesselDate":       "Vessel date",
	"orderDate":        "Order date",
	"transferDate":     "Transfer date",
	"createdAt":        "Created",
	"CreatedAt":        "Created",
	"daysLeft":         "Days left",
	"DaysLeft":         "Days left",
	"daysOpen":         "Days open",
	"DaysOpen":         "Days open",
	"daysOnHand":       "Days on hand",
	"DaysOnHand":       "Days on hand",
	"licenseNo":        "License number",
	"LicenseNo":        "License number",
	"feeCode":          "Fee",
	"FeeCode":          "Fee",
	"currencyCode":     "Currency",
	"CurrencyCode":     "Currency",
	"periodStart":      "Period",
	"PeriodStart":      "Period",
	"fromCustomer":     "From customer",
	"toCustomer":       "To customer",
	"depotName":        "Depot",
	"requestNumber":    "Request",
	"actualDelivered":  "Delivered (L)",
	"actualReceived":   "Received (L)",
	"variance":         "Variance (L)",
	"quantity":         "Quantity (L)",
	"Quantity":         "Quantity (L)",
	"amount":           "Amount",
	"Amount":           "Amount",
	"processName":      "Process",
	"documentNo":       "Document",
	"nodeName":         "Step",
	"orderNumber":      "Order",
	"expiresAt":        "Expires",
	"validUntil":       "Valid until",
	"licensee":         "Licensee",
	"class":            "Class",
	"Class":            "Class",
}

func humanizeHeaders(keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = humanizeHeader(k)
	}
	return out
}

func humanizeHeader(name string) string {
	if a, ok := headerAlias[name]; ok {
		return a
	}
	if name == "" {
		return name
	}
	var b strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		if i == 0 {
			b.WriteRune(unicode.ToUpper(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatCell(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return yesNo(t)
	case time.Time:
		return fmtTimeCell(t)
	case *time.Time:
		return fmtDateOnly(t)
	case decimal.Decimal:
		return fmtL(t)
	case *decimal.Decimal:
		if t == nil {
			return ""
		}
		return fmtL(*t)
	case int:
		return strconv.Itoa(t)
	case int8:
		return strconv.FormatInt(int64(t), 10)
	case int16:
		return strconv.FormatInt(int64(t), 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint16:
		return strconv.FormatUint(uint64(t), 10)
	case uint32:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case float32:
		return fmtL(decimal.NewFromFloat32(t))
	case float64:
		return fmtL(decimal.NewFromFloat(t))
	case []byte:
		return formatBytes(t)
	default:
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return ""
		}
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return ""
			}
			return formatCell(rv.Elem().Interface())
		}
		if rv.Kind() == reflect.String {
			return rv.String()
		}
		if rv.Kind() == reflect.Bool {
			return yesNo(rv.Bool())
		}
		return strings.TrimSpace(strings.Trim(rv.String(), `"`))
	}
}

func parseQty(s string) decimal.Decimal {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func fmtQtyStr(s string) string {
	return fmtL(parseQty(s))
}

func formatBytes(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	if d, err := decimal.NewFromString(s); err == nil {
		return fmtL(d)
	}
	return s
}

func filterOptions(c fiber.Ctx) error {
	type cat struct {
		UID  string `json:"id"`
		Name string `json:"name"`
	}
	var categories []cat
	_ = db.Db.Model(&models.StockCategory{}).
		Select("UID, Name").
		Where("IsActive = ?", true).
		Order("Name").
		Scan(&categories).Error
	var classes []string
	_ = db.Db.Model(&models.EwuraPetroleumLicense{}).
		Distinct("LicenseClass").
		Where("LicenseClass <> ''").
		Order("LicenseClass").
		Pluck("LicenseClass", &classes).Error
	if categories == nil {
		categories = []cat{}
	}
	if classes == nil {
		classes = []string{}
	}
	return response.OkDetail(c, fiber.Map{
		"categories": categories,
		"classes":    classes,
	})
}