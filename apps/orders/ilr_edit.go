package orders

import (
	"errors"
	"fmt"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var errNotDraft = errors.New("only a draft ILR can be edited")

func draftErr(c fiber.Ctx, err error) error {
	if errors.Is(err, errNotDraft) {
		return response.UnprocessableEntity(c, err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NotFound(c, "internal loading request")
	}
	return err
}

func (h handler) addGLRVessel(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	var in glrVesselIn
	if err := bindBody(c, &in); err != nil {
		return err
	}
	vessel, err := h.buildILRVessel(in, row)
	if err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if err := h.ensureVesselCap(row, vessel, 0); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	vessel.RequestID = row.ID
	if err := h.db.Omit("Vessel", "Product", "StockStatus").Create(&vessel).Error; err != nil {
		return writeDup(c, err, "this vessel is already on the request")
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.GantryLoadingRequestContent,
		"ILR "+row.DocumentNumber+" vessel line added", nil, vessel)
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return err
	}
	return response.Created(c, out)
}

func (h handler) updateGLRVessel(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	return h.updateGLRVesselByIndex(c, row)
}

func (h handler) updateGLRVesselByIndex(c fiber.Ctx, row models.GantryLoadingRequest) error {
	var in glrVesselIn
	if err := bindBody(c, &in); err != nil {
		return err
	}
	key := strings.TrimSpace(c.Params("vid"))
	var current models.GantryRequestVessel
	found := false
	for _, v := range row.Vessels {
		if v.UID == key {
			current = v
			found = true
			break
		}
	}
	if !found {
		return response.NotFound(c, "vessel line")
	}
	next, err := h.buildILRVessel(in, row)
	if err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if err := h.ensureVesselCap(row, next, current.ID); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	next.ID = current.ID
	next.UID = current.UID
	next.RequestID = row.ID
	if err := h.db.Omit("Vessel", "Product", "StockStatus").Save(&next).Error; err != nil {
		return writeDup(c, err, "this vessel is already on the request")
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.GantryLoadingRequestContent,
		"ILR "+row.DocumentNumber+" vessel line updated", current, next)
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return err
	}
	return okUpdate(c, out, current, next)
}

func (h handler) deleteGLRVessel(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	key := strings.TrimSpace(c.Params("vid"))
	for _, v := range row.Vessels {
		if v.UID == key {
			if err := h.db.Delete(&models.GantryRequestVessel{}, v.ID).Error; err != nil {
				return err
			}
			recordAudit(c, types.ModuleOrders, types.ActionDelete, row.UID, types.GantryLoadingRequestContent,
				"ILR "+row.DocumentNumber+" vessel line removed", v, nil)
			out, err := h.loadGLR(row.UID)
			if err != nil {
				return err
			}
			return response.OkDetail(c, out)
		}
	}
	return response.NotFound(c, "vessel line")
}

func (h handler) addGLRLine(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	var customer models.Customer
	if err := h.db.First(&customer, row.CustomerID).Error; err != nil {
		return response.BadRequest(c, "customer not found")
	}
	var in glrLineIn
	if err := bindBody(c, &in); err != nil {
		return err
	}
	line, err := h.buildILRLine(in, row, customer)
	if err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if err := h.ensureLineCap(row, line, ""); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		n, err := models.AssignDocumentNumber(tx, "ilo", "ILO")
		if err != nil {
			return err
		}
		line.DocumentNumber = n
		line.RequestID = row.ID
		return tx.Omit("Product", "ByProduct", "Transporter", "Driver", "Truck", "Request", "ToDestination", "ToDistrict").Create(&line).Error
	})
	if err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, line.UID, types.GantryLoadingLineContent,
		"ILO "+line.DocumentNumber+" created", nil, line)
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return err
	}
	return response.Created(c, out)
}

func (h handler) updateGLRLine(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	var current models.GantryLoadingLine
	if err := h.db.Where("UID = ? AND RequestID = ?", c.Params("lid"), row.ID).First(&current).Error; err != nil {
		return response.NotFound(c, "loading instruction")
	}
	var customer models.Customer
	if err := h.db.First(&customer, row.CustomerID).Error; err != nil {
		return response.BadRequest(c, "customer not found")
	}
	var in glrLineIn
	if err := bindBody(c, &in); err != nil {
		return err
	}
	line, err := h.buildILRLine(in, row, customer)
	if err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if err := h.ensureLineCap(row, line, current.UID); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	line.ID = current.ID
	line.UID = current.UID
	line.RequestID = row.ID
	line.DocumentNumber = current.DocumentNumber
	line.Status = current.Status
	if err := h.db.Omit("Product", "ByProduct", "Transporter", "Driver", "Truck", "Request", "ToDestination", "ToDistrict").Save(&line).Error; err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, line.UID, types.GantryLoadingLineContent,
		"ILO "+line.DocumentNumber+" updated", current, line)
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return err
	}
	return okUpdate(c, out, current, line)
}

func (h handler) deleteGLRLine(c fiber.Ctx) error {
	row, err := h.loadDraft(c.Params("uid"))
	if err != nil {
		return draftErr(c, err)
	}
	var current models.GantryLoadingLine
	if err := h.db.Where("UID = ? AND RequestID = ?", c.Params("lid"), row.ID).First(&current).Error; err != nil {
		return response.NotFound(c, "loading instruction")
	}
	if err := h.db.Delete(&current).Error; err != nil {
		return err
	}
	recordAudit(c, types.ModuleOrders, types.ActionDelete, current.UID, types.GantryLoadingLineContent,
		"ILO "+current.DocumentNumber+" deleted", current, nil)
	out, err := h.loadGLR(row.UID)
	if err != nil {
		return err
	}
	return response.OkDetail(c, out)
}

func (h handler) ensureVesselCap(row models.GantryLoadingRequest, next models.GantryRequestVessel, replaceID uint) error {
	sum := decimal.Zero
	for _, v := range row.Vessels {
		if v.ID == replaceID {
			continue
		}
		if v.ProductID == next.ProductID {
			sum = sum.Add(v.Quantity)
		}
	}
	sum = sum.Add(next.Quantity)
	if sum.GreaterThan(orderedQty(row, next.ProductID)) {
		return errors.New("total vessel volume for this product cannot exceed the ordered quantity")
	}
	return nil
}

func (h handler) ensureLineCap(row models.GantryLoadingRequest, next models.GantryLoadingLine, replaceUID string) error {
	sum := decimal.Zero
	for _, ln := range row.Lines {
		if ln.Amended || !ln.IsActive {
			continue
		}
		if ln.UID == replaceUID {
			continue
		}
		if ln.ProductID == next.ProductID {
			sum = sum.Add(ln.RequestedQty)
		}
	}
	sum = sum.Add(next.RequestedQty)
	if sum.GreaterThan(orderedQty(row, next.ProductID)) {
		return errors.New("total loading-instruction volume for this product cannot exceed the ordered quantity")
	}
	return nil
}

func validateEqualTotals(req models.GantryLoadingRequest) error {
	check := func(productID uint, want decimal.Decimal, label string) error {
		if productID == 0 || !want.IsPositive() {
			return nil
		}
		vsum := decimal.Zero
		for _, v := range req.Vessels {
			if v.ProductID == productID {
				vsum = vsum.Add(v.Quantity)
			}
		}
		if !vsum.Equal(want) {
			return fmt.Errorf("vessel volume for %s must equal the requested quantity", label)
		}
		lsum := decimal.Zero
		for _, ln := range req.Lines {
			if ln.Amended || !ln.IsActive {
				continue
			}
			if ln.ProductID == productID {
				lsum = lsum.Add(ln.RequestedQty)
			}
		}
		if !lsum.Equal(want) {
			return fmt.Errorf("loading-instruction volume for %s must equal the requested quantity", label)
		}
		return nil
	}
	if err := check(req.ProductID, req.Quantity, "the ordered product"); err != nil {
		return err
	}
	if req.ByProductID != nil {
		return check(*req.ByProductID, req.ByProductQuantity, "the by-product")
	}
	return nil
}

func writeDup(c fiber.Ctx, err error, msg string) error {
	if err == nil {
		return nil
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "unique") || strings.Contains(s, "duplicate") {
		return response.UnprocessableEntity(c, errors.New(msg))
	}
	return err
}

func (h handler) submitGLRValidated(c fiber.Ctx) error {
	row, err := h.loadGLR(c.Params("uid"))
	if err != nil {
		return err
	}
	if row.Status != types.OrderDraft {
		return response.UnprocessableEntity(c, errors.New("only a draft ILR can be submitted"))
	}
	if err := validateEqualTotals(row); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	if len(row.Lines) == 0 {
		return response.UnprocessableEntity(c, errors.New("at least one loading instruction is required"))
	}
	if len(row.Vessels) == 0 {
		return response.UnprocessableEntity(c, errors.New("at least one vessel is required"))
	}
	return h.submitDoc(c, &models.GantryLoadingRequest{}, types.GantryLoadingRequestContent)
}
