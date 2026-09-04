package masterdata

import (
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"

	"github.com/gofiber/fiber/v3"
	"github.com/jellydator/validation"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func (h handler) listTrucks(c fiber.Ctx) error {
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := applyActive(applyLike(
		h.db.WithContext(c.Context()).Model(&models.Truck{}),
		search, "PlateNumber", "Trailer", "TrailerTwo", "VehicleType",
	), search)
	return serveCatalogue(c, q, search, map[string]string{
		"plateNumber": "PlateNumber", "trailer": "Trailer", "vehicleType": "VehicleType",
	}, "PlateNumber", "Trucks", "trucks",
		[]any{"Plate", "Trailer", "Type", "Loading", "Active"},
		func(r models.Truck) []any {
			return []any{r.PlateNumber, r.Trailer, r.VehicleType, r.LoadingType, r.IsActive}
		},
	)
}

func (h handler) createTruck(c fiber.Ctx) error {
	var in truckRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	if err := validateTruckShape(in, true); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	mplw, gcwr, tare := decimal.NewFromInt(6000), decimal.NewFromInt(6000), decimal.NewFromInt(2000)
	if in.Mplw != "" {
		mplw, _ = parseDec(in.Mplw)
	}
	if in.Gcwr != "" {
		gcwr, _ = parseDec(in.Gcwr)
	}
	if in.TareWeight != "" {
		tare, _ = parseDec(in.TareWeight)
	}
	vt := resolveVehicleType(in)
	row := models.Truck{
		PlateNumber: in.PlateNumber,
		Trailer:     in.Trailer,
		TrailerTwo:  in.TrailerTwo,
		VehicleType: vt,
		LoadingType: types.LoadingTop,
		LngCng:      false,
		Mplw:        mplw,
		Gcwr:        gcwr,
		TareWeight:  tare,
		IsActive:    true,
	}
	err := h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return syncTruckTanks(tx, row)
	})
	if err != nil {
		return writeErr(c, err, "a truck with this plate already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, row.UID, types.TruckContent, "truck "+row.PlateNumber+" created", nil, row)
	return response.Created(c, truckDetail(h.db, row))
}

func (h handler) updateTruck(c fiber.Ctx) error {
	var row models.Truck
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "truck not found")
	}
	var in truckRequest
	if err := bindBody(c, &in); err != nil {
		return err
	}
	if in.PlateNumber != alphaNumUpper(row.PlateNumber) {
		return response.UnprocessableEntity(c, validation.Errors{
			"plateNumber": validation.NewError("validation_locked", "horse plate cannot be changed after registration"),
		})
	}
	if !types.VehicleTypeConfigured(types.VehicleType(in.VehicleType)) {
		if types.VehicleTypeConfigured(row.VehicleType) {
			in.VehicleType = string(row.VehicleType)
			in.Sanitize()
		} else {
			return response.UnprocessableEntity(c, validation.Errors{
				"vehicleType": validation.NewError("validation_required", "select vehicle type (straight, semi-trailer, or pulling)"),
			})
		}
	}
	if err := validateTruckShape(in, false); err != nil {
		return response.UnprocessableEntity(c, err)
	}
	before := row
	row.Trailer = in.Trailer
	row.TrailerTwo = in.TrailerTwo
	row.VehicleType = types.NormalizeVehicleType(in.VehicleType)
	if in.LoadingType == string(types.LoadingBottom) {
		row.LoadingType = types.LoadingBottom
	} else if in.LoadingType != "" {
		row.LoadingType = types.LoadingTop
	}
	row.LngCng = in.LngCng
	if in.Mplw != "" {
		row.Mplw, _ = parseDec(in.Mplw)
	}
	if in.Gcwr != "" {
		row.Gcwr, _ = parseDec(in.Gcwr)
	}
	if in.TareWeight != "" {
		row.TareWeight, _ = parseDec(in.TareWeight)
	}
	if in.IsActive != nil {
		row.IsActive = *in.IsActive
	}
	err := h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		return syncTruckTanks(tx, row)
	})
	if err != nil {
		return writeErr(c, err, "a truck with this plate already exists")
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.TruckContent, "truck "+row.PlateNumber+" updated", before, row)
	return okUpdate(c, truckDetail(h.db, row), before, row)
}
