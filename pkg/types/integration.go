package types

// IntegrationSetting.Key values. Keep the MSSQL check constraint in sync.
const (
	KeyMail      = "mail"
	KeySMS       = "sms"
	KeySchedules = "schedules"
	KeySage      = "sage"
	KeyUploads   = "uploads"
	KeySession   = "session"
	KeyAlma      = "alma"
	KeyNpgis     = "npgis"
	KeyPrecision = "precision"
	KeyOrders    = "orders"
)

// IntegrationKeys is every valid IntegrationSetting key (seed + migrate check).
var IntegrationKeys = []string{
	KeyMail, KeySMS, KeySchedules, KeySage, KeyUploads, KeySession, KeyAlma, KeyNpgis,
	KeyPrecision, KeyOrders,
}
