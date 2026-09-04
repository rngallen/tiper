package reports

import (
	"strings"

	"dfms/apps/models"
	"dfms/pkg/db"
	"dfms/pkg/export"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func usersReport(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	q := db.Db.Model(&models.User{}).Preload("Roles").Preload("Profile").
		Where("CreatedAt BETWEEN ? AND ?", search.FromDate, search.ToDate)
	if search.HasSearch() {
		like := search.Search
		q = q.Where("Email LIKE ? OR FirstName LIKE ? OR LastName LIKE ?", like, like, like)
	}
	if wantsExcel(c) {
		var rows []models.User
		if err := q.Order("Email ASC").Find(&rows).Error; err != nil {
			return response.InternalServerError(c)
		}
		data := make([][]any, 0, len(rows))
		for _, u := range rows {
			roles := make([]string, 0, len(u.Roles))
			for _, r := range u.Roles {
				roles = append(roles, r.Name)
			}
			last := ""
			if u.LastLogin != nil {
				last = u.LastLogin.Format("02/01/2006 15:04")
			}
			data = append(data, []any{len(data) + 1, u.Email, u.FullName(), yesNo(u.IsActive), yesNo(u.IsLocked), strings.Join(roles, ", "), last})
		}
		return export.Slice(c, "Users", "users", []any{"S/N", "Email", "Name", "Active", "Locked", "Roles", "Last login"}, data)
	}
	if wantsPDF(c) {
		var rows []models.User
		if err := q.Order("Email ASC").Find(&rows).Error; err != nil {
			return response.InternalServerError(c)
		}
		out := make([][]string, 0, len(rows))
		for _, u := range rows {
			roles := make([]string, 0, len(u.Roles))
			for _, r := range u.Roles {
				roles = append(roles, r.Name)
			}
			last := ""
			if u.LastLogin != nil {
				last = u.LastLogin.Format("02/01/2006 15:04")
			}
			active := "No"
			if u.IsActive {
				active = "Yes"
			}
			out = append(out, []string{u.Email, u.FullName(), active, strings.Join(roles, ", "), last})
		}
		return export.TablePDF(c, "System users", "users", []string{"S/N", "Email", "Name", "Active", "Roles", "Last login"}, withSerial(out))
	}
	var items []models.User
	result, err := response.NewPaginator(c, q.Order("Email ASC"), search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, result)
}

func rolesReport(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	q := db.Db.Model(&models.Role{}).Preload("Permissions")
	if search.HasSearch() {
		q = q.Where("Name LIKE ? OR Description LIKE ?", search.Search, search.Search)
	}
	if wantsExcel(c) || wantsPDF(c) {
		var rows []models.Role
		if err := q.Order("Name ASC").Find(&rows).Error; err != nil {
			return response.InternalServerError(c)
		}
		if wantsPDF(c) {
			out := make([][]string, 0, len(rows))
			for _, r := range rows {
				codes := make([]string, 0, len(r.Permissions))
				for _, p := range r.Permissions {
					codes = append(codes, p.Code)
				}
				out = append(out, []string{r.Name, r.Description, strings.Join(codes, ", ")})
			}
			return export.TablePDF(c, "Roles & permissions", "roles", []string{"S/N", "Role", "Description", "Permissions"}, withSerial(out))
		}
		data := make([][]any, 0, len(rows))
		for _, r := range rows {
			codes := make([]string, 0, len(r.Permissions))
			for _, p := range r.Permissions {
				codes = append(codes, p.Code)
			}
			data = append(data, []any{r.Name, r.Description, strings.Join(codes, ", ")})
		}
		return export.Slice(c, "Roles", "roles", []any{"Role", "Description", "Permissions"}, data)
	}
	var items []models.Role
	result, err := response.NewPaginator(c, q.Order("Name ASC"), search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, result)
}

func auditActivity(c fiber.Ctx) error {
	search, err := response.ParseSearchRequest(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	q := db.Db.Model(&models.AuditTrail{}).
		Where("CreatedAt BETWEEN ? AND ?", search.FromDate, search.ToDate)
	if search.HasSearch() {
		q = q.Where("UserName LIKE ? OR Description LIKE ?", search.Search, search.Search)
	}
	if module := strings.TrimSpace(c.Query("module")); module != "" {
		q = q.Where("Module = ?", module)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		q = q.Where("Action = ?", action)
	}
	if userName := strings.TrimSpace(c.Query("userName")); userName != "" {
		q = q.Where("UserName LIKE ?", "%"+userName+"%")
	}
	q = q.Order("CreatedAt DESC")
	if wantsExcel(c) {
		return export.Query(c, q, "Audit", "audit_activity",
			[]any{"When", "Actor", "Module", "Action", "Description", "IP"},
			func(e models.AuditTrail) []any {
				return []any{e.CreatedAt, e.UserName, e.Module, e.Action, e.Description, e.IPAddress}
			})
	}
	if wantsPDF(c) {
		var rows []models.AuditTrail
		if err := q.Limit(2000).Find(&rows).Error; err != nil {
			return response.InternalServerError(c)
		}
		out := make([][]string, 0, len(rows))
		for _, e := range rows {
			out = append(out, []string{
				e.CreatedAt.Format("02/01/2006 15:04"), e.UserName, string(e.Module),
				string(e.Action), e.Description, e.IPAddress,
			})
		}
		return export.TablePDF(c, "Audit activity", "audit_activity",
			[]string{"S/N", "When", "Actor", "Module", "Action", "Description", "IP"}, withSerial(out))
	}
	var items []models.AuditTrail
	result, err := response.NewPaginator(c, q, search, &items).Run()
	if err != nil {
		return response.BadRequest(c, err.Error())
	}
	return response.OkDetail(c, result)
}
