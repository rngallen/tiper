package logs

import (
	"io"
	stdlog "log"
	"os"
	"strings"
	"time"

	fiberzerolog "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	fiberlog "github.com/gofiber/fiber/v3/log"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// serviceName tags every structured log line so logs from multiple services
// can be distinguished in a shared aggregator.
const serviceName = "tiper-dfms"

// InitZerolog configures the process-wide application logger used by this
// package, github.com/rs/zerolog/log, and the Go standard library. HTTP
// access logs use Fiber's maintained zerolog middleware instead
// (https://docs.gofiber.io/contrib/zerolog/). The writer is the rotating
// logs/app.log handle from InitLogs — call Setup() so GORM is wired too.
func InitZerolog(w io.Writer) zerolog.Logger {
	if w == nil {
		w = os.Stderr
	}

	zerolog.TimeFieldFormat = time.RFC3339

	app := zerolog.New(w).With().
		Timestamp().
		Str("service", serviceName).
		Str("log_type", "app").
		Logger()

	log.Logger = app

	// stdlib log.Print* (and Fiber's default logger, which writes to an
	// io.Writer) become zerolog info lines in app.log instead of stderr.
	stdlog.SetFlags(0)
	stdlog.SetPrefix("")
	stdlog.SetOutput(app)
	fiberlog.SetOutput(app)
	fiberlog.SetLevel(fiberlog.LevelInfo)

	return app
}

// SetVerbose raises Fiber's built-in log level to debug. Application Debugf
// still writes regardless; callers that emit secrets must gate on DFMS.DEBUG.
func SetVerbose(debug bool) {
	if debug {
		fiberlog.SetLevel(fiberlog.LevelDebug)
		return
	}
	fiberlog.SetLevel(fiberlog.LevelInfo)
}

// AccessMiddleware is Fiber's official zerolog access logger
// (github.com/gofiber/contrib/v3/zerolog). Authorization and other credential
// headers are redacted by the middleware defaults. Health and docs routes are
// skipped. See https://docs.gofiber.io/contrib/zerolog/
func AccessMiddleware(w io.Writer) fiber.Handler {
	if w == nil {
		w = os.Stderr
	}

	access := zerolog.New(w).With().
		Timestamp().
		Str("service", serviceName).
		Str("log_type", "access").
		Logger()

	return fiberzerolog.New(fiberzerolog.Config{
		Logger: &access,
		Next:   skipAccessLog,
		Fields: []string{
			fiberzerolog.FieldRequestID,
			fiberzerolog.FieldIP,
			fiberzerolog.FieldLatency,
			fiberzerolog.FieldStatus,
			fiberzerolog.FieldMethod,
			fiberzerolog.FieldRoute,
			fiberzerolog.FieldURL,
			fiberzerolog.FieldBytesSent,
			fiberzerolog.FieldUserAgent,
			fiberzerolog.FieldError,
		},
		FieldsSnakeCase: true,
	})
}

func skipAccessLog(c fiber.Ctx) bool {
	switch c.Path() {
	case "/healthz", "/readyz", "/openapi.yaml":
		return true
	default:
		return strings.HasPrefix(c.Path(), "/api-docs")
	}
}
