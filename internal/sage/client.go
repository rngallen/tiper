// Package sage reads live Sage 200 / Evolution tables (Client, Currency).
package sage

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Sage Client.iCurrencyID → ISO 4217. CurrencyLink 1–5 match the Sage
// Currency table; 0 is the company home currency (TZS), which has no
// Currency row in this TIPER company.
var currencyBySageID = map[int]string{
	0: "TZS",
	1: "USD",
	2: "GBP",
	3: "EUR",
	4: "ZAR",
	5: "CHF",
}

// CurrencyCode maps Sage Client.iCurrencyID to an ISO code.
func CurrencyCode(id int) (string, bool) {
	code, ok := currencyBySageID[id]
	return code, ok
}

// Client is one Sage AR account (table Client).
type Client struct {
	Account      string `gorm:"column:Account"`
	Name         string `gorm:"column:Name"`
	OnHold       bool   `gorm:"column:On_Hold"`
	CurrencyID   int    `gorm:"column:iCurrencyID"`
	CurrencyCode string `gorm:"-"`
}

func (c *Client) Finish() {
	if c == nil {
		return
	}
	c.Account = strings.TrimSpace(c.Account)
	c.Name = strings.TrimSpace(c.Name)
	c.CurrencyCode, _ = CurrencyCode(c.CurrencyID)
}

const clientActive = `ISNULL(On_Hold, 0) = 0`

// GetClient loads one Client row by Account (exact, trimmed).
func GetClient(ctx context.Context, db *gorm.DB, account string) (Client, error) {
	account = strings.TrimSpace(account)
	if db == nil {
		return Client{}, fmt.Errorf("sage not connected")
	}
	if account == "" {
		return Client{}, gorm.ErrRecordNotFound
	}
	var row Client
	err := db.WithContext(ctx).Table("Client").Where("Account = ?", account).Limit(1).Find(&row).Error
	if err != nil {
		return Client{}, err
	}
	row.Finish()
	if row.Account == "" {
		return Client{}, gorm.ErrRecordNotFound
	}
	return row, nil
}

// ListQuery is the Sage Client list for GET /sage/clients. On-hold accounts
// are omitted. Callers paginate with response.NewPaginator (page / pageSize).
func ListQuery(db *gorm.DB, search string) *gorm.DB {
	q := db.Table("Client").Where(clientActive)
	term := strings.Trim(strings.TrimSpace(search), "%")
	if term != "" {
		like := "%" + escapeLike(term) + "%"
		q = q.Where("(Account LIKE ? OR Name LIKE ?)", like, like)
	}
	return q.Order("Account ASC")
}

func escapeLike(s string) string {
	// SQL Server LIKE treats % and _ as wildcards. Strip them so a search
	// for "100%" cannot match every account.
	s = strings.ReplaceAll(s, `%`, "")
	s = strings.ReplaceAll(s, `_`, "")
	return s
}
