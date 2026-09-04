package export

import (
	"strings"
	"sync"
)

// DefaultCompanyName is printed when Company is missing or unnamed.
const DefaultCompanyName = "Tanzania International Petroleum Reserves Limited"

// Letterhead is company identity printed at the top of official documents
// and tabular reports (same block as the ILR).
type Letterhead struct {
	CompanyName, Address, Address2, City, Postal, Country string
	Phone, Email, Website, TIN, VRN, ISO, LogoPath        string
}

var (
	letterheadMu sync.RWMutex
	letterheadFn func() Letterhead
)

// UseLetterhead registers how to load the organisation row (Company ID 1).
func UseLetterhead(fn func() Letterhead) {
	letterheadMu.Lock()
	letterheadFn = fn
	letterheadMu.Unlock()
}

// ActiveLetterhead returns the registered organisation, or an empty head.
func ActiveLetterhead() Letterhead {
	letterheadMu.RLock()
	fn := letterheadFn
	letterheadMu.RUnlock()
	if fn == nil {
		return Letterhead{}
	}
	return fn()
}

func resolveLetterhead(h Letterhead) Letterhead {
	if strings.TrimSpace(h.CompanyName) != "" || strings.TrimSpace(h.LogoPath) != "" {
		return h
	}
	if loaded := ActiveLetterhead(); strings.TrimSpace(loaded.CompanyName) != "" || strings.TrimSpace(loaded.LogoPath) != "" {
		return loaded
	}
	return h
}

func (d ILRDoc) Head() Letterhead {
	return Letterhead{
		CompanyName: d.CompanyName, Address: d.Address, Address2: d.Address2,
		City: d.City, Postal: d.Postal, Country: d.Country, Phone: d.Phone,
		Email: d.Email, Website: d.Website, TIN: d.TIN, VRN: d.VRN, ISO: d.ISO,
		LogoPath: d.LogoPath,
	}
}

// DisplayName is Company.Name from the database, with a last-resort fallback
// only when that row has not been seeded yet.
func (h Letterhead) DisplayName() string {
	h = resolveLetterhead(h)
	return first(h.CompanyName, DefaultCompanyName)
}

// Lines is the organisation block used on Excel reports (no logo).
func (h Letterhead) Lines() []string {
	h = resolveLetterhead(h)
	var lines []string
	lines = append(lines, h.DisplayName())
	lines = append(lines, letterheadAddress(h)...)
	if contact := joinNonEmpty(prefix("Tel ", h.Phone), prefix("Email ", h.Email), prefix("Web ", h.Website)); contact != "" {
		lines = append(lines, contact)
	}
	if tax := joinNonEmpty(prefix("VRN ", h.VRN), prefix("TIN ", h.TIN), h.ISO); tax != "" {
		lines = append(lines, tax)
	}
	return lines
}
