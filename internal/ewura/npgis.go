package ewura

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/internal/jobs"
	"dfms/pkg/logs"
	"dfms/pkg/types"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// NpgisConfig is the EWURA NPGIS retailer API (sage-npgis absorbed).
type NpgisConfig struct {
	Enabled     bool
	BaseURL     string
	LicenseNo   string
	APISourceID string
	DepotName   string
}

func (c NpgisConfig) url(path string) string {
	return strings.TrimRight(strings.TrimSpace(c.BaseURL), "/") + "/" + strings.TrimLeft(path, "/")
}

// EnqueueLoading queues a terminal-loading NPGIS record after ALMA completion.
func EnqueueLoading(db *gorm.DB, line *models.GantryLoadingLine, comp *models.Compartmentalization, qty decimal.Decimal) {
	if db == nil || line == nil {
		return
	}
	_ = db.Transaction(func(tx *gorm.DB) error {
		tid, err := models.NextTransactionID(tx)
		if err != nil {
			return err
		}
		payload := models.JSONMap{
			"LoadingOrderNo":           line.DocumentNumber,
			"TruckNo":                  line.TruckPlate,
			"ClientLicenceNo":          line.EwuraLicense,
			"DestinationDistrict":      line.District,
			"DestinationCountryRegion": line.Destination,
			"QuantityinLtr":            almaLitres(qty),
			"LoadingType":              "Bridging",
		}
		return tx.Create(&models.NpgisSubmission{
			Kind:           types.NpgisLoading,
			ReferenceType:  "GantryLoadingLine",
			ReferenceID:    line.ID,
			DocumentNumber: line.DocumentNumber,
			TransactionID:  tid,
			Payload:        payload,
		}).Error
	})
}

// EnqueuePumpOver queues a pump-over NPGIS record after the report is approved.
func EnqueuePumpOver(db *gorm.DB, req *models.PumpOverRequest, qty decimal.Decimal) {
	if db == nil || req == nil {
		return
	}
	_ = db.Transaction(func(tx *gorm.DB) error {
		tid, err := models.NextTransactionID(tx)
		if err != nil {
			return err
		}
		return tx.Create(&models.NpgisSubmission{
			Kind:           types.NpgisPumpOver,
			ReferenceType:  "PumpOverRequest",
			ReferenceID:    req.ID,
			DocumentNumber: req.DocumentNumber,
			TransactionID:  tid,
			Payload: models.JSONMap{
				"LoadingOrderNumber": req.DocumentNumber,
				"QuantityinLtr":      almaLitres(qty),
				"LoadingType":        "Bridging",
			},
		}).Error
	})
}

// EnqueueTank queues a terminal-tank NPGIS record after create or update.
func EnqueueTank(db *gorm.DB, tank *models.Tank, action string) {
	if db == nil || tank == nil || tank.ID == 0 {
		return
	}
	productCode := tank.Product.Code
	if productCode == "" && tank.ProductID > 0 {
		var p models.Product
		if db.Select("Code").First(&p, tank.ProductID).Error == nil {
			productCode = p.Code
		}
	}
	status := "Inactive"
	if tank.IsActive {
		status = "Active"
	}
	_ = db.Transaction(func(tx *gorm.DB) error {
		tid, err := models.NextTransactionID(tx)
		if err != nil {
			return err
		}
		return tx.Create(&models.NpgisSubmission{
			Kind:           types.NpgisTank,
			ReferenceType:  "Tank",
			ReferenceID:    tank.ID,
			DocumentNumber: tank.Code,
			TransactionID:  tid,
			Payload: models.JSONMap{
				"TankNo":          tank.Code,
				"TankDescription": tank.Name,
				"ProductCode":     productCode,
				"CapacityInLtr":   tank.MaximumCapacity.String(),
				"DeadStockInLtr":  tank.DeadStock.String(),
				"Status":          status,
				"Action":          action,
			},
		}).Error
	})
}

func almaLitres(m3 decimal.Decimal) int {
	return int(m3.Mul(decimal.NewFromInt(1000)).Round(0).IntPart())
}

func RegisterNpgisJob(ctx context.Context, m *jobs.Manager, db *gorm.DB, cfg func() NpgisConfig) {
	m.Register(jobs.EwuraNpgis, func() {
		if ctx.Err() != nil {
			return
		}
		if err := FlushNpgis(ctx, db, cfg()); err != nil {
			logs.Errorf("ewura npgis: %v", err)
		}
	})
}

const maxNpgisAttempts = 5

func FlushNpgis(ctx context.Context, db *gorm.DB, cfg NpgisConfig) error {
	if !cfg.Enabled || strings.TrimSpace(cfg.BaseURL) == "" {
		return nil
	}
	var rows []models.NpgisSubmission
	if err := db.Where("Sent = 0 AND Attempts < ?", maxNpgisAttempts).
		Order("ID ASC").Limit(50).Find(&rows).Error; err != nil {
		return err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	for i := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := postNpgis(client, cfg, &rows[i]); err != nil {
			_ = db.Model(&rows[i]).Updates(map[string]any{
				"Attempts":  rows[i].Attempts + 1,
				"LastError": err.Error(),
			}).Error
			logs.Errorf("NPGIS %s: %v", rows[i].DocumentNumber, err)
			continue
		}
		now := time.Now()
		_ = db.Model(&rows[i]).Updates(map[string]any{"Sent": true, "SentAt": now, "LastError": ""}).Error
	}
	return nil
}

func postNpgis(client *http.Client, cfg NpgisConfig, row *models.NpgisSubmission) error {
	path := "submitTerminalLoadingData"
	switch row.Kind {
	case types.NpgisPumpOver:
		path = "submitTerminalPumpOverData"
	case types.NpgisTank:
		path = "submitTerminalTankData"
	}
	record := map[string]any{}
	for k, v := range row.Payload {
		record[k] = v
	}
	record["LicenseNo"] = cfg.LicenseNo
	record["APISourceId"] = cfg.APISourceID
	record["TransId"] = row.TransactionID
	if cfg.DepotName != "" {
		record["LoadingDepot"] = cfg.DepotName
	}
	body, err := json.Marshal(map[string]any{"record": record, "signature": ""})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.url(path), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != 208 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, raw)
	}
	return nil
}
