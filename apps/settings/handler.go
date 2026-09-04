// Package settings exposes organisation profile data and DB-backed integration
// settings (mail, SMS, Sage 200, session, uploads). Integration secrets are encrypted
// at rest and reloaded into the process-wide integrations store on save.
package settings

import (
	"context"
	"errors"
	"strings"

	authmiddleware "dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/audit"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Handler serves settings endpoints.
type Handler struct {
	db *gorm.DB
}

// NewHandler constructs a Handler.
func NewHandler(gdb *gorm.DB) *Handler { return &Handler{db: gdb} }

// company loads the singleton company row (id = 1), creating an empty one if it
// does not yet exist.
func (h *Handler) company(ctx context.Context) (models.Company, error) {
	db := h.db.WithContext(ctx)
	var company models.Company
	err := db.First(&company).Error
	if err == gorm.ErrRecordNotFound {
		company = models.Company{ID: 1}
		if err := db.Create(&company).Error; err != nil {
			return company, err
		}
		return company, nil
	}
	return company, err
}

// GetCompany returns the company profile.
func (h *Handler) GetCompany(c fiber.Ctx) error {
	company, err := h.company(c.Context())
	if err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, company)
}

// UpdateCompany updates the company profile (records a before/after audit).
func (h *Handler) UpdateCompany(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	company, err := h.company(c.Context())
	if err != nil {
		return response.InternalServerError(c)
	}
	before := company

	var body companyRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	company.Name = body.Name
	company.TinNumber = body.TinNumber
	company.VrnNumber = body.VrnNumber
	company.IsoNumber = body.IsoNumber
	company.Address = body.Address
	company.Address2 = body.Address2
	company.City = body.City
	company.Country = body.Country
	company.PostalCode = body.PostalCode
	company.Phone = body.Phone
	company.Email = body.Email
	company.Website = body.Website
	company.PortalURL = strings.TrimRight(strings.TrimSpace(body.PortalURL), "/")
	// Currency is set once: only accept a new code when the company has none.
	existing := ""
	if company.CurrencyCode != nil {
		existing = *company.CurrencyCode
	}
	if existing == "" {
		if code := strings.TrimSpace(body.CurrencyCode); code != "" {
			company.CurrencyCode = &code
		}
	}

	if err := db.Save(&company).Error; err != nil {
		return response.InternalServerError(c)
	}

	if changes := audit.DropMetaKeys(audit.Diff(before, company)); len(changes) > 0 && audit.Default != nil {
		entry := audit.AuditEntry(
			c, types.ModuleSettings, types.ActionUpdate, "", types.CompanyContent,
			audit.EnrichDescription("updated company profile", changes),
		)
		entry.Changes = changes
		_ = audit.Default.Record(c.Context(), nil, entry)
	}
	return response.Ok(c, audit.UpdateMessage(before, company), company)
}

// Currencies returns the active currency list (for dropdowns and posting).
func (h *Handler) Currencies(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	var currencies []models.Currency
	if err := db.Where("IsActive = 1").Order("Code asc").Find(&currencies).Error; err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, currencies)
}

// CreateCurrency adds an ISO currency identified by its 3-letter code.
func (h *Handler) CreateCurrency(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())
	var body currencyCreateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequestBind(c, err)
	}
	if err := body.Validate(c.Context()); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	code := strings.ToUpper(strings.TrimSpace(body.Code))
	symbol := strings.TrimSpace(body.Symbol)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = code
	}

	var clash models.Currency
	err := db.Where("Code = ?", code).Take(&clash).Error
	if err == nil {
		return response.Conflict(c, "currency "+code+" is already registered")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.InternalServerError(c)
	}

	createdBy := authmiddleware.GetUserIDFromContext(c)
	if createdBy == 0 {
		return response.Unauthorized(c, "Unauthorized")
	}
	row := models.Currency{
		Code:        code,
		Name:        name,
		Symbol:      symbol,
		IsActive:    true,
		CreatedByID: createdBy,
	}
	if err := db.Create(&row).Error; err != nil {
		return response.InternalServerError(c)
	}
	audit.RecordHTTP(c, types.ModuleSettings, types.ActionCreate, row.Code, types.CurrencyContent,
		"registered currency "+row.Code, nil, row)
	return response.Created(c, row)
}

// Countries returns the active ISO 3166-1 country catalogue (for address dropdowns).
func (h *Handler) Countries(c fiber.Ctx) error {
	db := h.db.WithContext(c.Context())

	var countries []models.Country
	if err := db.Where("IsActive = 1").Order("Name asc").Find(&countries).Error; err != nil {
		return response.InternalServerError(c)
	}
	return response.OkDetail(c, countries)
}
