package docs

import (
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gopkg.in/yaml.v3"
)

// TestSwaggerRoutes verifies Register serves the interactive UI at /api-docs
// (referencing the locally embedded assets, not a CDN), the raw spec at
// /openapi.yaml, the asset files themselves, and passes other paths through.
func TestSwaggerRoutes(t *testing.T) {
	app := fiber.New()
	Register(app)
	app.Get("/other", func(c fiber.Ctx) error { return c.SendString("passthrough") })

	cases := []struct {
		path        string
		contains    string
		contentType string
	}{
		{"/api-docs", "/api-docs/assets/swagger-ui-bundle.js", ""},
		{"/openapi.yaml", "TIPER Depot Fuel Management System API", ""},
		{"/api-docs/assets/swagger-ui-bundle.js", "", "text/javascript; charset=utf-8"},
		{"/api-docs/assets/swagger-ui-standalone-preset.js", "", "text/javascript; charset=utf-8"},
		{"/api-docs/assets/swagger-ui.css", "", "text/css; charset=utf-8"},
		{"/api-docs/assets/favicon-32x32.png", "", "image/png"},
		{"/other", "passthrough", ""},
	}
	for _, tc := range cases {
		resp, err := app.Test(httptest.NewRequest("GET", tc.path, nil))
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("GET %s: status %d, want 200", tc.path, resp.StatusCode)
			continue
		}
		if tc.contentType != "" && resp.Header.Get("Content-Type") != tc.contentType {
			t.Errorf("GET %s: content type %q, want %q", tc.path, resp.Header.Get("Content-Type"), tc.contentType)
		}
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Errorf("GET %s: empty body", tc.path)
		}
		if tc.contains != "" && !strings.Contains(string(body), tc.contains) {
			t.Errorf("GET %s: body does not contain %q", tc.path, tc.contains)
		}
	}

	// The docs page must not reference any CDN.
	resp, err := app.Test(httptest.NewRequest("GET", "/api-docs", nil))
	if err != nil {
		t.Fatalf("GET /api-docs: %v", err)
	}
	page, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(page), "unpkg.com") {
		t.Error("docs page still references unpkg.com; assets should be local")
	}

	// Unknown assets must 404.
	resp, err = app.Test(httptest.NewRequest("GET", "/api-docs/assets/nope.js", nil))
	if err != nil {
		t.Fatalf("GET missing asset: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("missing asset: status %d, want 404", resp.StatusCode)
	}
}

// TestOpenAPISpec ensures the embedded OpenAPI document parses and that every
// internal $ref points at a node that actually exists, so a bad edit fails CI
// instead of breaking Swagger UI at runtime.
func TestOpenAPISpec(t *testing.T) {
	var spec map[string]any
	if err := yaml.Unmarshal(openAPISpec, &spec); err != nil {
		t.Fatalf("openapi.yaml is not valid YAML: %v", err)
	}

	for _, key := range []string{"openapi", "info", "paths", "components"} {
		if _, ok := spec[key]; !ok {
			t.Errorf("openapi.yaml is missing top-level %q", key)
		}
	}

	var refs []string
	collectRefs(spec, &refs)
	if len(refs) == 0 {
		t.Fatal("expected the spec to contain $ref usages")
	}

	for _, ref := range refs {
		if !strings.HasPrefix(ref, "#/") {
			t.Errorf("external $ref not supported: %s", ref)
			continue
		}
		if err := resolveRef(spec, ref); err != nil {
			t.Errorf("unresolved $ref %s: %v", ref, err)
		}
	}
}

func collectRefs(node any, out *[]string) {
	switch v := node.(type) {
	case map[string]any:
		for key, val := range v {
			if key == "$ref" {
				if s, ok := val.(string); ok {
					*out = append(*out, s)
				}
				continue
			}
			collectRefs(val, out)
		}
	case []any:
		for _, item := range v {
			collectRefs(item, out)
		}
	}
}

func resolveRef(spec map[string]any, ref string) error {
	var node any = spec
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, ok := node.(map[string]any)
		if !ok {
			return fmt.Errorf("segment %q: parent is not a mapping", part)
		}
		node, ok = m[part]
		if !ok {
			return fmt.Errorf("segment %q not found", part)
		}
	}
	return nil
}
