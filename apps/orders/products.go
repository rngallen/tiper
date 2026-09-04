package orders

import (
	"errors"
	"strings"

	"dfms/apps/models"
	"dfms/pkg/response"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

const (
	gantryAGO = "ago"
	gantryPMS = "pms"
)

// listILRProducts is GET /orders/loading-products — gantry grades only (AGO, PMS/MOGAS).
// exclude is a product UID already chosen as the main product; the pair is the other grade.
func (h handler) listILRProducts(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("search"))
	if q == "" {
		q = strings.TrimSpace(c.Query("q"))
	}
	exclude := strings.TrimSpace(firstNonEmpty(c.Query("exclude"), c.Query("pairOf")))
	tx := h.db.Where("IsActive = ?", true)
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("Name LIKE ? OR Code LIKE ?", like, like)
	}
	var rows []models.Product
	if err := tx.Order("Name").Find(&rows).Error; err != nil {
		return err
	}
	excludeGrade := ""
	if exclude != "" {
		var ex models.Product
		if h.db.Where("UID = ?", exclude).First(&ex).Error == nil {
			excludeGrade = gantryGradeOf(ex)
		}
	}
	out := make([]models.Product, 0, 2)
	for _, p := range rows {
		g := gantryGradeOf(p)
		if g == "" {
			continue
		}
		if excludeGrade != "" && g == excludeGrade {
			continue
		}
		out = append(out, p)
	}
	return response.OkDetail(c, out)
}

func gantryGradeOf(p models.Product) string {
	s := strings.ToUpper(strings.TrimSpace(p.Code + " " + p.Name))
	switch {
	case strings.Contains(s, "AGO"), strings.Contains(s, "GAS OIL"):
		return gantryAGO
	case strings.Contains(s, "PMS"), strings.Contains(s, "MOGAS"),
		strings.Contains(s, "MOTOR SPIRIT"), strings.Contains(s, "MOTOR GAS"):
		return gantryPMS
	default:
		return ""
	}
}

func validateGantryPair(db *gorm.DB, productID uint, byID *uint) error {
	if byID == nil {
		return nil
	}
	var main, by models.Product
	if db.First(&main, productID).Error != nil || db.First(&by, *byID).Error != nil {
		return nil
	}
	g1, g2 := gantryGradeOf(main), gantryGradeOf(by)
	if g1 != "" && g2 != "" && g1 == g2 {
		return errors.New("when the main product is AGO the by-product must be PMS, and vice versa")
	}
	return nil
}
