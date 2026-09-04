package workflow

import (
	"context"
	"errors"
	"strings"
	"time"

	"dfms/apps/models"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// TaskView is a flattened pending task for a user's inbox.
type TaskView struct {
	TaskUID        string            `json:"taskId" gorm:"column:TaskUID"`
	InstanceUID    string            `json:"instanceId" gorm:"column:InstanceUID"`
	No             string            `json:"no" gorm:"column:No"`
	Summary        string            `json:"summary" gorm:"column:Summary"`
	NodeName       string            `json:"nodeName" gorm:"column:NodeName"`
	NodeStatus     types.NodeStatus  `json:"nodeStatus" gorm:"column:NodeStatus"`
	DocContentType types.ContentType `json:"docContentType" gorm:"column:DocContentType"`
	ObjectID       uint              `json:"-" gorm:"column:ObjectID"`
	CreatedAt      time.Time         `json:"createdAt" gorm:"column:CreatedAt"`
	OnBehalfOfName string            `json:"onBehalfOfName,omitempty" gorm:"column:OnBehalfOfName"`

	DocumentNumber string `json:"documentNumber" gorm:"column:DocumentNumber"`
	Amount         string `json:"amount" gorm:"column:Amount"`
	CurrencyCode   string `json:"currencyCode" gorm:"column:CurrencyCode"`
	CustomerCode   string `json:"customerCode" gorm:"column:CustomerCode"`
	CustomerName   string `json:"customerName" gorm:"column:CustomerName"`
	DocUID         string `json:"docId" gorm:"column:DocUID"`
}

// TaskFilter scopes MyTasks / MyDecisions list queries.
type TaskFilter struct {
	Search          string              // matched against instance No / Summary
	DocContentType  types.ContentType   // 0 = all processes (unless DocContentTypes is set)
	DocContentTypes []types.ContentType // optional IN filter for mixed inboxes
	OrderBy         string
	SortDirection   string
}

// MyTasks returns a page of pending tasks assigned to a user across open
// instances: running approval steps and initiator amendment after a soft
// reject. Operators act from Approvals.
//
// COUNT skips document joins unless search or bank UID needs them; the list
// query uses idx_taskStatusUser (Status, UserID, CreatedAt) plus the substitute
// EXISTS on ApprovalSubstitute (seek idx_substituteDelegateProcess).
// OnBehalfOfName is the task owner's name when the viewer is covering them.
func (e *Engine) MyTasks(ctx context.Context, userID uint, page, pageSize int, filter TaskFilter) ([]TaskView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	now := time.Now().UTC()
	ct := joinDocContentType(filter)
	joins, joinArgs := documentJoins(ct)
	where := `
		WHERE t.Status = ?
			AND i.Status NOT IN (?, ?)
			AND (
				t.UserID = ?
				OR EXISTS (
					SELECT 1 FROM ApprovalSubstitute s
					WHERE s.PrincipalUserID = t.UserID
						AND s.ProcessID = i.ProcessID
						AND s.DelegateUserID = ?
						AND s.StartsAt <= ?
						AND (s.EndsAt IS NULL OR s.EndsAt >= ?)
						AND (s.NodeID IS NULL OR s.NodeID = t.NodeID)
				)
			)`
	whereArgs := []any{
		types.TaskPending, types.NodeCompleted, types.NodeRejected,
		userID, userID, now, now,
	}

	countJoins, countJoinArgs := joins, joinArgs
	if !taskFilterNeedsDocs(filter) {
		countJoins, countJoinArgs = "", nil
	}
	countBase := `
		FROM Task t
		JOIN ProcessInstance i ON i.ID = t.InstanceID
		` + countJoins + where
	countArgs := append(append([]any{}, countJoinArgs...), whereArgs...)
	countBase, countArgs = appendTaskFilters(countBase, countArgs, filter)

	var total int64
	if err := e.db.WithContext(ctx).Raw(`SELECT COUNT(*) `+countBase, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectBase := `
		FROM Task t
		JOIN ProcessInstance i ON i.ID = t.InstanceID
		JOIN Node n ON n.ID = t.NodeID
		LEFT JOIN [User] principal ON principal.ID = t.UserID
		` + joins + where
	selectWhereArgs := append(append([]any{}, joinArgs...), whereArgs...)
	selectBase, selectWhereArgs = appendTaskFilters(selectBase, selectWhereArgs, filter)

	selectArgs := make([]any, 0, len(selectWhereArgs)+3)
	selectArgs = append(selectArgs, userID)
	selectArgs = append(selectArgs, selectWhereArgs...)
	selectArgs = append(selectArgs, offset, pageSize)
	var out []TaskView
	err := e.db.WithContext(ctx).Raw(`
		SELECT t.UID AS TaskUID, i.UID AS InstanceUID, i.No AS No, i.Summary AS Summary,
			n.Name AS NodeName, n.Status AS NodeStatus,
			i.DocContentType AS DocContentType, i.ObjectID AS ObjectID, t.CreatedAt AS CreatedAt,
			CASE WHEN t.UserID = ? OR principal.ID IS NULL THEN ''
				ELSE CONCAT(principal.FirstName, ' ', principal.LastName)
			END AS OnBehalfOfName,
			`+documentSelectColumns(ct)+`
		`+selectBase+`
		ORDER BY `+taskOrderSQL(filter.OrderBy, filter.SortDirection, "t.CreatedAt", "t.ID", true, ct)+`
		OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`,
		selectArgs...).Scan(&out).Error
	return out, total, err
}

// DecisionView is a past agree/reject action taken by the signed-in user.
type DecisionView struct {
	InstanceUID    string            `json:"instanceId" gorm:"column:InstanceUID"`
	No             string            `json:"no" gorm:"column:No"`
	Summary        string            `json:"summary" gorm:"column:Summary"`
	ActType        types.EventAction `json:"actType" gorm:"column:ActType"`
	ActName        string            `json:"actName" gorm:"column:ActName"`
	Comment        string            `json:"comment" gorm:"column:Comment"`
	TotalRejection bool              `json:"totalRejection" gorm:"column:TotalRejection"`
	FromNode       *string           `json:"fromNode" gorm:"column:FromNode"`
	ToNode         *string           `json:"toNode" gorm:"column:ToNode"`
	InstanceStatus types.NodeStatus  `json:"instanceStatus" gorm:"column:InstanceStatus"`
	DocContentType types.ContentType `json:"docContentType" gorm:"column:DocContentType"`
	CreatedAt      time.Time         `json:"createdAt" gorm:"column:CreatedAt"`

	DocumentNumber string `json:"documentNumber" gorm:"column:DocumentNumber"`
	Amount         string `json:"amount" gorm:"column:Amount"`
	CurrencyCode   string `json:"currencyCode" gorm:"column:CurrencyCode"`
	CustomerCode   string `json:"customerCode" gorm:"column:CustomerCode"`
	CustomerName   string `json:"customerName" gorm:"column:CustomerName"`
	DocUID         string `json:"docId" gorm:"column:DocUID"`
}

// MyDecisions returns a page of agree or reject events recorded by the user
// (newest first). Uses idx_eventUserActCreated (UserID, ActType, CreatedAt).
func (e *Engine) MyDecisions(ctx context.Context, userID uint, act types.EventAction, page, pageSize int, filter TaskFilter) ([]DecisionView, int64, error) {
	switch act {
	case types.EventAgree, types.EventReject:
	default:
		return nil, 0, errors.New("act must be agree or reject")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	ct := joinDocContentType(filter)
	joins, joinArgs := documentJoins(ct)
	// [Event] — SQL Server can treat Event as a special identifier.
	agreeFilter := ""
	agreeArgs := []any{}
	if act == types.EventAgree {
		// Initiator resubmit is ActAgree from the draft node — not an approval.
		agreeFilter = ` AND ono.Status <> ? AND ev.ActName <> ?`
		agreeArgs = []any{types.NodeDraft, "Resubmitted"}
	}

	countJoins, countJoinArgs := joins, joinArgs
	if !taskFilterNeedsDocs(filter) {
		countJoins, countJoinArgs = "", nil
	}
	countBase := `
		FROM [Event] ev
		JOIN ProcessInstance i ON i.ID = ev.InstanceID
		LEFT JOIN Node ono ON ono.ID = ev.OldNodeID
		` + countJoins + `
		WHERE ev.UserID = ? AND ev.ActType = ?` + agreeFilter
	countArgs := append(append([]any{}, countJoinArgs...), userID, act)
	countArgs = append(countArgs, agreeArgs...)
	countBase, countArgs = appendTaskFilters(countBase, countArgs, filter)

	var total int64
	if err := e.db.WithContext(ctx).Raw(`SELECT COUNT(*) `+countBase, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	selectBase := `
		FROM [Event] ev
		JOIN ProcessInstance i ON i.ID = ev.InstanceID
		LEFT JOIN Node ono ON ono.ID = ev.OldNodeID
		LEFT JOIN Node nno ON nno.ID = ev.NewNodeID
		` + joins + `
		WHERE ev.UserID = ? AND ev.ActType = ?` + agreeFilter
	selectArgs := append(append([]any{}, joinArgs...), userID, act)
	selectArgs = append(selectArgs, agreeArgs...)
	selectBase, selectArgs = appendTaskFilters(selectBase, selectArgs, filter)

	var out []DecisionView
	err := e.db.WithContext(ctx).Raw(`
		SELECT i.UID AS InstanceUID, i.No AS No, i.Summary AS Summary,
			ev.ActType AS ActType, ev.ActName AS ActName, ev.Comment AS Comment,
			ev.TotalRejection AS TotalRejection,
			ono.Name AS FromNode, nno.Name AS ToNode,
			i.Status AS InstanceStatus, i.DocContentType AS DocContentType, ev.CreatedAt AS CreatedAt,
			`+documentSelectColumns(ct)+`
		`+selectBase+`
		ORDER BY `+taskOrderSQL(filter.OrderBy, filter.SortDirection, "ev.CreatedAt", "ev.ID", false, ct)+`
		OFFSET ? ROWS FETCH NEXT ? ROWS ONLY`,
		append(selectArgs, offset, pageSize)...).Scan(&out).Error
	return out, total, err
}

func taskFilterNeedsDocs(filter TaskFilter) bool {
	return strings.TrimSpace(filter.Search) != ""
}

// documentJoins attaches source documents so inbox rows show a human number
// and public UID (attachments / open-document). Mixed inboxes LEFT JOIN every
// approval kind; typed queues still use the same SQL (SQL Server elides unused
// joins cheaply vs maintaining per-type strings).
func documentJoins(types.ContentType) (string, []any) {
	return `
		LEFT JOIN Receipt r ON r.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN BillingRun br ON br.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer cust ON cust.ID = br.CustomerID
		LEFT JOIN ChangeOfService cos ON cos.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer cosCust ON cosCust.ID = cos.CustomerID
		LEFT JOIN ExchangeRate fx ON fx.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN FcfFeeBatch fcf ON fcf.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN VariableFeeBatch vcf ON vcf.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN KojFeeBatch koj ON koj.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN TbsFeeBatch tbs ON tbs.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN MiLossBatch mi ON mi.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN GantryLoadingRequest glr ON glr.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer glrCust ON glrCust.ID = glr.CustomerID
		LEFT JOIN GantryCompartmentalization comp ON comp.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer compCust ON compCust.ID = comp.CustomerID
		LEFT JOIN PumpOverRequest pdo ON pdo.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer pdoCust ON pdoCust.ID = pdo.CustomerID
		LEFT JOIN PumpOverReport por ON por.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN IttTransfer itt ON itt.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer ittCust ON ittCust.ID = itt.FromCustomerID
		LEFT JOIN ZerolizationTransfer zerol ON zerol.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN Customer zerolCust ON zerolCust.ID = zerol.CustomerID
		LEFT JOIN FinancialHoldRelease hold ON hold.ID = i.ObjectID AND i.DocContentType = ?
		LEFT JOIN OrderAmendment amd ON amd.ID = i.ObjectID AND i.DocContentType = ?`, []any{
		types.ReceiptContent, types.BillingRunContent, types.ChangeOfServiceContent, types.ExchangeRateContent,
		types.BillingProfileContent, types.VariableFeeBatchContent, types.KojFeeBatchContent,
		types.TbsFeeBatchContent, types.MiLossBatchContent, types.GantryLoadingRequestContent,
		types.CompartmentalizationContent, types.PumpOverRequestContent, types.PumpOverReportContent,
		types.IttTransferContent, types.ZerolizationContent, types.FinancialHoldContent,
		types.OrderAmendmentContent,
	}
}

func documentNumberSQL() string {
	return `COALESCE(
			NULLIF(r.DocumentNumber, ''),
			NULLIF(br.DocumentNumber, ''),
			NULLIF(cos.DocumentNumber, ''),
			NULLIF(fcf.DocumentNumber, ''),
			NULLIF(vcf.DocumentNumber, ''),
			NULLIF(koj.DocumentNumber, ''),
			NULLIF(tbs.DocumentNumber, ''),
			NULLIF(mi.DocumentNumber, ''),
			NULLIF(glr.DocumentNumber, ''),
			NULLIF(comp.DocumentNumber, ''),
			NULLIF(pdo.DocumentNumber, ''),
			NULLIF(por.DocumentNumber, ''),
			NULLIF(itt.DocumentNumber, ''),
			NULLIF(zerol.DocumentNumber, ''),
			NULLIF(hold.DocumentNumber, ''),
			NULLIF(amd.DocumentNumber, ''),
			NULLIF(CASE WHEN fx.ID IS NULL THEN ''
				ELSE fx.FromCurrency + '/' + fx.ToCurrency + ' · ' + CONVERT(varchar(10), fx.EffectiveFrom, 23)
			END, ''),
			''
		)`
}

func documentSelectColumns(types.ContentType) string {
	return `
		` + documentNumberSQL() + ` AS DocumentNumber,
		COALESCE(CONVERT(VARCHAR(40), br.Amount), '') AS Amount,
		COALESCE(br.CurrencyCode, '') AS CurrencyCode,
		COALESCE(cust.Code, cosCust.Code, glrCust.Code, compCust.Code, pdoCust.Code, ittCust.Code, zerolCust.Code, '') AS CustomerCode,
		COALESCE(cust.Name, cosCust.Name, glrCust.Name, compCust.Name, pdoCust.Name, ittCust.Name, zerolCust.Name, i.Summary, '') AS CustomerName,
		COALESCE(r.UID, br.UID, cos.UID, fx.UID, fcf.UID, vcf.UID, koj.UID, tbs.UID, mi.UID,
			glr.UID, comp.UID, pdo.UID, por.UID, itt.UID, zerol.UID, hold.UID, amd.UID, '') AS DocUID`
}

func taskOrderSQL(orderBy, dir, createdCol, idCol string, forTasks bool, ct types.ContentType) string {
	d := strings.ToUpper(strings.TrimSpace(dir))
	if d != "ASC" && d != "DESC" {
		if createdCol == "ev.CreatedAt" {
			d = "DESC"
		} else {
			d = "ASC"
		}
	}
	customerName := `COALESCE(cust.Name, cosCust.Name, glrCust.Name, compCust.Name, pdoCust.Name, ittCust.Name, zerolCust.Name, i.Summary)`
	customerCode := `COALESCE(cust.Code, cosCust.Code, glrCust.Code, compCust.Code, pdoCust.Code, ittCust.Code, zerolCust.Code, '')`
	currency := `COALESCE(br.CurrencyCode, '')`
	allowed := map[string]string{
		"customerName":   customerName,
		"customerCode":   customerCode,
		"createdAt":      createdCol,
		"currencyCode":   currency,
		"amount":         `br.Amount`,
		"documentNumber": `COALESCE(NULLIF(` + documentNumberSQL() + `, ''), i.No)`,
	}
	if forTasks {
		allowed["nodeName"] = `n.Name`
	}
	col, ok := allowed[strings.TrimSpace(orderBy)]
	if !ok {
		col = createdCol
	}
	return col + " " + d + ", " + idCol + " ASC"
}

func joinDocContentType(filter TaskFilter) types.ContentType {
	if len(filter.DocContentTypes) > 1 {
		return 0
	}
	if len(filter.DocContentTypes) == 1 {
		return filter.DocContentTypes[0]
	}
	return filter.DocContentType
}

func appendTaskFilters(base string, args []any, filter TaskFilter) (string, []any) {
	if n := len(filter.DocContentTypes); n > 0 {
		ph := strings.Repeat("?,", n)
		base += ` AND i.DocContentType IN (` + ph[:len(ph)-1] + `)`
		for _, ct := range filter.DocContentTypes {
			args = append(args, ct)
		}
	} else if filter.DocContentType != 0 {
		base += ` AND i.DocContentType = ?`
		args = append(args, filter.DocContentType)
	}
	if q := strings.TrimSpace(filter.Search); q != "" {
		like := "%" + q + "%"
		base += ` AND (
			i.No LIKE ? OR i.Summary LIKE ?
			OR r.DocumentNumber LIKE ? OR br.DocumentNumber LIKE ?
			OR fcf.DocumentNumber LIKE ? OR vcf.DocumentNumber LIKE ?
			OR koj.DocumentNumber LIKE ? OR tbs.DocumentNumber LIKE ?
			OR mi.DocumentNumber LIKE ? OR glr.DocumentNumber LIKE ?
			OR comp.DocumentNumber LIKE ? OR pdo.DocumentNumber LIKE ?
			OR por.DocumentNumber LIKE ? OR itt.DocumentNumber LIKE ?
			OR zerol.DocumentNumber LIKE ? OR hold.DocumentNumber LIKE ?
			OR amd.DocumentNumber LIKE ?
			OR cust.Code LIKE ? OR cust.Name LIKE ?
			OR glrCust.Code LIKE ? OR glrCust.Name LIKE ?
			OR fx.FromCurrency + '/' + fx.ToCurrency LIKE ?
		)`
		args = append(args, like, like, like, like, like, like, like, like, like, like,
			like, like, like, like, like, like, like, like, like, like, like, like)
	}
	return base, args
}

// EventView is a flattened history entry.
type EventView struct {
	ActType         types.EventAction `json:"actType"`
	ActName         string            `json:"actName"`
	Comment         string            `json:"comment"`
	TotalRejection  bool              `json:"totalRejection"`
	OldNode         string            `json:"oldNode,omitempty"`
	NewNode         string            `json:"newNode,omitempty"`
	UserEmail       string            `json:"userEmail,omitempty"`
	UserName        string            `json:"userName,omitempty"`
	UserTitle       string            `json:"userTitle,omitempty"`
	OnBehalfOfName  string            `json:"onBehalfOfName,omitempty"`
	OnBehalfOfEmail string            `json:"onBehalfOfEmail,omitempty"`
	IPAddress       string            `json:"ipAddress,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
}

// Progress returns the ordered history of a workflow instance, enriched with
// the approver's name, job title and IP for the audit trail.
//
// Two queries only (instance lookup + joined trail) — avoids N+1 node/user loads.
func (e *Engine) Progress(ctx context.Context, instanceUID string) ([]EventView, error) {
	db := e.db.WithContext(ctx)

	var inst models.ProcessInstance
	if err := db.Select("ID").Where("UID = ?", instanceUID).Take(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, err
	}

	type trailRow struct {
		ActType         types.EventAction
		ActName         string
		Comment         string
		TotalRejection  bool
		IPAddress       string
		CreatedAt       time.Time
		OldNode         string
		NewNode         string
		UserEmail       string
		UserName        string
		UserTitle       string
		OnBehalfOfEmail string
		OnBehalfOfName  string
	}
	var rows []trailRow
	// [Event] / [User] — SQL Server reserved identifiers.
	if err := db.Raw(`
		SELECT ev.ActType, ev.ActName, ev.Comment, ev.TotalRejection, ev.IPAddress, ev.CreatedAt,
			COALESCE(ono.Name, '') AS OldNode, COALESCE(nno.Name, '') AS NewNode,
			COALESCE(u.Email, '') AS UserEmail,
			CASE WHEN u.ID IS NULL THEN '' ELSE CONCAT(u.FirstName, ' ', u.LastName) END AS UserName,
			COALESCE(p.Title, '') AS UserTitle,
			COALESCE(ob.Email, '') AS OnBehalfOfEmail,
			CASE WHEN ob.ID IS NULL THEN '' ELSE CONCAT(ob.FirstName, ' ', ob.LastName) END AS OnBehalfOfName
		FROM [Event] ev
		LEFT JOIN Node ono ON ono.ID = ev.OldNodeID
		LEFT JOIN Node nno ON nno.ID = ev.NewNodeID
		LEFT JOIN [User] u ON u.ID = ev.UserID
		LEFT JOIN Profile p ON p.UserID = u.ID
		LEFT JOIN [User] ob ON ob.ID = ev.OnBehalfOfUserID
		WHERE ev.InstanceID = ?
		ORDER BY ev.ID ASC`, inst.ID).Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]EventView, 0, len(rows))
	for _, r := range rows {
		out = append(out, EventView{
			ActType:         r.ActType,
			ActName:         r.ActName,
			Comment:         r.Comment,
			TotalRejection:  r.TotalRejection,
			OldNode:         r.OldNode,
			NewNode:         r.NewNode,
			UserEmail:       r.UserEmail,
			UserName:        strings.TrimSpace(r.UserName),
			UserTitle:       r.UserTitle,
			OnBehalfOfEmail: r.OnBehalfOfEmail,
			OnBehalfOfName:  strings.TrimSpace(r.OnBehalfOfName),
			IPAddress:       r.IPAddress,
			CreatedAt:       r.CreatedAt,
		})
	}
	return out, nil
}

// ListProcesses returns all process definitions with nodes and transitions.
func (e *Engine) ListProcesses(ctx context.Context) ([]models.Process, error) {
	var procs []models.Process
	err := e.db.WithContext(ctx).
		Preload("Nodes").Preload("Transitions").
		Order("ID ASC").Find(&procs).Error
	return procs, err
}
