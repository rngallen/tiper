package types

// DocumentStatus is the approval / posting lifecycle shared by billing batches,
// ITT, zerolisation, change-of-service, and stock reversals.
type DocumentStatus string

const (
	DocDraft     DocumentStatus = "draft"
	DocSubmitted DocumentStatus = "submitted"
	DocApproved  DocumentStatus = "approved"
	DocReturned  DocumentStatus = "returned" // soft reject — amend and resubmit
	DocRejected  DocumentStatus = "rejected"
	DocPosted    DocumentStatus = "posted"
)

func (s DocumentStatus) Valid() bool {
	switch s {
	case DocDraft, DocSubmitted, DocApproved, DocReturned, DocRejected, DocPosted:
		return true
	}
	return false
}
