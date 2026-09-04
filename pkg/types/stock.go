package types

// StockStatusCode is a root book-stock bucket. Corridor subclasses (Congo,
// Rwanda, …) stay as master-data rows under TRANSIT.
type StockStatusCode string

const (
	StockLocal     StockStatusCode = "LOCAL"
	StockTransit   StockStatusCode = "TRANSIT"
	StockMining    StockStatusCode = "MINING"
	StockProration StockStatusCode = "PRORATION"
)
