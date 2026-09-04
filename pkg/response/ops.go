// Operational and catalogue list helpers. ServeList is the only list path:
// date window, whitelist sort, Excel, then either a full dump (catalogues)
// or NewPaginator (every operational document list).
package response

import (
	"strings"

	"dfms/pkg/export"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// ListOpts drives ServeList: sort whitelist, optional date window, Excel, pagination.
type ListOpts[T any] struct {
	Query        *gorm.DB
	Search       SearchOutput
	DateColumn   string
	Sort         map[string]string
	DefaultSort  string
	TieBreak     string
	Sheet        string
	File         string
	Headers      []any
	MapRow       func(T) []any
	DumpIfNoPage bool
}

// ApplyLike adds OR LIKE predicates when the client sent a search term.
func ApplyLike(q *gorm.DB, search SearchOutput, cols ...string) *gorm.DB {
	if !search.HasSearch() || len(cols) == 0 {
		return q
	}
	parts := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols))
	for _, col := range cols {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}
		parts = append(parts, col+" LIKE ?")
		args = append(args, search.Search)
	}
	if len(parts) == 0 {
		return q
	}
	return q.Where(strings.Join(parts, " OR "), args...)
}

// WantsPage is true when the client sent page or pageSize (catalogues dump
// all rows when omitted so form pickers keep working).
func WantsPage(c fiber.Ctx) bool {
	return strings.TrimSpace(c.Query("page")) != "" || strings.TrimSpace(c.Query("pageSize")) != ""
}

// ServeList applies date window, whitelist sort, Excel export, and pagination.
// Operational lists leave DumpIfNoPage false so every request is paged.
func ServeList[T any](c fiber.Ctx, opts ListOpts[T]) error {
	search := opts.Search
	col := strings.TrimSpace(opts.DefaultSort)
	if opts.Sort != nil {
		if mapped, ok := opts.Sort[search.OrderBy]; ok && strings.TrimSpace(mapped) != "" {
			col = mapped
		}
	}
	if col == "" {
		col = "ID"
	}

	q := opts.Query
	if opts.DateColumn != "" && !QueryAllDates(c) {
		q = q.Where(opts.DateColumn+" BETWEEN ? AND ?", search.FromDate, search.ToDate)
	}
	q = q.Order(StableOrderTie(col, search.SortDirection, opts.TieBreak))

	if search.Export && opts.MapRow != nil {
		sheet, file := opts.Sheet, opts.File
		if sheet == "" {
			sheet = "Export"
		}
		if file == "" {
			file = "export"
		}
		return export.Query(c, q, sheet, file, opts.Headers, opts.MapRow)
	}

	if opts.DumpIfNoPage && !WantsPage(c) {
		var rows []T
		if err := q.Find(&rows).Error; err != nil {
			return err
		}
		return OkDetail(c, rows)
	}

	var items []T
	resp, err := NewPaginator(c, q, search, &items).Run()
	if err != nil {
		return err
	}
	return OkDetail(c, resp)
}
