package attachment

import "dfms/pkg/types"

// Entity is the owning document for list / upload / download / delete.
type Entity struct {
	ID             uint
	UID            string
	DocumentNumber string
	CanMutate      bool
}

// CanMutateStatus is true for draft and returned (amend after soft reject).
func CanMutateStatus(status string) bool {
	return status == string(types.DocDraft) || status == string(types.DocReturned) ||
		status == string(types.OrderDraft) || status == string(types.ReceiptDraft)
}
