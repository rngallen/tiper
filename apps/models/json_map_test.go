package models

import "testing"

func TestJSONMapValueAndScan(t *testing.T) {
	m := JSONMap{"host": "smtp.example.com", "port": float64(587), "enabled": true}
	v, err := m.Value()
	if err != nil {
		t.Fatal(err)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		t.Fatalf("Value() want JSON string, got %#v", v)
	}

	var out JSONMap
	if err := out.Scan(s); err != nil {
		t.Fatal(err)
	}
	if out["host"] != "smtp.example.com" {
		t.Fatalf("host: got %#v", out["host"])
	}
	if out["enabled"] != true {
		t.Fatalf("enabled: got %#v", out["enabled"])
	}

	var empty JSONMap
	if err := empty.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if empty == nil {
		t.Fatal("Scan(nil) should yield empty map, not nil")
	}
}
