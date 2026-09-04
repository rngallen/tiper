package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ApprovalStep is one immutable trail row printed on a document.
// Native DFMS docs resolve this from Event; Django copies store it on the row.
type ApprovalStep struct {
	ActedAt  time.Time `json:"actedAt"`
	ActType  string    `json:"actType"`
	ActName  string    `json:"actName"`
	UserName string    `json:"userName"`
	Title    string    `json:"title"`
	Comment  string    `json:"comment"`
}

// ApprovalTrail is a JSON array stored on migrated documents (nvarchar(max)).
type ApprovalTrail []ApprovalStep

func (t ApprovalTrail) Value() (driver.Value, error) {
	if t == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]ApprovalStep(t))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (t *ApprovalTrail) Scan(src any) error {
	if t == nil {
		return fmt.Errorf("ApprovalTrail: Scan on nil pointer")
	}
	if src == nil {
		*t = ApprovalTrail{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("ApprovalTrail: unsupported scan type %T", src)
	}
	if len(b) == 0 {
		*t = ApprovalTrail{}
		return nil
	}
	var out []ApprovalStep
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	if out == nil {
		out = []ApprovalStep{}
	}
	*t = ApprovalTrail(out)
	return nil
}

func (t ApprovalTrail) AsILR() []ILRApproval {
	out := make([]ILRApproval, 0, len(t))
	for _, s := range t {
		out = append(out, ILRApproval{
			ApprovedOn: s.ActedAt.Format("02/01/2006 15:04"),
			ApprovedBy: s.UserName,
			Title:      s.Title,
			Comment:    s.Comment,
			ActType:    s.ActType,
			ActName:    s.ActName,
		})
	}
	return out
}
