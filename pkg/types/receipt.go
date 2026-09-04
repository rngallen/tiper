package types

// ReceiptType distinguishes in-house vs third-party vessel receipts.
type ReceiptType string

const (
	ReceiptInternal ReceiptType = "internal"
	ReceiptExternal ReceiptType = "external"
)

func (t ReceiptType) Valid() bool {
	return t == ReceiptInternal || t == ReceiptExternal
}

// PostsStock is true when the cargo lands in TIPER tanks (internal only).
func (t ReceiptType) PostsStock() bool { return t == ReceiptInternal }

// BillsStorage is true for FCF / VCF / TBS first-cycle (internal only).
func (t ReceiptType) BillsStorage() bool { return t == ReceiptInternal }

// ReceiptStatus is the vessel-receipt workflow. Values match DocumentStatus
// (draft → submitted → approved | rejected).
type ReceiptStatus string

const (
	ReceiptDraft     ReceiptStatus = "draft"
	ReceiptSubmitted ReceiptStatus = "submitted"
	ReceiptApproved  ReceiptStatus = "approved"
	ReceiptRejected  ReceiptStatus = "rejected"
)

func (s ReceiptStatus) Valid() bool {
	switch s {
	case ReceiptDraft, ReceiptSubmitted, ReceiptApproved, ReceiptRejected:
		return true
	}
	return false
}
