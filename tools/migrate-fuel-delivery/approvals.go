package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"
)

// Django models that have a DFMS workflow counterpart. Pump-over reports are
// not copied — that transaction will be redesigned.
var djangoApprovalModels = map[string]types.ContentType{
	"gantryilr":                  types.GantryLoadingRequestContent,
	"pipelinedeliveryrequest":    types.PumpOverRequestContent,
	"transferitt":                types.IttTransferContent,
	"gantrycompartmentalization": types.CompartmentalizationContent,
}

type djangoStep struct {
	objectID int64
	actedAt  time.Time
	actType  string
	actName  string
	userID   int64
	comment  string
}

// copyApprovalTrails writes an immutable ApprovalTrail JSON on approved
// Django-copied documents. It never creates ProcessInstance / Task / Event
// rows — DFMS workflow stays for native documents only.
func copyApprovalTrails(ctx context.Context, pg *pgx.Conn, dest *gorm.DB) error {
	ctIDs := djangoContentTypeIDs(ctx, pg)
	stepsByCT := map[types.ContentType]map[int64][]djangoStep{}
	for model, ct := range djangoApprovalModels {
		id, ok := ctIDs[model]
		if !ok {
			continue
		}
		steps := loadDjangoApprovalSteps(ctx, pg, id)
		if len(steps) == 0 {
			continue
		}
		byObj := map[int64][]djangoStep{}
		for _, s := range steps {
			byObj[s.objectID] = append(byObj[s.objectID], s)
		}
		stepsByCT[ct] = byObj
	}

	n := 0
	n += stampILRTrails(dest, stepsByCT[types.GantryLoadingRequestContent])
	n += stampPDOTrails(dest, stepsByCT[types.PumpOverRequestContent])
	n += stampITTTrails(dest, stepsByCT[types.IttTransferContent])
	n += stampCompTrails(dest, stepsByCT[types.CompartmentalizationContent])
	fmt.Printf("  approval trails: %d approved document(s)\n", n)
	return nil
}

func stampILRTrails(dest *gorm.DB, byObj map[int64][]djangoStep) int {
	var rows []models.GantryLoadingRequest
	dest.Select("ID", "DjangoID", "CreatedByID", "CreatedAt").
		Where("DjangoID > 0 AND Status IN ?", []types.OrderStatus{
			types.OrderApproved, types.OrderCompleted, types.OrderClosed,
		}).Find(&rows)
	n := 0
	for _, r := range rows {
		trail := buildTrail(dest, r.CreatedByID, r.CreatedAt, byObj[int64(r.DjangoID)])
		if len(trail) == 0 {
			continue
		}
		if err := dest.Model(&models.GantryLoadingRequest{}).Where("ID = ?", r.ID).
			Update("ApprovalTrail", trail).Error; err != nil {
			fmt.Printf("  skip ILR approval %d: %v\n", r.ID, err)
			continue
		}
		n++
	}
	return n
}

func stampPDOTrails(dest *gorm.DB, byObj map[int64][]djangoStep) int {
	var rows []models.PumpOverRequest
	dest.Select("ID", "DjangoID", "CreatedByID", "CreatedAt").
		Where("DjangoID > 0 AND Status IN ?", []types.OrderStatus{
			types.OrderApproved, types.OrderCompleted, types.OrderClosed,
		}).Find(&rows)
	n := 0
	for _, r := range rows {
		trail := buildTrail(dest, r.CreatedByID, r.CreatedAt, byObj[int64(r.DjangoID)])
		if len(trail) == 0 {
			continue
		}
		if err := dest.Model(&models.PumpOverRequest{}).Where("ID = ?", r.ID).
			Update("ApprovalTrail", trail).Error; err != nil {
			fmt.Printf("  skip PDO approval %d: %v\n", r.ID, err)
			continue
		}
		n++
	}
	return n
}

func stampITTTrails(dest *gorm.DB, byObj map[int64][]djangoStep) int {
	var rows []models.IttTransfer
	dest.Select("ID", "DjangoID", "CreatedByID", "CreatedAt").
		Where("DjangoID > 0 AND Status IN ?", []types.DocumentStatus{
			types.DocApproved, types.DocPosted,
		}).Find(&rows)
	n := 0
	for _, r := range rows {
		trail := buildTrail(dest, r.CreatedByID, r.CreatedAt, byObj[int64(r.DjangoID)])
		if len(trail) == 0 {
			continue
		}
		if err := dest.Model(&models.IttTransfer{}).Where("ID = ?", r.ID).
			Update("ApprovalTrail", trail).Error; err != nil {
			fmt.Printf("  skip ITT approval %d: %v\n", r.ID, err)
			continue
		}
		n++
	}
	return n
}

func stampCompTrails(dest *gorm.DB, byObj map[int64][]djangoStep) int {
	var rows []models.Compartmentalization
	dest.Select("ID", "DjangoID", "CreatedByID", "CreatedAt").
		Where("DjangoID > 0 AND Status IN ?", []types.OrderStatus{
			types.OrderApproved, types.OrderCompleted, types.OrderClosed,
			types.OrderLoaded, types.OrderInProgress, types.OrderRunning,
		}).Find(&rows)
	n := 0
	for _, r := range rows {
		trail := buildTrail(dest, r.CreatedByID, r.CreatedAt, byObj[int64(r.DjangoID)])
		if len(trail) == 0 {
			continue
		}
		if err := dest.Model(&models.Compartmentalization{}).Where("ID = ?", r.ID).
			Update("ApprovalTrail", trail).Error; err != nil {
			fmt.Printf("  skip comp approval %d: %v\n", r.ID, err)
			continue
		}
		n++
	}
	return n
}

func buildTrail(dest *gorm.DB, createdBy uint, createdAt time.Time, steps []djangoStep) models.ApprovalTrail {
	out := models.ApprovalTrail{}
	for _, s := range steps {
		actType, actName := normalizeDjangoAct(s.actType, s.actName)
		if actType == "pending" {
			continue
		}
		uid := userByDjango(dest, 0, s.userID)
		name, title := userLabel(dest, uid)
		out = append(out, models.ApprovalStep{
			ActedAt:  s.actedAt,
			ActType:  actType,
			ActName:  actName,
			UserName: name,
			Title:    title,
			Comment:  strings.TrimSpace(s.comment),
		})
	}
	if createdBy > 0 && !trailHasInitiate(out) {
		name, title := userLabel(dest, createdBy)
		init := models.ApprovalStep{
			ActedAt: createdAt, ActType: "initiate", ActName: "Initiated",
			UserName: name, Title: title,
		}
		out = append(models.ApprovalTrail{init}, out...)
	}
	return out
}

func trailHasInitiate(t models.ApprovalTrail) bool {
	for _, s := range t {
		if s.ActType == "initiate" {
			return true
		}
	}
	return false
}

func userLabel(dest *gorm.DB, userID uint) (name, title string) {
	if userID == 0 {
		return "", ""
	}
	var row struct {
		FirstName string
		LastName  string
		Title     string
	}
	_ = dest.Raw(`
		SELECT COALESCE(u.FirstName,'') AS FirstName, COALESCE(u.LastName,'') AS LastName,
			COALESCE(p.Title,'') AS Title
		FROM [User] u
		LEFT JOIN Profile p ON p.UserID = u.ID
		WHERE u.ID = ?`, userID).Scan(&row).Error
	return strings.TrimSpace(row.FirstName + " " + row.LastName), strings.TrimSpace(row.Title)
}

func normalizeDjangoAct(actType, actName string) (string, string) {
	raw := strings.ToLower(strings.TrimSpace(actType + " " + actName))
	name := strings.TrimSpace(actName)
	if name == "" {
		name = strings.TrimSpace(actType)
	}
	switch {
	case strings.Contains(raw, "reject"):
		return "reject", firstNonEmpty(name, "Rejected")
	case strings.Contains(raw, "approv") || strings.Contains(raw, "agree") || strings.Contains(raw, "complete"):
		return "approve", firstNonEmpty(name, "Approved")
	case strings.Contains(raw, "submit") || strings.Contains(raw, "init"):
		return "initiate", firstNonEmpty(name, "Submitted")
	case strings.Contains(raw, "pending") || strings.Contains(raw, "assigned") || strings.Contains(raw, "new"):
		return "pending", name
	default:
		return firstNonEmpty(actType, "approve"), firstNonEmpty(name, "Approved")
	}
}

func djangoContentTypeIDs(ctx context.Context, pg *pgx.Conn) map[string]int64 {
	out := map[string]int64{}
	for _, table := range []string{"django_content_type", "DjangoContentType"} {
		rows, err := pg.Query(ctx, `SELECT id, lower(model) FROM `+quoteIdent(table))
		if err != nil {
			continue
		}
		for rows.Next() {
			var id int64
			var model string
			if err := rows.Scan(&id, &model); err != nil {
				continue
			}
			model = strings.ToLower(strings.TrimSpace(model))
			if _, want := djangoApprovalModels[model]; want {
				out[model] = id
			}
		}
		rows.Close()
		if len(out) > 0 {
			return out
		}
	}
	return out
}

func quoteIdent(name string) string {
	if strings.Contains(name, ".") {
		return name
	}
	if name == strings.ToLower(name) {
		return name
	}
	return `"` + name + `"`
}

func loadDjangoApprovalSteps(ctx context.Context, pg *pgx.Conn, contentTypeID int64) []djangoStep {
	if contentTypeID == 0 {
		return nil
	}
	tables := gfkTables(ctx, pg)
	var out []djangoStep
	for _, t := range tables {
		steps := queryGFKSteps(ctx, pg, t, contentTypeID)
		if len(steps) > 0 {
			out = append(out, steps...)
		}
	}
	return out
}

type gfkTable struct {
	name, ctCol, objectCol, userCol, commentCol, timeCol, actCol string
}

func gfkTables(ctx context.Context, pg *pgx.Conn) []gfkTable {
	rows, err := pg.Query(ctx, `
		SELECT c.table_name, c.column_name
		FROM information_schema.columns c
		WHERE c.table_schema = 'public'
		ORDER BY c.table_name, c.ordinal_position`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	type cols struct {
		ct, obj, user, comment, when, act string
	}
	byTable := map[string]*cols{}
	for rows.Next() {
		var table, col string
		if err := rows.Scan(&table, &col); err != nil {
			continue
		}
		c := byTable[table]
		if c == nil {
			c = &cols{}
			byTable[table] = c
		}
		switch strings.ToLower(col) {
		case "content_type_id", "contenttype_id":
			c.ct = col
		case "object_id", "objectid":
			c.obj = col
		case "user_id", "owner_id", "approved_by_id", "actor_id", "performed_by_id":
			if c.user == "" {
				c.user = col
			}
		case "comment", "comments", "remarks", "notes", "message":
			if c.comment == "" {
				c.comment = col
			}
		case "created", "created_at", "finished", "finished_at", "completed_at", "acted_at", "approved_at":
			if c.when == "" {
				c.when = col
			}
		case "status", "action", "act_type", "act_name", "flow_task", "transition":
			if c.act == "" {
				c.act = col
			}
		}
	}
	var out []gfkTable
	for table, c := range byTable {
		if c.ct == "" || c.obj == "" {
			continue
		}
		low := strings.ToLower(table)
		if strings.Contains(low, "permission") || strings.Contains(low, "logentry") ||
			strings.Contains(low, "admin_log") || strings.Contains(low, "content_type") {
			continue
		}
		out = append(out, gfkTable{
			name: table, ctCol: c.ct, objectCol: c.obj, userCol: c.user,
			commentCol: c.comment, timeCol: c.when, actCol: c.act,
		})
	}
	return out
}

func queryGFKSteps(ctx context.Context, pg *pgx.Conn, t gfkTable, contentTypeID int64) []djangoStep {
	when := "NOW()"
	if t.timeCol != "" {
		when = quoteIdent(t.timeCol)
	}
	act := "''"
	if t.actCol != "" {
		act = "COALESCE(" + quoteIdent(t.actCol) + "::text, '')"
	}
	q := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s
		FROM %s
		WHERE %s = $1
		ORDER BY 1, 5`,
		quoteIdent(t.objectCol), userExpr(t.userCol), commentExpr(t.commentCol),
		act, when, quoteIdent(t.name), quoteIdent(t.ctCol))
	rows, err := pg.Query(ctx, q, contentTypeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []djangoStep
	for rows.Next() {
		var s djangoStep
		var actName string
		if err := rows.Scan(&s.objectID, &s.userID, &s.comment, &actName, &s.actedAt); err != nil {
			continue
		}
		s.actName = actName
		s.actType = actName
		out = append(out, s)
	}
	return out
}

func userExpr(col string) string {
	if strings.TrimSpace(col) == "" {
		return "0"
	}
	return "COALESCE(" + quoteIdent(col) + ", 0)"
}

func commentExpr(col string) string {
	if strings.TrimSpace(col) == "" {
		return "''"
	}
	return "COALESCE(" + quoteIdent(col) + "::text, '')"
}
