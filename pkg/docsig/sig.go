// Package docsig issues and checks HMAC tokens for public document verify links.
//
// Design matches the payment-portal voucher QR:
//   - Signing key is HMAC(APP.SYMMETRIC_KEY, purpose) so tokens are not a raw
//     reuse of the PASETO key.
//   - Message is versioned ("v1|" + kind + "|" + uid).
//   - Token is base64url (no padding) of HMAC-SHA256.
//   - No expiry: printed QR codes stay stable across reprints of the same document.
//   - Callers return 404 on failure (same as unknown UID) to avoid oracle leakage.
package docsig

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"dfms/apps/auth/middleware"
)

const (
	purpose   = "dfms/document-verify/v1"
	msgPrefix = "v1|"
)

// Document kinds used in signed URLs. Changing a slug invalidates printed QRs.
const (
	KindILR            = "ilr"
	KindDeliveryNote   = "delivery-note"
	KindGateIn         = "gate-in"
	KindGateOut        = "gate-out"
	KindPumpOver       = "pump-over"
	KindPumpOverReport = "pump-over-report"
	KindITT            = "itt"
	KindReceipt        = "receipt"
	KindZerolization   = "zerolization"
	KindHoldRelease    = "hold-release"
	KindMiLoss         = "miloss"
)

var kinds = map[string]string{
	KindILR:            "Internal loading request",
	KindDeliveryNote:   "Delivery note",
	KindGateIn:         "Gate-in pass",
	KindGateOut:        "Gate-out pass",
	KindPumpOver:       "Pump-over request",
	KindPumpOverReport: "Pump-over report",
	KindITT:            "In-tank transfer",
	KindReceipt:        "Vessel receipt",
	KindZerolization:   "Zerolization",
	KindHoldRelease:    "Financial hold release",
	KindMiLoss:         "MI loss",
}

var (
	mu  sync.RWMutex
	key []byte
)

// Init derives the document HMAC key from the PASETO access key.
func Init() error {
	if len(middleware.SymmetricKey) == 0 {
		return fmt.Errorf("docsig: symmetric key not loaded")
	}
	mac := hmac.New(sha256.New, middleware.SymmetricKey)
	mac.Write([]byte(purpose))
	derived := mac.Sum(nil)
	mu.Lock()
	key = derived
	mu.Unlock()
	return nil
}

func signingKey() ([]byte, error) {
	mu.RLock()
	defer mu.RUnlock()
	if len(key) == 0 {
		return nil, fmt.Errorf("docsig: not initialised")
	}
	return key, nil
}

// NormalizeKind returns the canonical slug, or empty if unknown.
func NormalizeKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	if _, ok := kinds[k]; ok {
		return k
	}
	return ""
}

// Label is the human title for a kind (empty if unknown).
func Label(kind string) string {
	return kinds[NormalizeKind(kind)]
}

// Sign returns a URL-safe HMAC token, or "" if inputs/key are missing.
func Sign(kind, uid string) string {
	kind = NormalizeKind(kind)
	uid = strings.TrimSpace(uid)
	if kind == "" || uid == "" {
		return ""
	}
	k, err := signingKey()
	if err != nil {
		return ""
	}
	mac := hmac.New(sha256.New, k)
	mac.Write([]byte(msgPrefix + kind + "|" + uid))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Valid reports whether sig is the correct token for kind+uid.
func Valid(kind, uid, sig string) bool {
	expected := Sign(kind, uid)
	if expected == "" || sig == "" {
		return false
	}
	if len(expected) != len(sig) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) == 1
}

// Path builds "/verify/document/{kind}/{uid}/{sig}". Empty if signing fails.
func Path(kind, uid string) string {
	kind = NormalizeKind(kind)
	sig := Sign(kind, uid)
	if kind == "" || sig == "" {
		return ""
	}
	return "/verify/document/" + kind + "/" + uid + "/" + sig
}
