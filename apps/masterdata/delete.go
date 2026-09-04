package masterdata

import (
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type colRef struct {
	table, column string
}

func countCol(db *gorm.DB, table, column string, id uint) (int64, error) {
	var n int64
	err := db.Table(table).Where(column+" = ?", id).Limit(1).Count(&n).Error
	return n, err
}

func (h handler) usedOn(c fiber.Ctx, id uint, refs []colRef) (bool, error) {
	db := h.db.WithContext(c.Context())
	for _, r := range refs {
		n, err := countCol(db, r.table, r.column, id)
		if err != nil {
			return false, err
		}
		if n > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (h handler) rejectInUse(c fiber.Ctx, row any, label string) error {
	if err := h.db.WithContext(c.Context()).Model(row).Update("HasData", true).Error; err != nil {
		return err
	}
	return response.Conflict(c, label+" is in use and cannot be deleted. Deactivate it instead.")
}

func (h handler) deleteIfUnused(c fiber.Ctx, row any, id uint, hasData bool, label string, refs []colRef, extra func(tx *gorm.DB) error) error {
	if hasData {
		return response.Conflict(c, label+" is in use and cannot be deleted. Deactivate it instead.")
	}
	used, err := h.usedOn(c, id, refs)
	if err != nil {
		return err
	}
	if used {
		return h.rejectInUse(c, row, label)
	}
	err = h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if extra != nil {
			if err := extra(tx); err != nil {
				return err
			}
		}
		return tx.Delete(row).Error
	})
	if err != nil {
		return writeErr(c, err, "could not delete "+label)
	}
	recordDeleted(c, row, label)
	return response.Deleted(c)
}

func recordDeleted(c fiber.Ctx, row any, label string) {
	switch m := row.(type) {
	case *models.Customer:
		recordAudit(c, types.ModuleCustomer, types.ActionDelete, m.UID, types.CustomerContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Supplier:
		recordAudit(c, types.ModuleCustomer, types.ActionDelete, m.UID, types.SupplierContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Vessel:
		recordAudit(c, types.ModuleInventory, types.ActionDelete, m.UID, types.VesselContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Product:
		recordAudit(c, types.ModuleInventory, types.ActionDelete, m.UID, types.ProductContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.StockCategory:
		recordAudit(c, types.ModuleInventory, types.ActionDelete, m.UID, types.StockCategoryContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Tank:
		recordAudit(c, types.ModuleInventory, types.ActionDelete, m.UID, types.TankContent, label+" "+m.Code+" deleted", *m, nil)
	case *models.Depot:
		recordAudit(c, types.ModuleInventory, types.ActionDelete, m.UID, types.DepotContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Transporter:
		recordAudit(c, types.ModuleOrders, types.ActionDelete, m.UID, types.TransporterContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Driver:
		recordAudit(c, types.ModuleOrders, types.ActionDelete, m.UID, types.DriverContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.Truck:
		recordAudit(c, types.ModuleOrders, types.ActionDelete, m.UID, types.TruckContent, label+" "+m.PlateNumber+" deleted", *m, nil)
	case *models.Destination:
		recordAudit(c, types.ModuleOrders, types.ActionDelete, m.UID, types.DestinationContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.District:
		recordAudit(c, types.ModuleOrders, types.ActionDelete, m.UID, types.DistrictContent, label+" "+m.Name+" deleted", *m, nil)
	case *models.StockStatus:
		recordAudit(c, types.ModuleInventory, types.ActionDelete, m.UID, types.StockStatusContent, label+" "+m.Name+" deleted", *m, nil)
	}
}

func (h handler) deleteCustomer(c fiber.Ctx) error {
	var row models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "customer not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "customer", []colRef{
		{"ReceiptDetail", "CustomerID"},
		{"GantryLoadingRequest", "CustomerID"},
		{"PumpOverRequest", "CustomerID"},
		{"IttTransfer", "FromCustomerID"},
		{"IttTransfer", "ToCustomerID"},
		{"ZerolizationTransfer", "CustomerID"},
		{"FinancialHoldReleaseLine", "CustomerID"},
		{"Depot", "CustomerID"},
		{"StockMovement", "CustomerID"},
		{"StockBalance", "CustomerID"},
		{"ChangeOfService", "CustomerID"},
	}, func(tx *gorm.DB) error {
		if err := tx.Where("CustomerID = ?", row.ID).Delete(&models.CustomerBillingAccount{}).Error; err != nil {
			return err
		}
		if err := tx.Where("OwnerKind = ? AND OwnerID = ?", ownerCustomer, row.ID).
			Delete(&models.SageAccountOwner{}).Error; err != nil {
			return err
		}
		return tx.Where("EntityType = ? AND EntityID = ?", types.CustomerContent, row.ID).
			Delete(&models.Attachment{}).Error
	})
}

func (h handler) deleteVessel(c fiber.Ctx) error {
	var row models.Vessel
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "vessel not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "vessel", []colRef{
		{"Receipt", "VesselID"},
		{"IttTransfer", "VesselID"},
		{"ZerolizationTransfer", "FromVesselID"},
		{"ZerolizationTransfer", "ToVesselID"},
		{"FinancialHoldReleaseLine", "VesselID"},
		{"GantryRequestVessel", "VesselID"},
		{"PumpOverVessel", "VesselID"},
		{"GantryVesselLoading", "VesselID"},
		{"StockMovement", "VesselID"},
		{"StockBalance", "VesselID"},
		{"ChangeOfService", "VesselID"},
	}, nil)
}

func (h handler) deleteSupplier(c fiber.Ctx) error {
	var row models.Supplier
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "supplier not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "supplier", []colRef{
		{"Receipt", "SupplierID"},
	}, func(tx *gorm.DB) error {
		if err := tx.Where("SupplierID = ?", row.ID).Delete(&models.SupplierBillingAccount{}).Error; err != nil {
			return err
		}
		if err := tx.Where("OwnerKind = ? AND OwnerID = ?", ownerSupplier, row.ID).
			Delete(&models.SageAccountOwner{}).Error; err != nil {
			return err
		}
		return tx.Where("EntityType = ? AND EntityID = ?", types.SupplierContent, row.ID).
			Delete(&models.Attachment{}).Error
	})
}

func (h handler) deleteProduct(c fiber.Ctx) error {
	var row models.Product
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "product not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "product", []colRef{
		{"Receipt", "ProductID"},
		{"GantryLoadingRequest", "ProductID"},
		{"GantryLoadingLine", "ProductID"},
		{"PumpOverRequest", "ProductID"},
		{"IttTransfer", "ProductID"},
		{"ZerolizationTransfer", "ProductID"},
		{"FinancialHoldReleaseLine", "ProductID"},
		{"Tank", "ProductID"},
		{"StockMovement", "ProductID"},
		{"StockBalance", "ProductID"},
		{"LineContent", "ProductID"},
		{"MiLoss", "ProductID"},
		{"MiLossProduct", "ProductID"},
		{"ChangeOfService", "ProductID"},
	}, nil)
}

func (h handler) deleteCategory(c fiber.Ctx) error {
	var row models.StockCategory
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "category not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "category", []colRef{
		{"Product", "StockCategoryID"},
	}, nil)
}

func (h handler) deleteTank(c fiber.Ctx) error {
	var row models.Tank
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "tank not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "tank", []colRef{
		{"PhysicalDip", "TankID"},
	}, func(tx *gorm.DB) error {
		return tx.Where("EntityType = ? AND EntityID = ?", types.TankContent, row.ID).
			Delete(&models.Attachment{}).Error
	})
}

func (h handler) deleteDepot(c fiber.Ctx) error {
	var row models.Depot
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "depot not found")
	}
	if row.IsInternal {
		return response.Conflict(c, "the internal depot cannot be deleted")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "depot", []colRef{
		{"ReceiptDetail", "DepotID"},
		{"PumpOverRequest", "DepotID"},
		{"IttTransfer", "DepotID"},
	}, nil)
}

func (h handler) deleteTransporter(c fiber.Ctx) error {
	var row models.Transporter
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "transporter not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "transporter", []colRef{
		{"GantryLoadingLine", "TransporterID"},
	}, nil)
}

func (h handler) deleteDriver(c fiber.Ctx) error {
	var row models.Driver
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "driver not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "driver", []colRef{
		{"GantryLoadingLine", "DriverID"},
	}, func(tx *gorm.DB) error {
		return tx.Where("EntityType = ? AND EntityID = ?", types.DriverContent, row.ID).
			Delete(&models.Attachment{}).Error
	})
}

func (h handler) deleteTruck(c fiber.Ctx) error {
	var row models.Truck
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "truck not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "truck", []colRef{
		{"GantryLoadingLine", "TruckID"},
	}, func(tx *gorm.DB) error {
		var tanks []models.TruckTank
		if err := tx.Where("TruckID = ?", row.ID).Find(&tanks).Error; err != nil {
			return err
		}
		ids := make([]uint, 0, len(tanks))
		for i := range tanks {
			ids = append(ids, tanks[i].ID)
		}
		if len(ids) > 0 {
			if err := tx.Where("TankID IN ?", ids).Delete(&models.TankCalibration{}).Error; err != nil {
				return err
			}
			if err := tx.Where("ID IN ?", ids).Delete(&models.TruckTank{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("EntityType = ? AND EntityID = ?", types.TruckContent, row.ID).
			Delete(&models.Attachment{}).Error
	})
}

func (h handler) deleteDestination(c fiber.Ctx) error {
	var row models.Destination
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "destination not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "destination", []colRef{
		{"District", "DestinationID"},
		{"GantryLoadingLine", "DestinationID"},
	}, nil)
}

func (h handler) deleteDistrict(c fiber.Ctx) error {
	var row models.District
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "district not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "district", []colRef{
		{"GantryLoadingLine", "DistrictID"},
	}, nil)
}

func (h handler) deleteStatus(c fiber.Ctx) error {
	var row models.StockStatus
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "status not found")
	}
	return h.deleteIfUnused(c, &row, row.ID, row.HasData, "stock status", []colRef{
		{"ReceiptDetail", "StockStatusID"},
		{"GantryLoadingRequest", "StockStatusID"},
		{"PumpOverRequest", "StockStatusID"},
		{"IttTransfer", "StockStatusID"},
		{"ZerolizationTransfer", "StockStatusID"},
		{"FinancialHoldReleaseLine", "StockStatusID"},
		{"StockStatus", "ParentID"},
	}, nil)
}
