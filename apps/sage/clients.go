package sage

import (
	"strings"

	"dfms/apps/models"
	internalsage "dfms/internal/sage"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type handler struct{ db *gorm.DB }

type sageClientDTO struct {
	Account      string `json:"account"`
	Name         string `json:"name"`
	OnHold       bool   `json:"onHold"`
	CurrencyCode string `json:"currencyCode"`
	Available    bool   `json:"available"`
	InUseBy      string `json:"inUseBy,omitempty"`
}

func (h handler) listClients(c fiber.Ctx) error {
	sageDB := db.Sage()
	if sageDB == nil {
		return response.ServiceUnavailable(c, "Sage 200 is not connected")
	}
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequestBind(c, err)
	}
	q := internalsage.ListQuery(sageDB.WithContext(c.Context()), search.Search)
	var rows []internalsage.Client
	page, err := response.NewPaginator(c, q, search, &rows).Run()
	if err != nil {
		logs.Error(err)
		return response.BadRequest(c, err.Error())
	}
	kind := strings.ToLower(strings.TrimSpace(c.Query("ownerType")))
	ownerID := h.ownerID(c, kind, c.Query("ownerId"))
	claimed := h.claimsFor(c, accountsOf(rows))
	out := make([]sageClientDTO, 0, len(rows))
	for i := range rows {
		rows[i].Finish()
		if rows[i].OnHold || rows[i].Account == "" {
			continue
		}
		dto := sageClientDTO{
			Account:      rows[i].Account,
			Name:         rows[i].Name,
			OnHold:       rows[i].OnHold,
			CurrencyCode: rows[i].CurrencyCode,
			Available:    rows[i].CurrencyCode != "",
		}
		if cl, ok := claimed[strings.ToUpper(rows[i].Account)]; ok {
			if kind == "" || cl.OwnerKind != kind || cl.OwnerID != ownerID {
				dto.Available = false
				dto.InUseBy = cl.Label
			}
		}
		out = append(out, dto)
	}
	if page != nil {
		page.Items = out
	}
	return response.OkDetail(c, page)
}

func accountsOf(rows []internalsage.Client) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Account != "" {
			out = append(out, r.Account)
		}
	}
	return out
}

type claimInfo struct {
	OwnerKind string
	OwnerID   uint
	Label     string
}

func (h handler) claimsFor(c fiber.Ctx, accounts []string) map[string]claimInfo {
	out := map[string]claimInfo{}
	if len(accounts) == 0 || h.db == nil {
		return out
	}
	var rows []models.SageAccountOwner
	if err := h.db.WithContext(c.Context()).Where("SageAccount IN ?", accounts).Find(&rows).Error; err != nil {
		logs.Error(err)
		return out
	}
	var custIDs, suppIDs []uint
	for _, r := range rows {
		if r.OwnerKind == "customer" {
			custIDs = append(custIDs, r.OwnerID)
		} else if r.OwnerKind == "supplier" {
			suppIDs = append(suppIDs, r.OwnerID)
		}
	}
	custNames := customerNames(h.db.WithContext(c.Context()), custIDs)
	suppNames := supplierNames(h.db.WithContext(c.Context()), suppIDs)
	for _, r := range rows {
		label := r.OwnerKind
		switch r.OwnerKind {
		case "customer":
			if n := custNames[r.OwnerID]; n != "" {
				label = "Customer " + n
			} else {
				label = "another customer"
			}
		case "supplier":
			if n := suppNames[r.OwnerID]; n != "" {
				label = "Supplier " + n
			} else {
				label = "another supplier"
			}
		}
		out[strings.ToUpper(r.SageAccount)] = claimInfo{
			OwnerKind: r.OwnerKind, OwnerID: r.OwnerID, Label: label,
		}
	}
	return out
}

func (h handler) ownerID(c fiber.Ctx, kind, uid string) uint {
	uid = strings.TrimSpace(uid)
	if uid == "" || h.db == nil {
		return 0
	}
	switch kind {
	case "customer":
		var row models.Customer
		if h.db.WithContext(c.Context()).Select("ID").Where("UID = ?", uid).First(&row).Error == nil {
			return row.ID
		}
	case "supplier":
		var row models.Supplier
		if h.db.WithContext(c.Context()).Select("ID").Where("UID = ?", uid).First(&row).Error == nil {
			return row.ID
		}
	}
	return 0
}

func customerNames(db *gorm.DB, ids []uint) map[uint]string {
	out := map[uint]string{}
	if len(ids) == 0 {
		return out
	}
	var rows []models.Customer
	if err := db.Select("ID", "Name", "Code").Where("ID IN ?", ids).Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		out[r.ID] = displayName(r.Name, r.Code)
	}
	return out
}

func supplierNames(db *gorm.DB, ids []uint) map[uint]string {
	out := map[uint]string{}
	if len(ids) == 0 {
		return out
	}
	var rows []models.Supplier
	if err := db.Select("ID", "Name", "Code").Where("ID IN ?", ids).Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		out[r.ID] = displayName(r.Name, r.Code)
	}
	return out
}

func displayName(name, code string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(code)
}
