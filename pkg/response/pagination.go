// Package response: pagination & search for Fiber + GORM handlers.
//
// Features:
//   - Safe pagination with sane defaults and a maximum page size
//   - Date range filtering (reports: last 2 years; operational lists: last 90 days)
//   - Free-text search and boolean (isActive) filters
//   - Stable, configurable sort direction
//   - Per-instance caching of total counts to avoid redundant COUNT(*) queries
package response

import (
	"dfms/utils"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
	"gorm.io/gorm"
)

// Pagination configuration constants.
const (
	// DefaultPageNumber is the lowest valid page number (always 1-based).
	DefaultPageNumber = 1

	// DefaultPageSize is the default number of items per page.
	DefaultPageSize = 10

	// MaxPageSize caps the page size to prevent abuse.
	MaxPageSize = 100

	// MinDateRangeYears is the default fromDate lookback for reports.
	MinDateRangeYears = 2

	// DefaultOpsLookbackDays is the default fromDate for operational lists
	// (operational lists, live audit). Explicit fromDate/toDate still win.
	DefaultOpsLookbackDays = 90

	// DateLayout is the accepted format for date query parameters.
	DateLayout = "02/01/2006"

	SortAsc  = "ASC"
	SortDesc = "DESC"
)

// allowedSortDirections is the set of legal values for the sortDirection
// query parameter (empty string is allowed and treated as "DESC").
var allowedSortDirections = []string{"", "asc", "desc", "ASC", "DESC"}

// PaginationResponse is the JSON shape returned by paginated endpoints.
type PaginationResponse struct {
	Items        any   `json:"items,omitempty"`        // Current page items (slice/array)
	PreviousPage *int  `json:"previousPage,omitempty"` // Previous page number (nil if first)
	Page         int   `json:"page"`                   // Current page number
	NextPage     *int  `json:"nextPage,omitempty"`     // Next page number (nil if last)
	PageSize     int   `json:"pageSize"`               // Items per page
	TotalPages   int   `json:"totalPages"`             // Total number of pages (rounded up)
	ItemsCount   int64 `json:"itemsCount"`             // Total items matching query
}

// searchParams maps query string parameters for binding.
type searchParams struct {
	Page          int    `query:"page"`
	PageSize      int    `query:"pageSize"`
	OrderBy       string `query:"orderBy"`
	SortDirection string `query:"sortDirection"`
	Search        string `query:"search"`
	IsActive      *bool  `query:"isActive"`
	Export        bool   `query:"export"`       // Export data as Excel
	FromDate      string `query:"fromDate"`     // Format: 02/01/2006
	ToDate        string `query:"toDate"`       // Format: 02/01/2006
	CurrencyCode  string `query:"currencyCode"` // ISO 4217, optional
}

// SearchOutput holds normalized and validated search parameters ready for use.
type SearchOutput struct {
	Page          int       `json:"page"`          // Current page number
	PageSize      int       `json:"pageSize"`      // Items per page
	Search        string    `json:"search"`        // LIKE pattern (with % wildcards)
	OrderBy       string    `json:"orderBy"`       // Field to order by
	SortDirection string    `json:"sortDirection"` // "ASC" or "DESC"
	IsActive      *bool     `json:"isActive"`      // Active filter
	Export        bool      `json:"export"`        // Export data as Excel
	FromDate      time.Time `json:"fromDate"`      // Start date (DD/MM/YYYY)
	ToDate        time.Time `json:"toDate"`        // End date (DD/MM/YYYY)
	CurrencyCode  string    `json:"currencyCode"`  // Upper-cased ISO filter; empty = all
}

// HasSearch is true when the client sent a non-empty search term.
// ParseSearchRequest stores an empty term as "%%", which must not be used in
// LIKE predicates — leading-and-trailing wildcards on every row force a scan.
func (s SearchOutput) HasSearch() bool {
	t := strings.TrimSpace(s.Search)
	return t != "" && t != "%%"
}

// Paginator encapsulates everything needed to run a single paginated query.
type Paginator struct {
	Context fiber.Ctx    // Fiber request context
	Query   *gorm.DB     // Base GORM query (before pagination)
	Search  SearchOutput // Parsed and validated search/filter parameters
	Target  any          // Pointer to slice that receives results (e.g. &[]Model)

	// countCache memoizes the COUNT(*) result so repeated calls in the same
	// request lifecycle don't re-issue the query.
	countCache struct {
		sync.RWMutex
		valid     bool      // True once count has been populated.
		count     int64     // Cached total item count.
		timestamp time.Time // When cache was last updated.
	}
}

// NewPaginator constructs a Paginator. Target must be a pointer to a slice
// of the model type, e.g. &[]models.Tank{}.
func NewPaginator(c fiber.Ctx, q *gorm.DB, search SearchOutput, target any) *Paginator {
	return &Paginator{
		Context: c,
		Query:   q,
		Search:  search,
		Target:  target,
	}
}

// ParseSearchRequest parses, validates, and normalizes search/pagination
// parameters from the request:
//
//   - Pagination (page, pageSize) — defaulted and capped at MaxPageSize.
//   - Sorting (orderBy, sortDirection) — direction normalized to ASC/DESC.
//   - Free-text search — wrapped with % for LIKE/ILIKE queries.
//   - Boolean filter (isActive).
//   - Currency filter (currencyCode) — upper-cased ISO; empty = all.
//   - Date range — defaults to (now - MinDateRangeYears, now) unless a lookback
//     in days is passed via ParseOpsSearchRequest.
//
// Expected date format: "02/01/2006" (DD/MM/YYYY).
func ParseSearchRequest(c fiber.Ctx) (SearchOutput, error) {
	return parseSearchRequest(c, 0)
}

// ParseOpsSearchRequest is ParseSearchRequest with a 90-day default fromDate.
func ParseOpsSearchRequest(c fiber.Ctx) (SearchOutput, error) {
	return parseSearchRequest(c, DefaultOpsLookbackDays)
}

// QueryAllDates is true when the client asked to skip the default date window
// (operational lists "Clear dates").
func QueryAllDates(c fiber.Ctx) bool {
	v := strings.TrimSpace(c.Query("allDates"))
	return v == "1" || strings.EqualFold(v, "true")
}

func parseSearchRequest(c fiber.Ctx, lookbackDays int) (SearchOutput, error) {
	var output SearchOutput
	var params searchParams

	if err := c.Bind().Query(&params); err != nil {
		return output, fmt.Errorf("failed to parse query parameters: %w", err)
	}

	// Validate the values that have a fixed shape (dates and sort direction).
	err := validation.Errors{
		"fromDate": validation.Validate(params.FromDate,
			validation.When(params.FromDate != "", validation.Date(DateLayout))),
		"toDate": validation.Validate(params.ToDate,
			validation.When(params.ToDate != "", validation.Date(DateLayout))),
		"sortDirection": validation.Validate(params.SortDirection,
			validation.In(toAnySlice(allowedSortDirections)...)),
	}.Filter()
	if err != nil {
		return output, fmt.Errorf("invalid query parameter: %w", err)
	}

	// Parse and finalize the date range. Default fromDate is lookback before
	// the range *end* (toDate, or now). As-of reports send only toDate (e.g.
	// 08/05/2008); anchoring fromDate to now would make from > to.
	now := time.Now()
	toDateTime := now
	if params.ToDate != "" {
		t, err := utils.ParseDate(params.ToDate)
		if err != nil {
			return output, fmt.Errorf("invalid toDate format: %w", err)
		}
		// Inclusive end-of-day so a single calendar day returns all events.
		toDateTime = endOfDay(t)
	}
	fromDateTime := defaultFromForTo(toDateTime, lookbackDays)
	if params.FromDate != "" {
		t, err := utils.ParseDate(params.FromDate)
		if err != nil {
			return output, fmt.Errorf("invalid fromDate format: %w", err)
		}
		fromDateTime = t
	}
	if fromDateTime.After(toDateTime) {
		return output, errors.New("fromDate cannot be after toDate")
	}

	// Normalize pagination.
	page := max(params.Page, DefaultPageNumber)
	pageSize := params.PageSize
	switch {
	case pageSize < 1:
		pageSize = DefaultPageSize
	case pageSize > MaxPageSize:
		pageSize = MaxPageSize
	}

	// Normalize sort direction (default DESC).
	sortDirection := strings.ToUpper(strings.TrimSpace(params.SortDirection))
	if sortDirection != SortAsc && sortDirection != SortDesc {
		sortDirection = SortDesc
	}

	if len(strings.TrimSpace(params.Search)) > 80 {
		return output, errors.New("search is too long")
	}
	if len(strings.TrimSpace(params.OrderBy)) > 60 {
		return output, errors.New("orderBy is too long")
	}
	if cc := strings.TrimSpace(params.CurrencyCode); len(cc) > 3 {
		return output, errors.New("currencyCode is too long")
	}

	// Wrap the search term for ILIKE/LIKE.
	searchTerm := "%" + strings.TrimSpace(params.Search) + "%"

	output = SearchOutput{
		Page:          page,
		PageSize:      pageSize,
		Search:        searchTerm,
		OrderBy:       strings.TrimSpace(params.OrderBy),
		SortDirection: sortDirection,
		IsActive:      params.IsActive,
		Export:        params.Export,
		FromDate:      fromDateTime,
		ToDate:        toDateTime,
		CurrencyCode:  strings.ToUpper(strings.TrimSpace(params.CurrencyCode)),
	}
	return output, nil
}

// defaultFromForTo is the implicit fromDate when the client omitted it:
// lookbackDays before `to`, or MinDateRangeYears when lookbackDays is 0.
func defaultFromForTo(to time.Time, lookbackDays int) time.Time {
	if lookbackDays > 0 {
		return to.AddDate(0, 0, -lookbackDays)
	}
	return to.AddDate(-MinDateRangeYears, 0, 0)
}

// ParseAsOf is a cutoff date (toDate query, DD/MM/YYYY). Empty means end of today.
// As-of reports (aging, stock position) use TransactionDate <= asOf from the
// start of the ledger — there is no fromDate window.
func ParseAsOf(raw string, now time.Time) (time.Time, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if strings.TrimSpace(raw) == "" {
		return endOfDay(now), nil
	}
	t, err := utils.ParseDate(raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid toDate format: %w", err)
	}
	return endOfDay(t), nil
}

// ParseAsOfRequest reads optional toDate for as-of reports.
func ParseAsOfRequest(c fiber.Ctx) (time.Time, error) {
	return ParseAsOf(c.Query("toDate"), time.Now())
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// applyPaginationScope returns a GORM scope that adds OFFSET and LIMIT.
func (p *Paginator) applyPaginationScope() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (p.Search.Page - 1) * p.Search.PageSize
		return db.Offset(offset).Limit(p.Search.PageSize)
	}
}

// totalItems returns the total count and computed page count, using the
// instance cache when it's still fresh (within cacheTTL).
func (p *Paginator) totalItems(cacheTTL time.Duration) (count int64, pages int, err error) {
	p.countCache.RLock()
	if p.countCache.valid && time.Since(p.countCache.timestamp) < cacheTTL {
		count = p.countCache.count
		p.countCache.RUnlock()
		return count, pageCount(count, p.Search.PageSize), nil
	}
	p.countCache.RUnlock()

	p.countCache.Lock()
	defer p.countCache.Unlock()

	// Re-check after acquiring the write lock in case another goroutine
	// populated the cache while we were upgrading.
	if p.countCache.valid && time.Since(p.countCache.timestamp) < cacheTTL {
		return p.countCache.count, pageCount(p.countCache.count, p.Search.PageSize), nil
	}

	// Session clone so COUNT cannot mutate ORDER BY / SELECT on the list query.
	if err = p.Query.Session(&gorm.Session{}).Count(&count).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to count records: %w", err)
	}
	p.countCache.count = count
	p.countCache.timestamp = time.Now()
	p.countCache.valid = true

	return count, pageCount(count, p.Search.PageSize), nil
}

// pageCount returns the total number of pages for the given item count
// and page size, rounded up.
func pageCount(items int64, pageSize int) int {
	if pageSize <= 0 || items <= 0 {
		return 0
	}
	return int(math.Ceil(float64(items) / float64(pageSize)))
}

// Run executes the paginated query and builds the response payload.
//
//	var users []User
//	p := response.NewPaginator(c, db.Where("status = ?", "active"), search, &users)
//	resp, err := p.Run()
//	if err != nil { ... }
//	return c.JSON(resp)
func (p *Paginator) Run() (*PaginationResponse, error) {
	const cacheTTL = 5 * time.Minute

	itemsCount, totalPages, err := p.totalItems(cacheTTL)
	if err != nil {
		return nil, err
	}

	// Clamp the requested page to the valid range before OFFSET/LIMIT.
	if totalPages > 0 && p.Search.Page > totalPages {
		p.Search.Page = totalPages
	}

	if err := p.Query.Scopes(p.applyPaginationScope()).Find(p.Target).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	return BuildPagination(p.Search, p.Target, itemsCount), nil
}

// BuildPagination constructs a PaginationResponse for handlers that paginate
// outside NewPaginator (e.g. raw SQL inboxes).
func BuildPagination(search SearchOutput, items any, itemsCount int64) *PaginationResponse {
	totalPages := pageCount(itemsCount, search.PageSize)
	page := search.Page
	if totalPages > 0 && page > totalPages {
		page = totalPages
	}
	var nextPage, prevPage *int
	if page < totalPages {
		n := page + 1
		nextPage = &n
	}
	if page > 1 {
		pr := page - 1
		prevPage = &pr
	}
	return &PaginationResponse{
		Items:        items,
		PreviousPage: prevPage,
		Page:         page,
		NextPage:     nextPage,
		PageSize:     search.PageSize,
		TotalPages:   totalPages,
		ItemsCount:   itemsCount,
	}
}

// toAnySlice converts a slice of any element type to []any. It exists so
// the validation library's variadic API can consume a typed slice without
// callers having to redeclare the slice as []any.
func toAnySlice[T any](xs []T) []any {
	out := make([]any, len(xs))
	for i, x := range xs {
		out[i] = x
	}
	return out
}
