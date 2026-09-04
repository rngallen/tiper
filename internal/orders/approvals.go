package orders

import (
	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// DocumentApprovals is the reprint trail for one document.
// Native DFMS rows have a ProcessInstance — those always use live Event rows
// (same pattern as the payment portal). Django copies have no instance; they
// use the immutable ApprovalTrail snapshot written at copy time.
func DocumentApprovals(db *gorm.DB, ct types.ContentType, objectID uint, snapshot models.ApprovalTrail) models.ApprovalTrail {
	if db == nil || objectID == 0 {
		return snapshot
	}
	if hasProcessInstance(db, ct, objectID) {
		return liveApprovals(db, ct, objectID)
	}
	return snapshot
}

func hasProcessInstance(db *gorm.DB, ct types.ContentType, objectID uint) bool {
	var n int64
	_ = db.Model(&models.ProcessInstance{}).
		Where("DocContentType = ? AND ObjectID = ?", ct, objectID).
		Limit(1).Count(&n).Error
	return n > 0
}

func liveApprovals(db *gorm.DB, ct types.ContentType, objectID uint) models.ApprovalTrail {
	var steps []models.ApprovalStep
	_ = db.Raw(`
		SELECT ev.CreatedAt AS ActedAt, ev.ActType, ev.ActName,
			CASE WHEN u.ID IS NULL THEN '' ELSE CONCAT(u.FirstName, ' ', u.LastName) END AS UserName,
			COALESCE(p.Title, '') AS Title, ev.Comment
		FROM [Event] ev
		JOIN ProcessInstance i ON i.ID = ev.InstanceID
		LEFT JOIN [User] u ON u.ID = ev.UserID
		LEFT JOIN Profile p ON p.UserID = u.ID
		WHERE i.DocContentType = ? AND i.ObjectID = ?
		ORDER BY ev.ID ASC`, ct, objectID).Scan(&steps).Error
	return steps
}
