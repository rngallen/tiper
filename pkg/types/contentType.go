// Package types holds shared value types and enumerations.
package types

// ContentType identifies the kind of entity a generic row refers to (attachments,
// audit trail, refresh tokens, workflow). Auth/setup IDs 1–23 are packed (no
// empty slots). Domain types from 40 keep historical numbers so the SPA and
// GORM checks stay stable across remigrate. 71 is change of service.
type ContentType uint8

const (
	RefreshTokenContent          ContentType = iota + 1 // 1
	UserProfileContent                                  // 2
	UserContent                                         // 3
	PermissionContent                                   // 4
	RoleContent                                         // 5
	TitleContent                                        // 6
	CompanyContent                                      // 7
	AttachmentContent                                   // 8
	IntegrationSettingContent                           // 9
	OTPChallengeContent                                 // 10
	AuditTrailContent                                   // 11
	ProcessContent                                      // 12
	NodeContent                                         // 13
	TransitionContent                                   // 14
	ProcessInstanceContent                              // 15
	TaskContent                                         // 16
	EventContent                                        // 17
	InitiatorPoolContent                                // 18
	SubstituteContent                                   // 19
	CurrencyContent                                     // 20
	DocumentNumberCounterContent                        // 21
	CountryContent                                      // 22
	PasswordHistoryContent                              // 23
)

// DFMS domain content types start at 40.
const (
	StockCategoryContent            ContentType = iota + 40 // 40
	ProductContent                                          // 41
	StockStatusContent                                      // 42
	UnitOfMeasureContent                                    // 43
	TankContent                                             // 44
	VesselContent                                           // 45
	DepotContent                                            // 46
	CustomerContent                                         // 47
	CustomerBillingAccountContent                           // 48
	SupplierContent                                         // 49
	SupplierBillingAccountContent                           // 50
	ReceiptContent                                          // 51
	ReceiptDetailContent                                    // 52
	FinancialHoldContent                                    // 53
	StockMovementContent                                    // 54
	IttTransferContent                                      // 55
	ZerolizationContent                                     // 56
	InventoryEventContent                                   // 57
	_                                                       // 58 retired stock reversal
	BillingCycleContent                                     // 59
	FeeContent                                              // 60
	BillingProfileContent                                   // 61 FCF fee batch
	MiLossBatchContent                                      // 62
	VariableFeeBatchContent                                 // 63
	KojFeeBatchContent                                      // 64
	TbsFeeBatchContent                                      // 65
	BillingRunContent                                       // 66
	ChargeLineContent                                       // 67
	EwuraLicenseContent                                     // 68
	PhysicalDipContent                                      // 69
	ReportSnapshotContent                                   // 70
	ChangeOfServiceContent                                  // 71 customer parcel delivery-method switch
	StockReservationContent                                 // 72
	BillingExceptionContent                                 // 73
	_                                                       // 74 retired decimal-precision singleton
	LookupContent                                           // 75
	SagePostingLogContent                                   // 76
	MailOutboxContent                                       // 77
	ExchangeRateContent                                     // 78
	LineContentContent                                      // 79
	GantryLoadingRequestContent                             // 80
	GantryLoadingLineContent                                // 81
	PumpOverRequestContent                                  // 82
	PumpOverReportContent                                   // 83
	TransporterContent                                      // 84
	DriverContent                                           // 85
	TruckContent                                            // 86
	DestinationContent                                      // 87
	DistrictContent                                         // 88
	TruckTankContent                                        // 89
	TankCalibrationContent                                  // 90
	TankCompartmentContent                                  // 91
	RfidBadgeContent                                        // 92
	CompartmentalizationContent                             // 93
	CompartmentalizationLineContent                         // 94
	GantryLoadingContent                                    // 95
	OrderAmendmentContent                                   // 96
	_                                                       // 97 retired amendment batch
	AlmaFileLogContent                                      // 98
	NpgisSubmissionContent                                  // 99
	StockBalanceContent                                     // 100
	StockDailyPositionContent                               // 101
	ProductDailyBalanceContent                              // 102
	GantryLoadingSummaryContent                             // 103
	GantryVesselLoadingContent                              // 104
	ReceptionFactContent                                    // 105
)

// ApprovalInboxPath is the SPA queue for a document content type.
func ApprovalInboxPath(ct ContentType) string {
	switch ct {
	case ReceiptContent:
		return "/approvals/receipts"
	case GantryLoadingRequestContent:
		return "/approvals/gantry"
	case CompartmentalizationContent:
		return "/approvals/compartmentalization"
	case OrderAmendmentContent:
		return "/approvals/amendments"
	case PumpOverRequestContent:
		return "/approvals/pump-over"
	case PumpOverReportContent:
		return "/approvals/pump-over-reports"
	case IttTransferContent:
		return "/approvals/itt"
	case ZerolizationContent:
		return "/approvals/zerolization"
	case FinancialHoldContent:
		return "/approvals/financial-hold"
	case BillingRunContent:
		return "/approvals/billing-runs"
	case MiLossBatchContent:
		return "/approvals/mi-loss"
	case BillingProfileContent:
		return "/approvals/fsf-fees"
	case ExchangeRateContent:
		return "/approvals/fx-rates"
	case VariableFeeBatchContent:
		return "/approvals/vsf-fees"
	case KojFeeBatchContent:
		return "/approvals/koj-fees"
	case TbsFeeBatchContent:
		return "/approvals/tbs-fees"
	case ChangeOfServiceContent:
		return "/approvals/change-of-service"
	default:
		return "/approvals"
	}
}

// ContentTypeLabel is the operator-facing name used in approvals and mail.
func ContentTypeLabel(ct ContentType) string {
	switch ct {
	case ReceiptContent:
		return "Vessel receipt"
	case FinancialHoldContent:
		return "Financial hold release"
	case BillingProfileContent:
		return "Fixed storage fee"
	case ExchangeRateContent:
		return "Exchange rate"
	case MiLossBatchContent:
		return "MI loss"
	case VariableFeeBatchContent:
		return "Variable storage fee"
	case KojFeeBatchContent:
		return "KOJ fee"
	case TbsFeeBatchContent:
		return "TBS fee"
	case BillingRunContent:
		return "Billing run"
	case ChangeOfServiceContent:
		return "Change of service"
	case ZerolizationContent:
		return "Zerolization"
	case GantryLoadingRequestContent:
		return "Internal loading request"
	case GantryLoadingLineContent:
		return "Internal loading order"
	case PumpOverRequestContent:
		return "Pump-over request"
	case PumpOverReportContent:
		return "Pump-over report"
	case IttTransferContent:
		return "In-tank transfer"
	case CompartmentalizationContent:
		return "Compartmentalization"
	case OrderAmendmentContent:
		return "Order amendment"
	case ReceptionFactContent:
		return "Reception fact"
	case MailOutboxContent:
		return "Notification"
	case CustomerContent:
		return "Customer"
	case SupplierContent:
		return "Supplier"
	case TankContent:
		return "Tank"
	case DriverContent:
		return "Driver"
	case TruckContent:
		return "Truck"
	case TransporterContent:
		return "Hauler"
	case EwuraLicenseContent:
		return "EWURA licence"
	default:
		return "Document"
	}
}

// ContentTypeFolder is the on-disk attachment category under Settings → Attachments
// ({root}/{folder}/YYYY/MM/{document-number-or-code}). Each attachable entity has
// its own folder — never a catch-all like "Other".
func ContentTypeFolder(ct ContentType) string {
	switch ct {
	case ReceiptContent:
		return "Receipts"
	case CustomerContent:
		return "Customers"
	case SupplierContent:
		return "Suppliers"
	case GantryLoadingRequestContent:
		return "ILR"
	case GantryLoadingLineContent:
		return "ILO"
	case IttTransferContent:
		return "ITT"
	case PumpOverRequestContent:
		return "Pump-over-request"
	case PumpOverReportContent:
		return "Pump-over-report"
	case CompartmentalizationContent:
		return "Compartmentalization"
	case OrderAmendmentContent:
		return "Amendments"
	case ChangeOfServiceContent:
		return "Change-of-service"
	case TankContent:
		return "Tanks"
	case DriverContent:
		return "Drivers"
	case TruckContent:
		return "Trucks"
	case TransporterContent:
		return "Haulers"
	case EwuraLicenseContent:
		return "EWURA-licences"
	case FinancialHoldContent:
		return "Financial-hold"
	case ZerolizationContent:
		return "Zerolization"
	case BillingRunContent:
		return "Billing"
	case BillingProfileContent:
		return "Fixed-storage-fees"
	case VariableFeeBatchContent:
		return "Variable-storage-fees"
	case KojFeeBatchContent:
		return "KOJ-fees"
	case TbsFeeBatchContent:
		return "TBS-fees"
	case MiLossBatchContent:
		return "MI-loss"
	case ExchangeRateContent:
		return "Exchange-rates"
	default:
		return ""
	}
}
