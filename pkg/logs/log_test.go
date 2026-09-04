package logs

import (
	"bytes"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestErrorfIncludesCallSite(t *testing.T) {
	var buf bytes.Buffer
	InitZerolog(&buf)
	Errorf("test error %s", "here")
	out := buf.String()
	if !strings.Contains(out, "log_test.go") {
		t.Fatalf("error log missing call site, got %s", out)
	}
	if strings.Contains(out, "log.go") && !strings.Contains(out, "log_test.go") {
		t.Fatalf("caller skipped too few frames: %s", out)
	}
}

func TestInitZerologRoutesStdlib(t *testing.T) {
	var buf bytes.Buffer
	InitZerolog(&buf)

	Infof("via pkg/logs")
	stdlog.Print("via stdlib log")

	out := buf.String()
	for _, want := range []string{"via pkg/logs", "via stdlib log", `"log_type":"app"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}
}

func TestGormWriterUsesDBZerolog(t *testing.T) {
	var buf bytes.Buffer
	InitDBZerolog(&buf)
	GormWriter{}.Printf("SELECT 1\n")
	out := buf.String()
	if !strings.Contains(out, "SELECT 1") {
		t.Fatalf("missing SQL in %s", out)
	}
	if !strings.Contains(out, `"log_type":"db"`) {
		t.Fatalf("missing db log_type in %s", out)
	}
}

func TestAccessMiddlewareUsesContribZerolog(t *testing.T) {
	var buf bytes.Buffer
	app := fiber.New()
	app.Use(AccessMiddleware(&buf))
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/healthz", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	out := buf.String()
	for _, want := range []string{`"log_type":"access"`, `"method":"GET"`, `"status":200`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %s", want, out)
		}
	}

	buf.Reset()
	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if buf.Len() != 0 {
		t.Fatalf("healthz should skip access log, got %s", buf.String())
	}
}
