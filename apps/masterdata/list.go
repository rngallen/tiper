package masterdata

import (
	"strings"

	"dfms/pkg/export"
	"dfms/pkg/logs"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func parseList(c fiber.Ctx) (response.SearchOutput, error) {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		logs.Error(err)
		return search, response.BadRequestBind(c, err)
	}
	return search, nil
}

func applyLike(q *gorm.DB, search response.SearchOutput, cols ...string) *gorm.DB {
	return response.ApplyLike(q, search, cols...)
}

func applyActive(q *gorm.DB, search response.SearchOutput) *gorm.DB {
	if search.IsActive != nil {
		return q.Where("IsActive = ?", *search.IsActive)
	}
	return q
}

func filterByUID[T any](c fiber.Ctx, db *gorm.DB, q *gorm.DB, queryKey, column string) (*gorm.DB, error) {
	uid := strings.TrimSpace(c.Query(queryKey))
	if uid == "" {
		return q, nil
	}
	var parent T
	if err := db.WithContext(c.Context()).Where("UID = ?", uid).First(&parent).Error; err != nil {
		return q, response.BadRequest(c, strings.TrimSuffix(queryKey, "Id")+" not found")
	}
	id := rowID(&parent)
	if id == 0 {
		return q, response.BadRequest(c, strings.TrimSuffix(queryKey, "Id")+" not found")
	}
	return q.Where(column+" = ?", id), nil
}

func wantsPage(c fiber.Ctx) bool {
	return response.WantsPage(c)
}

func serveCatalogue[T any](
	c fiber.Ctx,
	query *gorm.DB,
	search response.SearchOutput,
	allowed map[string]string,
	defaultCol, sheet, prefix string,
	headers []any,
	mapRow func(T) []any,
	tieBreak ...string,
) error {
	sortColumn, ok := allowed[search.OrderBy]
	if !ok {
		sortColumn = defaultCol
	}
	dir := search.SortDirection
	if !wantsPage(c) && strings.TrimSpace(search.OrderBy) == "" {
		dir = response.SortAsc
	}
	// Code-keyed catalogs (tenders, routes, …) and billing cycles have no ID column.
	tie := "ID"
	if len(tieBreak) > 0 && strings.TrimSpace(tieBreak[0]) != "" {
		tie = tieBreak[0]
	}
	query = query.Order(response.StableOrderTie(sortColumn, dir, tie))
	if search.Export {
		return export.Query(c, query, sheet, prefix, headers, mapRow)
	}
	if wantsPage(c) {
		var items []T
		page, err := response.NewPaginator(c, query, search, &items).Run()
		if err != nil {
			logs.Error(err)
			return response.BadRequest(c, err.Error())
		}
		return response.OkDetail(c, page)
	}
	var rows []T
	if err := query.Find(&rows).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, rows)
}
