package masterdata

import (
	"errors"
	"time"

	"dfms/apps/auth/middleware"
	"dfms/apps/models"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"github.com/gofiber/fiber/v3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func (h handler) getTruck(c fiber.Ctx) error {
	var row models.Truck
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &row); err != nil {
		return notFound(c, err, "truck not found")
	}
	return response.OkDetail(c, truckDetail(h.db, row))
}

func (h handler) listTruckTanks(c fiber.Ctx) error {
	var truck models.Truck
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &truck); err != nil {
		return notFound(c, err, "truck not found")
	}
	var tanks []models.TruckTank
	if err := h.db.WithContext(c.Context()).Where("TruckID = ?", truck.ID).Order("[Index], ID").Find(&tanks).Error; err != nil {
		return err
	}
	out := make([]fiber.Map, 0, len(tanks))
	for _, t := range tanks {
		out = append(out, tankView(h.db, t))
	}
	return response.OkDetail(c, out)
}

func (h handler) saveTruckCalibration(c fiber.Ctx) error {
	var truck models.Truck
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &truck); err != nil {
		return notFound(c, err, "truck not found")
	}
	if !types.VehicleTypeConfigured(truck.VehicleType) {
		return response.UnprocessableEntity(c, errors.New("configure vehicle type before calibration"))
	}
	var tank models.TruckTank
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("tankId"), &tank); err != nil {
		return notFound(c, err, "tank not found")
	}
	if tank.TruckID == nil || *tank.TruckID != truck.ID {
		return response.BadRequest(c, "tank does not belong to this truck")
	}
	var in struct {
		ValidTo string `json:"validTo"`
		Lines   []struct {
			Index    int    `json:"index"`
			Capacity string `json:"capacity"`
		} `json:"lines"`
	}
	if err := bindBody(c, &in); err != nil {
		return err
	}
	to := parseOptionalDate(in.ValidTo)
	if to == nil {
		return response.UnprocessableEntity(c, errors.New("certification end date is required"))
	}
	from := time.Now().UTC().Truncate(24 * time.Hour)
	caps := map[int]decimal.Decimal{}
	for _, ln := range in.Lines {
		if ln.Index < 1 || ln.Index > 10 {
			return response.UnprocessableEntity(c, errors.New("compartment index must be 1–10"))
		}
		d, err := decimal.NewFromString(stripCommas(ln.Capacity))
		if err != nil || d.IsNegative() {
			return response.UnprocessableEntity(c, errors.New("compartment capacity must be a number"))
		}
		caps[ln.Index] = d
	}
	var prev models.TankCalibration
	_ = h.db.WithContext(c.Context()).Preload("Lines").
		Where("TankID = ? AND IsActive = ?", tank.ID, true).Order("ValidTo DESC").First(&prev).Error
	row := models.TankCalibration{TankID: tank.ID, ValidFrom: from, ValidTo: *to, IsActive: true}
	for i := 1; i <= 10; i++ {
		row.Lines = append(row.Lines, models.TankCompartment{Index: i, Capacity: caps[i]})
	}
	err := h.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.TankCalibration{}).Where("TankID = ?", tank.ID).Update("IsActive", false).Error; err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return err
	}
	var before any
	if prev.ID != 0 {
		before = prev
	}
	recordAudit(c, types.ModuleOrders, types.ActionUpdate, row.UID, types.TankCalibrationContent,
		"calibration chart saved for tank "+tank.PlateNumber, before, row)
	return response.Created(c, tankView(h.db, tank))
}

func (h handler) createTruckTank(c fiber.Ctx) error {
	var truck models.Truck
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &truck); err != nil {
		return notFound(c, err, "truck not found")
	}
	if !types.VehicleTypeConfigured(truck.VehicleType) {
		return response.UnprocessableEntity(c, errors.New("configure vehicle type before adding tanks"))
	}
	var in struct {
		PlateNumber string `json:"plateNumber"`
		Index       int    `json:"index"`
	}
	if err := bindBody(c, &in); err != nil {
		return err
	}
	plate := alphaNumUpper(in.PlateNumber)
	if plate == "" {
		return response.UnprocessableEntity(c, errors.New("plate number is required (letters and digits only)"))
	}
	idx := in.Index
	if idx < 1 {
		idx = 1
	}
	tank, err := linkOrCreateTank(h.db.WithContext(c.Context()), truck.ID, plate, idx)
	if err != nil {
		return writeErr(c, err, "could not link tank")
	}
	recordAudit(c, types.ModuleOrders, types.ActionCreate, tank.UID, types.TruckTankContent,
		"truck tank "+tank.PlateNumber+" linked to "+truck.PlateNumber, nil, tank)
	return response.Created(c, tankView(h.db, tank))
}

func truckDetail(db *gorm.DB, row models.Truck) fiber.Map {
	var tanks []models.TruckTank
	_ = db.Where("TruckID = ?", row.ID).Order("[Index], ID").Find(&tanks).Error
	views := make([]fiber.Map, 0, len(tanks))
	total := decimal.Zero
	for _, t := range tanks {
		v := tankView(db, t)
		views = append(views, v)
		if cap, ok := v["capacity"].(string); ok {
			if d, err := decimal.NewFromString(cap); err == nil {
				total = total.Add(d)
			}
		}
	}
	return fiber.Map{
		"id": row.UID, "plateNumber": row.PlateNumber, "trailer": row.Trailer, "trailerTwo": row.TrailerTwo,
		"displayPlate": models.TruckComboPlate(row.PlateNumber, row.Trailer, row.TrailerTwo),
		"vehicleType":  row.VehicleType, "loadingType": row.LoadingType, "lngCng": row.LngCng,
		"mplw": row.Mplw, "gcwr": row.Gcwr, "tareWeight": row.TareWeight, "isActive": row.IsActive,
		"totalCapacity": total.String(),
		"tanks":         views,
	}
}

func tankView(db *gorm.DB, tank models.TruckTank) fiber.Map {
	var cal models.TankCalibration
	_ = db.Preload("Lines").Where("TankID = ? AND IsActive = ?", tank.ID, true).
		Order("ValidTo DESC").First(&cal).Error
	lines := mergeCompartments(cal.Lines)
	validTo := ""
	if !cal.ValidTo.IsZero() {
		validTo = cal.ValidTo.Format("2006-01-02")
	}
	return fiber.Map{
		"id": tank.UID, "plateNumber": tank.PlateNumber, "index": tank.Index, "isActive": tank.IsActive,
		"capacity": tankCapacity(lines).String(), "validTo": validTo,
		"compartments": lines,
	}
}

func (h handler) placeFromLicense(c fiber.Ctx) error {
	no := compact(c.Query("number"))
	if no == "" {
		return response.BadRequest(c, "license number is required")
	}
	var lic models.EwuraPetroleumLicense
	if err := h.db.WithContext(c.Context()).Where("LicenseNumber = ?", no).First(&lic).Error; err != nil {
		return notFound(c, err, "license not found")
	}
	var dest models.Destination
	var dist models.District
	if lic.DistrictName != "" {
		if h.db.Where("Name = ?", lic.DistrictName).First(&dist).Error == nil {
			_ = h.db.First(&dest, dist.DestinationID).Error
		}
	}
	if dest.ID == 0 && lic.RegionName != "" {
		_ = h.db.Where("Name = ?", lic.RegionName).First(&dest).Error
	}
	destName := dest.Name
	if destName == "" {
		destName = lic.RegionName
	}
	distName := dist.Name
	if distName == "" {
		distName = lic.DistrictName
	}
	return response.OkDetail(c, fiber.Map{
		"licenseNumber": lic.LicenseNumber,
		"districtName":  lic.DistrictName,
		"regionName":    lic.RegionName,
		"destinationId": dest.UID,
		"destination":   destName,
		"districtId":    dist.UID,
		"district":      distName,
	})
}

var errUnsupportedMaster = errors.New("unsupported entity")

func (h handler) attachedMaster(c fiber.Ctx, ct types.ContentType) (id uint, label string, err error) {
	db := h.db.WithContext(c.Context())
	uid := c.Params("id")
	switch ct {
	case types.CustomerContent:
		var row models.Customer
		err = firstUID(db, uid, &row)
		return row.ID, row.Code, err
	case types.SupplierContent:
		var row models.Supplier
		err = firstUID(db, uid, &row)
		return row.ID, row.Code, err
	case types.TruckContent:
		var row models.Truck
		err = firstUID(db, uid, &row)
		return row.ID, row.PlateNumber, err
	case types.TankContent:
		var row models.Tank
		err = firstUID(db, uid, &row)
		return row.ID, row.Code, err
	case types.DriverContent:
		var row models.Driver
		err = firstUID(db, uid, &row)
		return row.ID, row.LicenseNumber, err
	default:
		return 0, "", errUnsupportedMaster
	}
}

func (h handler) masterAttachID(c fiber.Ctx, ct types.ContentType, notFoundMsg string) (uint, string, error) {
	id, label, err := h.attachedMaster(c, ct)
	if err != nil {
		if errors.Is(err, errUnsupportedMaster) {
			return 0, "", response.BadRequest(c, "unsupported entity")
		}
		return 0, "", notFound(c, err, notFoundMsg)
	}
	return id, label, nil
}

func (h handler) listEntityAttachments(c fiber.Ctx, ct types.ContentType, notFoundMsg string) error {
	id, _, err := h.masterAttachID(c, ct, notFoundMsg)
	if err != nil {
		return err
	}
	search, err := parseList(c)
	if err != nil {
		return err
	}
	q := h.db.WithContext(c.Context()).Model(&models.Attachment{}).
		Where("EntityType = ? AND EntityID = ?", ct, id)
	if search.IsActive != nil {
		q = q.Where("IsActive = ?", *search.IsActive)
	}
	q = q.Order("CreatedAt DESC")
	if wantsPage(c) {
		var items []models.Attachment
		page, err := response.NewPaginator(c, q, search, &items).Run()
		if err != nil {
			return response.BadRequest(c, err.Error())
		}
		annotateAttachmentUse(h.db.WithContext(c.Context()), items)
		if page != nil {
			page.Items = items
		}
		return response.OkDetail(c, page)
	}
	var rows []models.Attachment
	if err := q.Find(&rows).Error; err != nil {
		return err
	}
	annotateAttachmentUse(h.db.WithContext(c.Context()), rows)
	return response.OkDetail(c, rows)
}

func annotateAttachmentUse(db *gorm.DB, rows []models.Attachment) {
	if len(rows) == 0 {
		return
	}
	ids := make([]uint, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	var used []uint
	_ = db.Model(&models.Attachment{}).Where("CopiedFromID IN ?", ids).
		Pluck("CopiedFromID", &used).Error
	set := make(map[uint]struct{}, len(used))
	for _, id := range used {
		set[id] = struct{}{}
	}
	for i := range rows {
		_, rows[i].InUse = set[rows[i].ID]
	}
}

func (h handler) patchCustomerAttachment(c fiber.Ctx) error {
	var cust models.Customer
	if err := firstUID(h.db.WithContext(c.Context()), c.Params("id"), &cust); err != nil {
		return notFound(c, err, "customer not found")
	}
	var doc models.Attachment
	if err := h.db.Where("UID = ? AND EntityType = ? AND EntityID = ?", c.Params("aid"), types.CustomerContent, cust.ID).
		First(&doc).Error; err != nil {
		return notFound(c, err, "attachment not found")
	}
	var in struct {
		IsActive *bool `json:"isActive"`
	}
	if err := c.Bind().Body(&in); err != nil {
		return response.BadRequestBind(c, err)
	}
	if in.IsActive == nil {
		return response.BadRequest(c, "isActive is required")
	}
	if err := h.db.Model(&doc).Update("IsActive", *in.IsActive).Error; err != nil {
		return err
	}
	doc.IsActive = *in.IsActive
	rows := []models.Attachment{doc}
	annotateAttachmentUse(h.db, rows)
	msg := "Document deactivated"
	if *in.IsActive {
		msg = "Document activated"
	}
	return response.Ok(c, msg, rows[0])
}

func (h handler) uploadEntityAttachments(c fiber.Ctx, ct types.ContentType, notFoundMsg string) error {
	id, label, err := h.masterAttachID(c, ct, notFoundMsg)
	if err != nil {
		return err
	}
	form, err := c.MultipartForm()
	if err != nil || form == nil || len(form.File["files"]) == 0 {
		return response.BadRequest(c, "files are required")
	}
	if err := attachment.UploadAttachments(attachment.UploadAttachmentsRequest{
		Ctx: c, Db: h.db, DocumentNumber: label,
		ContentType: ct, EntityID: id, UploadedBy: middleware.GetUserIDFromContext(c),
		Attachments: form.File["files"],
	}); err != nil {
		return response.BadRequest(c, err.Error())
	}
	var rows []models.Attachment
	_ = h.db.Where("EntityType = ? AND EntityID = ?", ct, id).Order("CreatedAt DESC").Find(&rows).Error
	return response.Ok(c, "File uploaded", rows)
}

func (h handler) downloadEntityAttachment(c fiber.Ctx, ct types.ContentType, notFoundMsg string) error {
	id, _, err := h.masterAttachID(c, ct, notFoundMsg)
	if err != nil {
		return err
	}
	var doc models.Attachment
	if err := h.db.Where("UID = ? AND EntityType = ? AND EntityID = ?", c.Params("aid"), ct, id).First(&doc).Error; err != nil {
		return notFound(c, err, "attachment not found")
	}
	return attachment.DownloadAttachment(attachment.DownloadAttachmentRequest{
		Ctx: c, Db: h.db, AttachmentID: doc.ID, EntityID: id, EntityType: ct,
	})
}

func (h handler) deleteEntityAttachment(c fiber.Ctx, ct types.ContentType, notFoundMsg string) error {
	id, _, err := h.masterAttachID(c, ct, notFoundMsg)
	if err != nil {
		return err
	}
	var doc models.Attachment
	if err := h.db.Where("UID = ? AND EntityType = ? AND EntityID = ?", c.Params("aid"), ct, id).First(&doc).Error; err != nil {
		return notFound(c, err, "attachment not found")
	}
	if doc.CopiedFromID != nil {
		return response.UnprocessableEntity(c, errors.New("copied attachments cannot be removed"))
	}
	var used int64
	if err := h.db.Model(&models.Attachment{}).Where("CopiedFromID = ?", doc.ID).Count(&used).Error; err != nil {
		return err
	}
	if used > 0 {
		return response.UnprocessableEntity(c, errors.New("this attachment is in use and cannot be deleted"))
	}
	if err := h.db.Delete(&doc).Error; err != nil {
		return err
	}
	return response.OkMessage(c, "Attachment removed")
}

func (h handler) listCustomerAttachments(c fiber.Ctx) error {
	return h.listEntityAttachments(c, types.CustomerContent, "customer not found")
}
func (h handler) uploadCustomerAttachments(c fiber.Ctx) error {
	return h.uploadEntityAttachments(c, types.CustomerContent, "customer not found")
}
func (h handler) downloadCustomerAttachment(c fiber.Ctx) error {
	return h.downloadEntityAttachment(c, types.CustomerContent, "customer not found")
}
func (h handler) deleteCustomerAttachment(c fiber.Ctx) error {
	return h.deleteEntityAttachment(c, types.CustomerContent, "customer not found")
}
func (h handler) listTruckAttachments(c fiber.Ctx) error {
	return h.listEntityAttachments(c, types.TruckContent, "truck not found")
}
func (h handler) uploadTruckAttachments(c fiber.Ctx) error {
	return h.uploadEntityAttachments(c, types.TruckContent, "truck not found")
}
func (h handler) downloadTruckAttachment(c fiber.Ctx) error {
	return h.downloadEntityAttachment(c, types.TruckContent, "truck not found")
}
func (h handler) deleteTruckAttachment(c fiber.Ctx) error {
	return h.deleteEntityAttachment(c, types.TruckContent, "truck not found")
}
func (h handler) listTankAttachments(c fiber.Ctx) error {
	return h.listEntityAttachments(c, types.TankContent, "tank not found")
}
func (h handler) uploadTankAttachments(c fiber.Ctx) error {
	return h.uploadEntityAttachments(c, types.TankContent, "tank not found")
}
func (h handler) downloadTankAttachment(c fiber.Ctx) error {
	return h.downloadEntityAttachment(c, types.TankContent, "tank not found")
}
func (h handler) deleteTankAttachment(c fiber.Ctx) error {
	return h.deleteEntityAttachment(c, types.TankContent, "tank not found")
}
func (h handler) listDriverAttachments(c fiber.Ctx) error {
	return h.listEntityAttachments(c, types.DriverContent, "driver not found")
}
func (h handler) uploadDriverAttachments(c fiber.Ctx) error {
	return h.uploadEntityAttachments(c, types.DriverContent, "driver not found")
}
func (h handler) downloadDriverAttachment(c fiber.Ctx) error {
	return h.downloadEntityAttachment(c, types.DriverContent, "driver not found")
}
func (h handler) deleteDriverAttachment(c fiber.Ctx) error {
	return h.deleteEntityAttachment(c, types.DriverContent, "driver not found")
}
func (h handler) listSupplierAttachments(c fiber.Ctx) error {
	return h.listEntityAttachments(c, types.SupplierContent, "supplier not found")
}
func (h handler) uploadSupplierAttachments(c fiber.Ctx) error {
	return h.uploadEntityAttachments(c, types.SupplierContent, "supplier not found")
}
func (h handler) downloadSupplierAttachment(c fiber.Ctx) error {
	return h.downloadEntityAttachment(c, types.SupplierContent, "supplier not found")
}
func (h handler) deleteSupplierAttachment(c fiber.Ctx) error {
	return h.deleteEntityAttachment(c, types.SupplierContent, "supplier not found")
}
