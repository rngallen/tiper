// Package docs serves the API documentation using the official
// gofiber/contrib/v3/swaggerui middleware (https://docs.gofiber.io/contrib/swaggerui/).
// The OpenAPI specification and the full Swagger-UI asset bundle
// (swagger-ui-dist) are embedded into the binary, so the interactive explorer
// is fully offline, version-locked to the build and served from the same
// origin — no CDN. The /api-docs routes use a dedicated CSP profile (see
// pkg/middleware/security.go) that permits Swagger UI's inline bootstrap
// script while keeping every asset same-origin.
//
// To upgrade Swagger UI assets:
//
//	cd pkg/docs/assets && for f in swagger-ui-bundle.js \
//	  swagger-ui-standalone-preset.js swagger-ui.css \
//	  favicon-32x32.png favicon-16x16.png; do
//	  curl -fsSLO "https://cdn.jsdelivr.net/npm/swagger-ui-dist@<version>/$f"
//	done
package docs

import (
	"embed"
	"path"

	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
)

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed assets
var assetsFS embed.FS

// assetsPrefix is where the embedded Swagger-UI files are served from.
const assetsPrefix = "/api-docs/assets/"

// swaggerHandler renders the docs page at GET /api-docs and the raw spec at
// GET /openapi.yaml, with all page assets pointed at the embedded copies.
var swaggerHandler = swaggerui.New(swaggerui.Config{
	BasePath:    "/",
	FilePath:    "openapi.yaml",
	FileContent: openAPISpec,
	Path:        "api-docs",
	Title:       "TIPER Depot Fuel Management System API",
	CacheAge:    300,

	SwaggerURL:       assetsPrefix + "swagger-ui-bundle.js",
	SwaggerPresetURL: assetsPrefix + "swagger-ui-standalone-preset.js",
	SwaggerStylesURL: assetsPrefix + "swagger-ui.css",
	Favicon32:        assetsPrefix + "favicon-32x32.png",
	Favicon16:        assetsPrefix + "favicon-16x16.png",
})

// Register mounts the documentation routes on the app: the embedded Swagger-UI
// assets under /api-docs/assets/, the interactive UI at /api-docs and the raw
// OpenAPI document at /openapi.yaml.
func Register(app *fiber.App) {
	app.Get(assetsPrefix+"*", serveAsset)
	app.Use(swaggerHandler)
}

// serveAsset returns one embedded Swagger-UI file. Only the base name of the
// request path is used, so path traversal is impossible.
func serveAsset(c fiber.Ctx) error {
	name := path.Base(c.Path())
	data, err := assetsFS.ReadFile("assets/" + name)
	if err != nil {
		return fiber.ErrNotFound
	}

	// Explicit types (instead of mime.TypeByExtension) keep behaviour
	// identical across platforms — Windows resolves MIME from the registry.
	switch path.Ext(name) {
	case ".js":
		c.Set(fiber.HeaderContentType, "text/javascript; charset=utf-8")
	case ".css":
		c.Set(fiber.HeaderContentType, "text/css; charset=utf-8")
	case ".png":
		c.Set(fiber.HeaderContentType, "image/png")
	}
	c.Set(fiber.HeaderCacheControl, "public, max-age=86400")
	return c.Send(data)
}
