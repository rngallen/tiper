// Package schema lists GORM models for schema creation and AutoMigrate.
package schema

import "dfms/apps/models"

// Group is a labelled bundle of models (order matters for foreign keys).
type Group struct {
	Label  string
	Models []any
}

// Groups returns the model groups in dependency order.
func Groups() []Group {
	return []Group{
		{Label: "auth", Models: []any{
			&models.User{}, &models.Profile{}, &models.Role{}, &models.Permission{},
			&models.UserRole{}, &models.RolesPermission{},
			&models.RefreshToken{}, &models.Title{},
			&models.UserOTPChallenge{}, &models.PasswordHistory{},
		}},
		{Label: "setup", Models: []any{
			&models.Currency{}, &models.Country{}, &models.Company{},
			&models.DocumentNumberCounter{}, &models.IntegrationSetting{},
		}},
		{Label: "master", Models: []any{
			&models.UnitOfMeasure{}, &models.StockCategory{}, &models.Product{},
			&models.StockStatus{}, &models.Vessel{},
			&models.BillingCycle{}, &models.ImportTenderType{}, &models.DeliveryMethod{},
			&models.ProcurementMethod{}, &models.DischargeRoute{}, &models.ContractType{},
			&models.PricingNature{},
			&models.Fee{},
			&models.Customer{}, &models.CustomerBillingAccount{},
			&models.Supplier{}, &models.SupplierBillingAccount{},
			&models.SageAccountOwner{},
			&models.Depot{}, &models.Tank{},
			&models.EwuraPetroleumLicense{},
			&models.Transporter{}, &models.Driver{}, &models.Truck{},
			&models.Destination{}, &models.District{},
			&models.TruckTank{}, &models.TankCalibration{}, &models.TankCompartment{},
			&models.RfidBadge{},
		}},
		{Label: "orders", Models: []any{
			&models.GantryLoadingRequest{}, &models.GantryRequestVessel{}, &models.GantryLoadingLine{},
			&models.GantryStockPosition{}, &models.GantryCustomerOutstanding{},
			&models.GantryOutstandingCharge{},
			&models.PumpOverRequest{}, &models.PumpOverVessel{}, &models.PumpOverReport{},
			&models.Compartmentalization{}, &models.CompartmentalizationLine{},
			&models.GantryLoading{}, &models.GantryLoadingProduct{},
			&models.GantryLoadingSummary{}, &models.GantryVesselLoading{},
			&models.OrderAmendment{},
		}},
		{Label: "inventory", Models: []any{
			&models.Receipt{}, &models.ReceiptDetail{}, &models.ReceptionFact{},
			&models.StockMovement{}, &models.IttTransfer{},
			&models.ZerolizationTransfer{}, &models.FinancialHoldRelease{}, &models.FinancialHoldReleaseLine{},
			&models.InventoryEventLog{},
			&models.StockReservation{},
			&models.PhysicalDip{}, &models.LineContent{},
			&models.StockBalance{}, &models.StockDailyPosition{}, &models.ProductDailyBalance{},
		}},
		{Label: "billing", Models: []any{
			&models.FcfFeeBatch{}, &models.FcfFee{}, &models.FcfFeeTier{},
			&models.ExchangeRate{}, &models.MiLossBatch{}, &models.MiLossProduct{}, &models.MiLoss{},
			&models.VariableFeeBatch{}, &models.ProductConfig{}, &models.ProductContractRate{},
			&models.KojFeeBatch{}, &models.KojFee{},
			&models.TbsFeeBatch{}, &models.TbsFee{},
			&models.BillingRun{}, &models.ChargeLine{},
			&models.BillingException{}, &models.SagePostingLog{},
			&models.ReportSnapshot{}, &models.ChangeOfService{},
		}},
		{Label: "attachments", Models: []any{
			&models.Attachment{},
		}},
		{Label: "audit", Models: []any{
			&models.AuditTrail{},
			&models.AlmaFileLog{},
			&models.TransactionSequence{},
			&models.NpgisSubmission{},
			&models.MailOutbox{},
		}},
		{Label: "workflow", Models: []any{
			&models.Process{}, &models.Node{}, &models.Transition{},
			&models.ProcessInstance{}, &models.Task{}, &models.Event{},
			&models.InitiatorPool{}, &models.WorkflowInitiatorPoolUser{},
			&models.WorkflowNotifyUser{},
			&models.ApprovalSubstitute{},
			&models.NodeOperatorRole{}, &models.NodeOperatorUser{},
		}},
	}
}

// AllModels returns every model in migration order.
func AllModels() []any {
	var out []any
	for _, g := range Groups() {
		out = append(out, g.Models...)
	}
	return out
}
