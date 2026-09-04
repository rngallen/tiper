package masterdata

import (
	"strings"

	"dfms/apps/models"
	"dfms/pkg/logs"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func (h handler) lookups(c fiber.Ctx) error {
	var cycles []models.BillingCycle
	var tenders []models.ImportTenderType
	var methods []models.DeliveryMethod
	var procs []models.ProcurementMethod
	var routes []models.DischargeRoute
	var contracts []models.ContractType
	var pricing []models.PricingNature
	var fees []models.Fee
	_ = h.db.Order("Days").Find(&cycles).Error
	_ = h.db.Order("Code").Find(&tenders).Error
	_ = h.db.Order("Code").Find(&methods).Error
	_ = h.db.Order("Code").Find(&procs).Error
	_ = h.db.Order("Code").Find(&routes).Error
	_ = h.db.Order("Code").Find(&contracts).Error
	_ = h.db.Order("Code").Find(&pricing).Error
	_ = h.db.Find(&fees).Error
	return response.OkDetail(c, fiber.Map{
		"billingCycles": cycles, "tenders": tenders, "deliveryMethods": methods,
		"procurementMethods": procs, "routes": routes, "contractTypes": contracts,
		"pricingNatures": pricing, "fees": fees,
	})
}

func (h handler) listUOMs(c fiber.Ctx) error {
	var rows []models.UnitOfMeasure
	if err := h.db.WithContext(c.Context()).Order("Code").Find(&rows).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, rows)
}

func (h handler) listCountries(c fiber.Ctx) error {
	var rows []models.Country
	q := h.db.WithContext(c.Context()).Where("IsActive = ?", true).Order("Name")
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("Name LIKE ? OR Code LIKE ?", like, like)
	}
	if err := q.Limit(80).Find(&rows).Error; err != nil {
		logs.Error(err)
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, rows)
}
