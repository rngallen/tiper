// Package app is the composition root.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	auditapi "dfms/apps/audit"
	"dfms/apps/auth"
	authmiddleware "dfms/apps/auth/middleware"
	billingapi "dfms/apps/billing"
	ewuraapi "dfms/apps/ewura"
	inventoryapi "dfms/apps/inventory"
	masterapi "dfms/apps/masterdata"
	ordersapi "dfms/apps/orders"
	publicapi "dfms/apps/public"
	reportsapi "dfms/apps/reports"
	sageapi "dfms/apps/sage"
	settingsapi "dfms/apps/settings"
	wfapi "dfms/apps/workflow"
	"dfms/internal/alma"
	"dfms/internal/billing"
	"dfms/internal/ewura"
	"dfms/internal/integrations"
	"dfms/internal/inventory"
	"dfms/internal/jobs"
	"dfms/internal/notify"
	"dfms/internal/orders"
	wfengine "dfms/internal/workflow"
	"dfms/pkg/audit"
	"dfms/pkg/config"
	"dfms/pkg/db"
	"dfms/pkg/docs"
	"dfms/pkg/docsig"
	"dfms/pkg/logs"
	"dfms/pkg/middleware"
	"dfms/pkg/migrate"
	"dfms/pkg/response"
	"dfms/pkg/types"
	"dfms/pkg/types/attachment"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

const (
	appName                = "TIPER Depot Fuel Management System"
	serverHeader           = "DFMS"
	defaultShutdownTimeout = 20 * time.Second
)

var defaultAllowedHeaders = []string{
	"Origin", "Content-Type", "Accept", "Authorization",
	"Content-Length", "Accept-Encoding", "Idempotency-Key", "X-Request-ID",
	"X-Csrf-Token",
}

func normalizeCORSOrigins(origins []string) []string {
	out := make([]string, 0, len(origins))
	seen := make(map[string]struct{})
	for _, raw := range origins {
		for part := range strings.SplitSeq(raw, ",") {
			o := strings.TrimSpace(part)
			if o == "" {
				continue
			}
			if _, ok := seen[o]; ok {
				continue
			}
			seen[o] = struct{}{}
			out = append(out, o)
		}
	}
	return out
}

func Run(ctx context.Context) error {
	appLogger, accessLogger, dbLogger, err := logs.Setup()
	if err != nil || appLogger == nil || accessLogger == nil || dbLogger == nil {
		return fmt.Errorf("init logs: %w", err)
	}
	logs.Info("Logging initialized")

	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()

	if err := config.InitConfig(appCtx); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logs.Info("Configuration loaded successfully")
	if config.Conf.App.Debug {
		logs.Warn("DFMS.DEBUG=true: development conveniences enabled — disable in production")
		logs.SetVerbose(true)
	}

	cronScheduler := cron.New(cron.WithSeconds())
	jobMgr := jobs.Bind(cronScheduler)

	appDBCtx, appDBCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer appDBCancel()
	if err := db.ConnectDatabase(appDBCtx); err != nil {
		return fmt.Errorf("connect app database: %w", err)
	}
	if err := migrate.RequireReady(db.Db); err != nil {
		return err
	}
	logs.Info("application database ready (schema migrate/seed is CLI-only: migrate up)")

	if err := integrations.Bootstrap(db.Db); err != nil {
		return fmt.Errorf("integration settings: %w", err)
	}

	sageDBCtx, sageDBCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer sageDBCancel()
	if err := db.ApplySageConfig(sageDBCtx, integrations.Default.Sage()); err != nil {
		logs.Warnf("Sage not connected (configure under Settings → Sage): %v", err)
	}

	audit.Default = audit.NewRecorder(db.Db)
	reportsapi.BindLetterhead()
	if err := authmiddleware.InitKeys(); err != nil {
		return err
	}
	if err := docsig.Init(); err != nil {
		return fmt.Errorf("document verify signatures: %w", err)
	}
	if err := authmiddleware.InitPermissionCache(); err != nil {
		return fmt.Errorf("auth caches: %w", err)
	}
	auth.SetDeliverer(notify.NewOTPDeliverer(db.Db))

	if err := logs.RegisterLogRotationJob(appCtx, jobMgr, appLogger, accessLogger, dbLogger); err != nil {
		return fmt.Errorf("log rotation job: %w", err)
	}

	engine := wfengine.New(db.Db, notify.NewWorkflowNotifier(db.Db))
	ledger := inventory.NewService(db.Db, engine)
	bills := billing.NewService(db.Db, engine, ledger)
	orderSvc := orders.NewService(db.Db, engine, ledger)
	engine.RegisterHook(types.ReceiptContent, inventory.NewReceiptHook(ledger, bills))
	engine.RegisterHook(types.ZerolizationContent, inventory.NewZerolHook(ledger))
	engine.RegisterHook(types.IttTransferContent, inventory.NewIttHook(ledger))
	engine.RegisterHook(types.FinancialHoldContent, inventory.NewHoldHook(ledger))
	engine.RegisterHook(types.GantryLoadingRequestContent, orders.NewGLRHook(orderSvc))
	engine.RegisterHook(types.PumpOverRequestContent, orders.NewPumpHook(orderSvc))
	engine.RegisterHook(types.PumpOverReportContent, orders.NewPumpReportHook(orderSvc))
	engine.RegisterHook(types.CompartmentalizationContent, orders.NewCompHook(orderSvc))
	engine.RegisterHook(types.OrderAmendmentContent, orders.NewAmendHook(orderSvc))
	billing.RegisterHooks(engine)

	orderSvc.SetAlmaRoot(integrations.Default.Alma().FilePath)
	if paths, ok := alma.EnabledRoot(integrations.Default.Alma().FilePath); ok {
		if err := alma.Watch(appCtx, db.Db, paths, orderSvc); err != nil {
			logs.Warnf("ALMA watcher not started: %v", err)
		}
	}

	ewura.RegisterJobs(appCtx, jobMgr, db.Db)
	ewura.RegisterNpgisJob(appCtx, jobMgr, db.Db, func() ewura.NpgisConfig {
		c := integrations.Default.Npgis()
		return ewura.NpgisConfig{
			Enabled: c.Enabled, BaseURL: c.BaseURL, LicenseNo: c.LicenseNo,
			APISourceID: c.APISourceID, DepotName: c.DepotName,
		}
	})
	bills.RegisterJobs(appCtx, jobMgr)
	orderSvc.RegisterJobs(appCtx, jobMgr)
	notify.RegisterJobs(jobMgr, db.Db)

	integrations.SetApplySchedulesHook(func(cfg config.SchedulesConfig) error {
		return jobMgr.Apply(cfg)
	})
	if err := integrations.Default.ApplySchedules(); err != nil {
		return fmt.Errorf("apply job schedules: %w", err)
	}

	trustProxy := config.Conf.App.TrustForwardedFor

	httpApp := fiber.New(fiber.Config{
		AppName:            appName,
		ServerHeader:       serverHeader,
		CaseSensitive:      true,
		StrictRouting:      false,
		JSONEncoder:        json.Marshal,
		JSONDecoder:        json.Unmarshal,
		BodyLimit:          attachment.ProcessBodyLimit,
		ErrorHandler:       response.ErrorHandler,
		TrustProxy:         trustProxy,
		ProxyHeader:        fiber.HeaderXForwardedFor,
		EnableIPValidation: trustProxy,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true,
			Private:  true,
		},
	})

	httpApp.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(_ fiber.Ctx, e any) {
			logs.Errorf("recovered panic: %v\n%s", e, debug.Stack())
		},
	}))
	httpApp.Use(requestid.New())
	if trustProxy {
		httpApp.Use(middleware.OverwriteRealIP())
	}
	httpApp.Use(logs.AccessMiddleware(accessLogger))
	httpApp.Use(limiter.New(limiter.Config{
		Max:        200,
		Expiration: time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			if p == "/healthz" || p == "/readyz" || strings.HasPrefix(p, "/api/v1/public/") {
				return true
			}
			return false
		},
		LimitReached: func(c fiber.Ctx) error {
			return response.TooManyRequests(c)
		},
	}))

	corsOrigins := normalizeCORSOrigins(config.Conf.App.Cors)
	httpApp.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowMethods: []string{
			fiber.MethodGet, fiber.MethodHead, fiber.MethodPost,
			fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete, fiber.MethodOptions,
		},
		AllowHeaders:     defaultAllowedHeaders,
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           7200,
	}))
	middleware.SetupSecurityMiddleware(httpApp)
	httpApp.Use(csrf.New(csrf.Config{
		CookieName:        types.CsrfCookieName,
		CookieSecure:      config.Conf.App.CookieSecure(),
		CookieHTTPOnly:    false,
		CookieSameSite:    "Lax",
		CookieSessionOnly: true,
		CookiePath:        types.CsrfCookiePath,
		Extractor:         extractors.FromHeader(csrf.HeaderName),
		TrustedOrigins:    corsOrigins,
		Next:              middleware.SkipCSRF,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logs.Errorf("csrf %s %s: %v", c.Method(), c.Path(), err)
			return response.Forbidden(c, "CSRF validation failed — refresh the page and try again")
		},
	}))

	httpApp.Get("/healthz", healthcheck.New(healthcheck.Config{
		ResponseFormat: healthcheck.FormatJSON,
	}))
	httpApp.Get("/readyz", healthcheck.New(healthcheck.Config{
		ResponseFormat: healthcheck.FormatJSON,
		Probe: func(fiber.Ctx) bool {
			return pingGorm(db.Db)
		},
	}))

	if config.Conf.App.Debug {
		docs.Register(httpApp)
	}

	auth.AuthRouter(httpApp)
	publicapi.Router(httpApp)
	wfapi.WorkflowRouter(httpApp, engine)
	auditapi.AuditRouter(httpApp)
	settingsapi.SettingsRouter(httpApp)
	masterapi.Router(httpApp)
	inventoryapi.Router(httpApp, ledger)
	ordersapi.Router(httpApp, orderSvc)
	billingapi.Router(httpApp, bills, engine)
	ewuraapi.Router(httpApp)
	reportsapi.Router(httpApp)
	sageapi.Router(httpApp)

	cronScheduler.Start()

	listenConfig := fiber.ListenConfig{
		ShutdownTimeout:       config.Conf.App.ShutdownTimeout,
		GracefulContext:       appCtx,
		DisableStartupMessage: !config.Conf.App.Debug,
	}
	if listenConfig.ShutdownTimeout <= 0 {
		listenConfig.ShutdownTimeout = defaultShutdownTimeout
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 4)
	logs.Infof("Starting HTTP server on %s", config.Conf.App.ListenAddress)
	logs.WGGoSafe(&wg, "http.server", func() {
		if err := httpApp.Listen(config.Conf.App.ListenAddress, listenConfig); err != nil {
			errChan <- fmt.Errorf("fiber listen: %w", err)
		}
	})

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	select {
	case sig := <-signalChan:
		logs.Infof("Received shutdown signal: %v", sig)
	case err := <-errChan:
		logs.Errorf("Critical service failure: %v", err)
	case <-appCtx.Done():
		logs.Info("Application context cancelled")
	}
	appCancel()
	auth.StopBackgroundWorkers()
	return shutdown(httpApp, cronScheduler, dbLogger, appLogger, accessLogger, &wg, listenConfig.ShutdownTimeout)
}

func pingGorm(g *gorm.DB) bool {
	if g == nil {
		return false
	}
	sqlDB, err := g.DB()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx) == nil
}
