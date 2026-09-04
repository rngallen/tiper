package masterdata

import (
	"errors"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func applyVehiclePlateRules(in *truckRequest) {
	if in == nil || !types.VehicleTypeConfigured(types.VehicleType(in.VehicleType)) {
		return
	}
	switch types.NormalizeVehicleType(in.VehicleType) {
	case types.VehicleStraight:
		in.Trailer = in.PlateNumber
		in.TrailerTwo = ""
	case types.VehicleSemi:
		in.TrailerTwo = ""
	}
}

func validateTruckShape(in truckRequest, isCreate bool) error {
	if isCreate && !types.VehicleTypeConfigured(types.VehicleType(in.VehicleType)) {
		if in.PlateNumber == "" {
			return errors.New("horse plate number is required")
		}
		return nil
	}
	horse, tankOne, tankTwo := in.PlateNumber, in.Trailer, in.TrailerTwo
	vt := types.NormalizeVehicleType(in.VehicleType)
	switch vt {
	case types.VehicleStraight:
		if horse == "" {
			return errors.New("straight requires a horse plate")
		}
		if tankOne != "" && tankOne != horse {
			return errors.New("straight: horse plate and tank one must be the same")
		}
		if tankTwo != "" {
			return errors.New("straight: tank two must be empty")
		}
	case types.VehicleSemi:
		if horse == "" || tankOne == "" {
			return errors.New("semi-trailer requires horse plate and tank one plate")
		}
		if tankTwo != "" {
			return errors.New("semi-trailer: tank two must be empty")
		}
	case types.VehiclePulling:
		if horse == "" || tankOne == "" || tankTwo == "" {
			return errors.New("pulling requires horse, tank one, and tank two plate numbers")
		}
	}
	return nil
}

func resolveVehicleType(in truckRequest) types.VehicleType {
	if types.VehicleTypeConfigured(types.VehicleType(in.VehicleType)) {
		return types.NormalizeVehicleType(in.VehicleType)
	}
	return types.VehiclePending
}

func syncTruckTanks(tx *gorm.DB, truck models.Truck) error {
	if !types.VehicleTypeConfigured(truck.VehicleType) {
		return nil
	}
	want := map[int]string{}
	switch types.NormalizeVehicleType(string(truck.VehicleType)) {
	case types.VehicleSemi:
		want[1] = truck.Trailer
	case types.VehiclePulling:
		want[1] = truck.Trailer
		want[2] = truck.TrailerTwo
	default:
		want[1] = truck.PlateNumber
	}
	seen := map[uint]bool{}
	for idx, plate := range want {
		plate = upper(plate)
		if plate == "" {
			continue
		}
		tank, err := linkOrCreateTank(tx, truck.ID, plate, idx)
		if err != nil {
			return err
		}
		seen[tank.ID] = true
	}
	var existing []models.TruckTank
	if err := tx.Where("TruckID = ?", truck.ID).Find(&existing).Error; err != nil {
		return err
	}
	for _, t := range existing {
		if seen[t.ID] {
			continue
		}
		if err := tx.Model(&t).Updates(map[string]any{"TruckID": nil, "IsActive": false}).Error; err != nil {
			return err
		}
	}
	return nil
}

func linkOrCreateTank(tx *gorm.DB, truckID uint, plate string, index int) (models.TruckTank, error) {
	plate = upper(plate)
	var tank models.TruckTank
	// Find+Limit, not First: GORM First() always appends ORDER BY <pk>, so
	// Order("ID DESC").First becomes ORDER BY ID DESC, ID — MSSQL error 169.
	if err := tx.Where("PlateNumber = ?", plate).Order("ID DESC").Limit(1).Find(&tank).Error; err != nil {
		return tank, err
	}
	if tank.ID != 0 {
		tid := truckID
		if err := tx.Model(&tank).Updates(map[string]any{
			"TruckID":  tid,
			"Index":    index,
			"IsActive": true,
		}).Error; err != nil {
			return tank, err
		}
		tank.TruckID = &tid
		tank.Index = index
		tank.IsActive = true
		return tank, nil
	}
	tid := truckID
	tank = models.TruckTank{TruckID: &tid, PlateNumber: plate, Index: index, IsActive: true}
	if err := tx.Create(&tank).Error; err != nil {
		return tank, err
	}
	return tank, nil
}

func defaultCompartments() []models.TankCompartment {
	out := make([]models.TankCompartment, 0, 10)
	for i := 1; i <= 10; i++ {
		out = append(out, models.TankCompartment{Index: i, Capacity: decimal.Zero})
	}
	return out
}

func mergeCompartments(existing []models.TankCompartment) []models.TankCompartment {
	byIdx := map[int]decimal.Decimal{}
	for _, ln := range existing {
		byIdx[ln.Index] = ln.Capacity
	}
	out := defaultCompartments()
	for i := range out {
		if cap, ok := byIdx[out[i].Index]; ok {
			out[i].Capacity = cap
		}
	}
	return out
}

func tankCapacity(lines []models.TankCompartment) decimal.Decimal {
	sum := decimal.Zero
	for _, ln := range lines {
		sum = sum.Add(ln.Capacity)
	}
	return sum
}

func stripCommas(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), ",", "")
}
