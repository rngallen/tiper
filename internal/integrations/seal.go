package integrations

import (
	"fmt"
	"strings"

	"dfms/pkg/crypto"
	"dfms/pkg/types"
)

var secretKeysBySetting = map[string][]string{
	types.KeyMail: {"password"},
	types.KeySMS:  {"apiKey"},
	types.KeySage: {"password"},
}

func sealConfigSecrets(key string, cfg map[string]any, keyMaterial string) (map[string]any, error) {
	if cfg == nil {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(cfg))
	for k, v := range cfg {
		out[k] = v
	}
	for _, secretKey := range secretKeysBySetting[key] {
		raw, ok := out[secretKey]
		if !ok || raw == nil {
			continue
		}
		plain, ok := raw.(string)
		if !ok {
			continue
		}
		plain = strings.TrimSpace(plain)
		if plain == "" {
			delete(out, secretKey)
			continue
		}
		if crypto.IsSealed(plain) {
			continue
		}
		sealed, err := crypto.Seal(plain, keyMaterial)
		if err != nil {
			return nil, fmt.Errorf("seal %s.%s: %w", key, secretKey, err)
		}
		out[secretKey] = sealed
	}
	return out, nil
}

func openConfigSecrets(key string, cfg map[string]any, keyMaterial string) (map[string]any, error) {
	if cfg == nil {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(cfg))
	for k, v := range cfg {
		out[k] = v
	}
	for _, secretKey := range secretKeysBySetting[key] {
		raw, ok := out[secretKey]
		if !ok || raw == nil {
			continue
		}
		sealed, ok := raw.(string)
		if !ok {
			continue
		}
		sealed = strings.TrimSpace(sealed)
		if sealed == "" {
			delete(out, secretKey)
			continue
		}
		plain, err := crypto.Open(sealed, keyMaterial)
		if err != nil {
			return nil, fmt.Errorf("open %s.%s: %w", key, secretKey, err)
		}
		out[secretKey] = plain
	}
	return out, nil
}
