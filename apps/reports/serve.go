package reports

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"dfms/pkg/export"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type reportMeta struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Group       string   `json:"group"`
	Description string   `json:"description"`
	Filters     []string `json:"filters"`
	Href        string   `json:"href,omitempty"`
}

func registry() []reportMeta {
	return []reportMeta{
		{Code: "dashboard", Name: "Operations snapshot", Group: "Stock", Description: "Open receipts, billing runs, licenses, and book stock.", Filters: []string{}},
		{Code: "stock-position", Name: "Stock position", Group: "Stock", Description: "Customer × product book stock, tank dips, line content, gain/loss, and ullage.", Filters: []string{}},
		{Code: "eom-stock", Name: "End of month stock", Group: "Stock", Description: "Customer balances as of a cutoff date, pivoted by product.", Filters: []string{"date"}},
		{Code: "balances-status", Name: "Balances with status", Group: "Stock", Description: "Local, transit, mining, and proration quantities.", Filters: []string{}},
		{Code: "customer-range", Name: "Customer range", Group: "Stock", Description: "In / out / net stock for a date range.", Filters: []string{"from", "to"}},
		{Code: "customer-statement", Name: "Customer stock statement", Group: "Stock", Description: "Opening, reception, loading, pump-over, ITT, adjustments, closing.", Filters: []string{"from", "to"}},
		{Code: "cargo-tracking", Name: "Cargo tracking", Group: "Stock", Description: "Open parcels by customer, vessel, and status.", Filters: []string{}},
		{Code: "vessel-parcel", Name: "Vessel / parcel movement", Group: "Stock", Description: "Balances by vessel, customer, product, and status (billing reference).", Filters: []string{}},
		{Code: "stock-movement", Name: "Stock movement subsidiary", Group: "Stock", Description: "Opening / movements / closing by customer and product for a month.", Filters: []string{"year", "month"}},
		{Code: "stock-aging", Name: "Stock aging", Group: "Stock", Description: "Open parcels ranked by days since vessel date.", Filters: []string{}},
		{Code: "daily-throughput", Name: "Daily throughput", Group: "Stock", Description: "Receipts, loading, pump-over, and ITT by day.", Filters: []string{"from", "to"}},
		{Code: "sbm-reception", Name: "SBM reception", Group: "Reception", Description: "Approved SBM vessel receipts.", Filters: []string{"from", "to"}},
		{Code: "koj-reception", Name: "KOJ reception", Group: "Reception", Description: "Approved KOJ vessel receipts.", Filters: []string{"from", "to"}},
		{Code: "sbm-monthly", Name: "SBM monthly reception", Group: "Reception", Description: "Depot × vessel crosstab for a calendar month.", Filters: []string{"year", "month"}},
		{Code: "koj-monthly", Name: "KOJ monthly reception", Group: "Reception", Description: "Depot × vessel crosstab for a calendar month.", Filters: []string{"year", "month"}},
		{Code: "market-share", Name: "Market share", Group: "Reception", Description: "Ships and volume by customer and route.", Filters: []string{"from", "to"}},
		{Code: "market-share-ships", Name: "Received ships market share", Group: "Reception", Description: "TIPER vs other terminals per ship.", Filters: []string{"from", "to", "route"}},
		{Code: "market-share-monthly", Name: "Market share per month", Group: "Reception", Description: "TIPER vs others and financial hold by month.", Filters: []string{"year", "route"}},
		{Code: "sbm-ships-share", Name: "SBM received ships market share", Group: "Reception", Description: "SBM vessels — TIPER vs other terminals per ship.", Filters: []string{"from", "to"}},
		{Code: "koj-ships-share", Name: "KOJ received ships market share", Group: "Reception", Description: "KOJ vessels — TIPER vs other terminals per ship.", Filters: []string{"from", "to"}},
		{Code: "sbm-monthly-share", Name: "SBM market share per month", Group: "Reception", Description: "SBM (Single Buoy Mooring) TIPER vs others and financial hold by month.", Filters: []string{"year"}},
		{Code: "koj-monthly-share", Name: "KOJ market share per month", Group: "Reception", Description: "KOJ (Kurasini Oil Jetty) TIPER vs others and financial hold by month.", Filters: []string{"year"}},
		{Code: "sbm-receipts", Name: "SBM receipts (TIPER vs others)", Group: "Reception", Description: "SBM quantities at TIPER vs other terminals per vessel — period and cumulative.", Filters: []string{"from", "to"}},
		{Code: "koj-receipts", Name: "KOJ receipts (TIPER vs others)", Group: "Reception", Description: "KOJ quantities at TIPER vs other terminals per vessel — period and cumulative.", Filters: []string{"from", "to"}},
		{Code: "srt-vessels", Name: "SRT vessel report", Group: "Reception", Description: "Vessels received under Single Receiving Terminal contracts, with status and financial hold.", Filters: []string{"from", "to"}},
		{Code: "periodic-stock-status", Name: "Periodic stock status", Group: "Stock", Description: "TIPER on-hand by transit / local / mining, receipts, financial hold, and SRT volume.", Filters: []string{"from", "to"}},
		{Code: "gain-loss", Name: "Gain / loss", Group: "Stock", Description: "Book stock versus physical dips for the period.", Filters: []string{"from", "to"}},
		{Code: "ewura-weekly", Name: "EWURA weekly stock", Group: "Stock", Description: "Customer code, name, stored quantity by status and financial hold.", Filters: []string{"to"}},
		{Code: "pbpa-weekly", Name: "PBPA weekly book stock", Group: "Stock", Description: "System book stock by product (AGO, PMS, HFO, DPK, …) and status.", Filters: []string{"to"}},
		{Code: "customer-stock", Name: "Individual customer stock", Group: "Stock", Description: "Balances, inflows, outflows, hold, and transit destinations (Congo, Malawi, Zambia, …).", Filters: []string{"from", "to", "customer"}},
		{Code: "daily-mass-balance", Name: "Daily book stock mass balance", Group: "Stock", Description: "Daily book versus dip by product with cumulative gain/loss.", Filters: []string{"from", "to"}},
		{Code: "monthly-confirmation", Name: "Monthly stock confirmation", Group: "Stock", Description: "Opening, receipts, adjustments, pump-overs, loading, ITT, and closing.", Filters: []string{"year", "month"}},
		{Code: "itt-summary", Name: "ITT summary", Group: "Stock", Description: "Inter-customer transfers: transferor, transferee, quantity, date, approval reference.", Filters: []string{"from", "to"}},
		{Code: "pump-over-status", Name: "Pump-over to terminals", Group: "Reception", Description: "TIPER to other terminals by destination depot and transit / local / mining.", Filters: []string{"from", "to"}},
		{Code: "hold-register", Name: "Financial hold register", Group: "Stock", Description: "Open parcels still on financial hold.", Filters: []string{}},
		{Code: "reservation-exposure", Name: "Open reservations", Group: "Stock", Description: "Soft holds taken when an order starts.", Filters: []string{}},
		{Code: "billing-list", Name: "Billing / invoice list", Group: "Finance", Description: "Billing runs for a period (use the date range for daily, monthly, or yearly).", Filters: []string{"from", "to"}},
		{Code: "billing-by-trade", Name: "Billing by class of trade", Group: "Finance", Description: "Amounts grouped by billing-profile class of trade and fee.", Filters: []string{"from", "to"}},
		{Code: "billing-by-product", Name: "Billing by product", Group: "Finance", Description: "Amounts grouped by product (AGO, PMS, HFO, DPK, …) and fee.", Filters: []string{"from", "to"}},
		{Code: "billing-exceptions", Name: "Billing exceptions", Group: "Finance", Description: "Waivers and exceptions still on file.", Filters: []string{}},
		{Code: "pump-over-loading", Name: "Pump-over & loading events", Group: "Stock", Description: "Posted gantry and pump-over events.", Filters: []string{"from", "to"}},
		{Code: "pump-over-omc", Name: "Pump-over & loading to OMCs", Group: "Reception", Description: "AGO/PMS transit and local by customer for a month.", Filters: []string{"year", "month"}},
		{Code: "billing-summary", Name: "Billing summary", Group: "Finance", Description: "Billing runs by fee, currency, and status.", Filters: []string{"from", "to"}},
		{Code: "tank-ullage", Name: "Tank ullage", Group: "Stock", Description: "Capacity, latest dip, and remaining ullage per tank.", Filters: []string{"active"}},
		{Code: "open-orders", Name: "Open orders", Group: "Gantry", Description: "ILR and pump-over documents still in progress.", Filters: []string{}},
		{Code: "license-expiry", Name: "EWURA license expiry", Group: "Finance", Description: "Petroleum licenses with days remaining.", Filters: []string{"active", "class"}},
		{Code: "workflow-aging", Name: "Workflow aging", Group: "Access", Description: "Open approval instances and days waiting.", Filters: []string{}},

		{Code: "customers", Name: "Customers list", Group: "Master", Description: "OMC customers storing product at TIPER.", Filters: []string{"active"}},
		{Code: "products", Name: "Products list", Group: "Master", Description: "AGO, PMS, and other stock items.", Filters: []string{"active", "category"}},
		{Code: "vessels", Name: "Vessels list", Group: "Master", Description: "Vessels that discharge at TIPER.", Filters: []string{"active"}},
		{Code: "drivers", Name: "Drivers list", Group: "Master", Description: "Gantry drivers.", Filters: []string{"active"}},
		{Code: "trucks", Name: "Trucks list", Group: "Master", Description: "Horse and trailer plates.", Filters: []string{"active"}},
		{Code: "transporters", Name: "Haulers list", Group: "Master", Description: "Haulers used at the gantry.", Filters: []string{"active"}},
		{Code: "depots", Name: "Depots list", Group: "Master", Description: "TIPER and pump-over destination depots.", Filters: []string{"active"}},
		{Code: "tanks", Name: "Tanks list", Group: "Master", Description: "Shore tanks and capacities.", Filters: []string{"active"}},
		{Code: "itts", Name: "In-tank transfers", Group: "Documents", Description: "Ownership transfers between customers.", Filters: []string{"from", "to"}},
		{Code: "pump-over-requests", Name: "Pump-over requests", Group: "Documents", Description: "Pipeline delivery orders (DR).", Filters: []string{"from", "to"}},
		{Code: "pump-over-reports", Name: "Pump-over reports", Group: "Documents", Description: "Executed pump-over delivery reports.", Filters: []string{"from", "to"}},

		{Code: "yearly-loading", Name: "Yearly loading summary", Group: "Gantry", Description: "AGO/PMS local and transit by month.", Filters: []string{"year"}},
		{Code: "monthly-loading", Name: "Monthly loading summary", Group: "Gantry", Description: "Loaded trucks for a calendar month.", Filters: []string{"year", "month"}},
		{Code: "daily-loading", Name: "Daily loaded trucks", Group: "Gantry", Description: "Trucks and volumes for one loading day.", Filters: []string{"date"}},
		{Code: "loading-plan", Name: "Loading plan", Group: "Gantry", Description: "Open gantry lines not yet loaded as of a date.", Filters: []string{"date"}},
		{Code: "transit-destination", Name: "Transit by destination", Group: "Gantry", Description: "Transit loadings by customer and destination.", Filters: []string{"year", "month"}},
		{Code: "ewura-loaded-trucks", Name: "EWURA loaded trucks", Group: "Gantry", Description: "Daily loaded trucks for the regulator.", Filters: []string{"date"}},
		{Code: "marked-fuel", Name: "Marked / local fuel", Group: "Gantry", Description: "Local (TBS-marked) loadings split 1–14 and 15–end.", Filters: []string{"year", "month"}},
		{Code: "glr-status", Name: "ILR loading status", Group: "Gantry", Description: "Internal loading-request status for a date range.", Filters: []string{"from", "to"}},
		{Code: "glr-approvals", Name: "ILR approvals", Group: "Gantry", Description: "Approved / completed internal loading requests.", Filters: []string{"from", "to"}},
		{Code: "truck-seals", Name: "Loaded truck seals", Group: "Gantry", Description: "Top, dip, and bottom seals on completed loads.", Filters: []string{"from", "to"}},

		{Code: "glr-document", Name: "ILR document", Group: "Documents", Description: "Print an internal loading request.", Filters: []string{"id"}},
		{Code: "delivery-note", Name: "Delivery note", Group: "Documents", Description: "Delivery note for a completed compartmentalization.", Filters: []string{"id"}},
		{Code: "gate-in", Name: "Gate-in pass", Group: "Documents", Description: "Gate-in pass for a running load.", Filters: []string{"id"}},
		{Code: "gate-out", Name: "Gate-out pass", Group: "Documents", Description: "Gate-out pass for a loaded truck.", Filters: []string{"id"}},
		{Code: "pump-over-document", Name: "Pump-over request", Group: "Documents", Description: "Pipeline delivery request (DR).", Filters: []string{"id"}},
		{Code: "pump-over-report", Name: "Pump-over report", Group: "Documents", Description: "Executed pump-over delivery report.", Filters: []string{"id"}},
		{Code: "itt-document", Name: "ITT document", Group: "Documents", Description: "In-tank transfer between customers.", Filters: []string{"id"}},
		{Code: "receipt-document", Name: "Vessel receipt", Group: "Documents", Description: "Print an internal or external vessel receipt.", Filters: []string{"id"}},
		{Code: "zerolization-document", Name: "Zerolization", Group: "Documents", Description: "Print a vessel consolidation.", Filters: []string{"id"}},
		{Code: "hold-release-document", Name: "Financial hold release", Group: "Documents", Description: "Print a financial-hold release.", Filters: []string{"id"}},
		{Code: "miloss-document", Name: "MI loss", Group: "Documents", Description: "Print an MI-loss rate batch.", Filters: []string{"id"}},

		{Code: "users", Name: "System users", Group: "Access", Description: "Access review listing.", Filters: []string{"from", "to"}, Href: "/reports/users"},
		{Code: "roles", Name: "Roles & permissions", Group: "Access", Description: "Permissions grouped by module.", Filters: []string{}, Href: "/reports/roles"},
		{Code: "audit-activity", Name: "Audit activity", Group: "Access", Description: "Who did what — filter by actor, action, or module.", Filters: []string{"from", "to"}, Href: "/reports/audit-activity"},
	}
}

func wantsExcel(c fiber.Ctx) bool {
	return c.Query("export") == "true" || strings.EqualFold(c.Query("format"), "xlsx")
}

func wantsPDF(c fiber.Ctx) bool {
	return strings.HasSuffix(strings.ToLower(c.Path()), ".pdf") || strings.EqualFold(c.Query("format"), "pdf")
}

func serveAny(c fiber.Ctx, title, file string, rows any) error {
	if headers, data, ok := tableFromSlice(rows); ok {
		if wantsPDF(c) || wantsExcel(c) {
			return serveTable(c, title, file, humanizeHeaders(headers), data)
		}
		return response.OkDetail(c, rows)
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return response.InternalServerError(c)
	}
	var maps []map[string]any
	if err := json.Unmarshal(raw, &maps); err != nil || maps == nil {
		if wantsExcel(c) || wantsPDF(c) {
			return serveMaps(c, title, file, nil, nil)
		}
		return response.OkDetail(c, rows)
	}
	keys := humanizeHeaders(columnKeys(maps))
	rawKeys := columnKeys(maps)
	if wantsPDF(c) {
		return export.TablePDF(c, title, file, keys, stringRows(maps, rawKeys))
	}
	if wantsExcel(c) {
		headers := make([]any, len(keys))
		data := make([][]any, 0, len(maps))
		for i, k := range keys {
			headers[i] = k
		}
		for _, m := range maps {
			row := make([]any, len(rawKeys))
			for i, k := range rawKeys {
				row[i] = formatCell(m[k])
			}
			data = append(data, row)
		}
		return export.Slice(c, title, file, headers, data)
	}
	return response.OkDetail(c, rows)
}

func serveMaps(c fiber.Ctx, title, file string, maps []map[string]any, keys []string) error {
	if keys == nil {
		keys = columnKeys(maps)
	}
	if wantsPDF(c) {
		return export.TablePDF(c, title, file, keys, stringRows(maps, keys))
	}
	if wantsExcel(c) {
		headers := make([]any, len(keys))
		data := make([][]any, 0, len(maps))
		for i, k := range keys {
			headers[i] = k
		}
		for _, m := range maps {
			row := make([]any, len(keys))
			for i, k := range keys {
				row[i] = m[k]
			}
			data = append(data, row)
		}
		return export.Slice(c, title, file, headers, data)
	}
	return response.OkDetail(c, maps)
}

func columnKeys(maps []map[string]any) []string {
	if len(maps) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var keys []string
	for _, m := range maps {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	return keys
}

func stringRows(maps []map[string]any, keys []string) [][]string {
	out := make([][]string, 0, len(maps))
	for _, m := range maps {
		row := make([]string, len(keys))
		for i, k := range keys {
			row[i] = formatCell(m[k])
		}
		out = append(out, row)
	}
	return out
}

func queryYear(c fiber.Ctx) int {
	if y, err := strconv.Atoi(c.Query("year")); err == nil && y >= 2000 && y <= 2100 {
		return y
	}
	return time.Now().Year()
}

func queryMonth(c fiber.Ctx) int {
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		return m
	}
	return int(time.Now().Month())
}

func queryDate(c fiber.Ctx) time.Time {
	if d := strings.TrimSpace(c.Query("date")); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			return t
		}
	}
	return time.Now()
}

func monthBounds(year, month int) (time.Time, time.Time) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return start, end
}

func dayBounds(d time.Time) (time.Time, time.Time) {
	start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
	return start, end
}
