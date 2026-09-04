package reports

// latestClosingSQL is the latest StockDailyPosition closing on or before a date,
// rolled up to customer × product. Callers bind one date argument.
const latestClosingSQL = `
	SELECT d.CustomerID, d.ProductID, SUM(d.ClosingQty) AS ClosingQty
	FROM (
		SELECT x.CustomerID, x.ProductID, x.StockStatusID, x.ClosingQty,
			ROW_NUMBER() OVER (
				PARTITION BY x.CustomerID, x.ProductID, x.StockStatusID
				ORDER BY x.PositionDate DESC
			) rn
		FROM StockDailyPosition x
		WHERE x.PositionDate <= ?
	) d
	WHERE d.rn = 1
	GROUP BY d.CustomerID, d.ProductID`

// periodFlowSQL sums daily flow columns in [from, to]. Bind from, to.
const periodFlowSQL = `
	SELECT CustomerID, ProductID,
		SUM(ReceivedQty) AS ReceivedQty,
		SUM(OutflowQty) AS OutflowQty,
		SUM(LoadingQty) AS LoadingQty,
		SUM(PumpOverQty) AS PumpOverQty,
		SUM(ITTQty) AS ITTQty,
		SUM(AdjustmentQty) AS AdjustmentQty
	FROM StockDailyPosition
	WHERE PositionDate >= ? AND PositionDate <= ?
	GROUP BY CustomerID, ProductID`
