package main

import (
	"fmt"
	"time"

	"dfms/apps/models"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type loadingSummaryKey struct {
	Year      int
	Month     int
	ProductID uint
}

type loadingSummaryAcc struct {
	ProductCode      string
	ProductName      string
	MonthName        string
	LocalLoaded      decimal.Decimal
	TransitLoaded    decimal.Decimal
	LocalRequested   decimal.Decimal
	TransitRequested decimal.Decimal
	LocalWeight      decimal.Decimal
	TransitWeight    decimal.Decimal
}

func rebuildLoadingSummaries(dest *gorm.DB) error {
	if err := dest.Exec("DELETE FROM GantryLoadingSummary").Error; err != nil {
		return err
	}

	transitByStatus := map[uint]bool{}
	var statuses []models.StockStatus
	if err := dest.Select("ID", "IsTransit").Find(&statuses).Error; err != nil {
		return err
	}
	for _, s := range statuses {
		transitByStatus[s.ID] = s.IsTransit
	}

	var loads []models.GantryLoading
	if err := dest.Select("ID", "Year", "Month", "RequestedQty", "StockStatusID").Find(&loads).Error; err != nil {
		return err
	}

	prods, err := loadAllLoadingProducts(dest)
	if err != nil {
		return err
	}

	rows := aggregateLoadingSummaries(loads, prods, transitByStatus)
	if len(rows) == 0 {
		fmt.Printf("  rebuilt 0 gantry loading headers into GantryLoadingSummary\n")
		return nil
	}
	if err := dest.CreateInBatches(rows, 100).Error; err != nil {
		return err
	}
	fmt.Printf("  rebuilt %d gantry loading headers into %d GantryLoadingSummary rows\n", len(loads), len(rows))
	return nil
}

func loadAllLoadingProducts(dest *gorm.DB) ([]models.GantryLoadingProduct, error) {
	var prods []models.GantryLoadingProduct
	// Full scan — no IN list. A Preload on 29k parent IDs exceeds MSSQL's 2100 parameters.
	err := dest.Select("LoadingID", "ProductID", "ProductCode", "ProductName", "StandardVolume", "Weight").
		Find(&prods).Error
	return prods, err
}

func aggregateLoadingSummaries(loads []models.GantryLoading, prods []models.GantryLoadingProduct, transitByStatus map[uint]bool) []models.GantryLoadingSummary {
	type loadKey struct {
		Year, Month   int
		RequestedQty  decimal.Decimal
		StockStatusID uint
	}
	byLoad := make(map[uint]loadKey, len(loads))
	for _, l := range loads {
		byLoad[l.ID] = loadKey{l.Year, l.Month, l.RequestedQty, l.StockStatusID}
	}

	prodsByLoad := make(map[uint][]models.GantryLoadingProduct, len(loads))
	stdByLoad := make(map[uint]decimal.Decimal, len(loads))
	for _, p := range prods {
		if _, ok := byLoad[p.LoadingID]; !ok {
			continue
		}
		prodsByLoad[p.LoadingID] = append(prodsByLoad[p.LoadingID], p)
		stdByLoad[p.LoadingID] = stdByLoad[p.LoadingID].Add(p.StandardVolume)
	}

	agg := map[loadingSummaryKey]*loadingSummaryAcc{}
	for id, load := range byLoad {
		plist := prodsByLoad[id]
		if len(plist) == 0 {
			continue
		}
		totalStd := stdByLoad[id]
		transit := transitByStatus[load.StockStatusID]
		for _, p := range plist {
			requested := load.RequestedQty
			if totalStd.IsPositive() && len(plist) > 1 {
				requested = load.RequestedQty.Mul(p.StandardVolume).Div(totalStd)
			}
			k := loadingSummaryKey{Year: load.Year, Month: load.Month, ProductID: p.ProductID}
			a := agg[k]
			if a == nil {
				a = &loadingSummaryAcc{
					ProductCode: p.ProductCode,
					ProductName: p.ProductName,
					MonthName:   time.Month(load.Month).String(),
				}
				agg[k] = a
			}
			if transit {
				a.TransitLoaded = a.TransitLoaded.Add(p.StandardVolume)
				a.TransitRequested = a.TransitRequested.Add(requested)
				a.TransitWeight = a.TransitWeight.Add(p.Weight)
			} else {
				a.LocalLoaded = a.LocalLoaded.Add(p.StandardVolume)
				a.LocalRequested = a.LocalRequested.Add(requested)
				a.LocalWeight = a.LocalWeight.Add(p.Weight)
			}
		}
	}

	out := make([]models.GantryLoadingSummary, 0, len(agg))
	for k, a := range agg {
		out = append(out, models.GantryLoadingSummary{
			Year: k.Year, Month: k.Month, MonthName: a.MonthName,
			ProductID: k.ProductID, ProductCode: a.ProductCode, ProductName: a.ProductName,
			LocalLoaded: a.LocalLoaded, TransitLoaded: a.TransitLoaded,
			LocalRequested: a.LocalRequested, TransitRequested: a.TransitRequested,
			LocalWeight: a.LocalWeight, TransitWeight: a.TransitWeight,
		})
	}
	return out
}
