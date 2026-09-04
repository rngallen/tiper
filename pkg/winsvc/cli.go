package winsvc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"dfms/pkg/config"
	"dfms/pkg/db"
	"dfms/pkg/logs"
	"dfms/pkg/migrate"
)

const (
	cmdInstall   = "install"
	cmdUninstall = "uninstall"
	cmdStart     = "start"
	cmdStop      = "stop"
	cmdRestart   = "restart"
	cmdStatus    = "status"
	cmdRun       = "run"
	cmdConsole   = "console"
	cmdVersion   = "version"
	cmdHelp      = "help"
	cmdMigrate   = "migrate"
)

// Version is set at build time via:
//
//	GOOS=windows GOARCH=amd64 go build -ldflags "-X dfms/pkg/winsvc.Version=1.2.3"
var Version = "dev"

// RunCLI parses os.Args and dispatches the matching command.
func RunCLI() error {
	if len(os.Args) < 2 {
		return runDefault()
	}

	cmd := strings.ToLower(strings.TrimSpace(os.Args[1]))
	switch cmd {
	case "-h", "-help", "--help", cmdHelp, "?":
		printUsage(os.Stdout)
		return nil
	case cmdVersion, "-v", "--version":
		printVersion(os.Stdout)
		return nil
	case cmdInstall, cmdUninstall, cmdStart, cmdStop, cmdRestart, cmdStatus:
		return runServiceCommand(cmd)
	case cmdRun, cmdConsole:
		return runForeground()
	case cmdMigrate:
		return runMigrate(os.Args[2:])
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q — run %s help", cmd, filepath.Base(os.Args[0]))
	}
}

// runMigrate connects to the application database and applies schema commands.
func runMigrate(args []string) error {
	sub := "up"
	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	appLogger, _, dbLogger, err := logs.Setup()
	if err != nil {
		return fmt.Errorf("init logs: %w", err)
	}
	defer func() {
		_ = appLogger.Close()
		_ = dbLogger.Close()
	}()

	if err := config.InitConfig(ctx); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := db.ConnectDatabase(ctx); err != nil {
		return fmt.Errorf("connect app database: %w", err)
	}
	defer func() { _ = db.CloseDatabase(context.Background()) }()

	switch sub {
	case "up", "sync", "bootstrap":
		if err := migrate.Up(db.Db); err != nil {
			return err
		}
		fmt.Println("migration complete")
		return nil
	case "reset":
		if err := migrate.Reset(db.Db); err != nil {
			return err
		}
		fmt.Println("schema reset complete")
		return nil
	case "status":
		empty := migrate.IsEmpty(db.Db)
		if empty {
			fmt.Println("database: empty (run: migrate up)")
		} else {
			fmt.Println("database: initialized")
		}
		return nil
	default:
		return fmt.Errorf("unknown migrate subcommand %q (up|reset|status)", sub)
	}
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s %s (%s/%s)\n", serviceDisplay, Version, runtime.GOOS, runtime.GOARCH)
}

func printUsage(w io.Writer) {
	exe := filepath.Base(os.Args[0])
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s — %s\n\n", serviceDisplay, serviceDescription)
	fmt.Fprintf(w, "Usage:\n  %s [command] [options]\n\n", exe)
	fmt.Fprintln(w, "Commands (no command = service mode under SCM, else foreground):")
	fmt.Fprintln(tw, "  (none)\tRun as service when started by SCM; otherwise foreground")
	fmt.Fprintln(tw, "  run, console\tRun API and workers in the foreground")
	fmt.Fprintln(tw, "  install\tRegister the service (Administrator)")
	fmt.Fprintln(tw, "  uninstall\tRemove service registration")
	fmt.Fprintln(tw, "  start\tStart the installed service")
	fmt.Fprintln(tw, "  stop\tStop the installed service")
	fmt.Fprintln(tw, "  restart\tStop then start the service")
	fmt.Fprintln(tw, "  status\tShow service status")
	fmt.Fprintln(tw, "  migrate\tDatabase schema (see below)")
	fmt.Fprintln(tw, "  version\tPrint build version")
	fmt.Fprintln(tw, "  help\tShow this help")
	_ = tw.Flush()

	fmt.Fprintln(w, "\nMigrate subcommands:")
	fmt.Fprintf(w, "  %s migrate up       Schema + seed (auth, reference, workflow) — run before server\n", exe)
	fmt.Fprintf(w, "  %s migrate status   Report whether the schema is initialized\n", exe)
	fmt.Fprintf(w, "  %s migrate reset    Drop all tables, re-migrate and re-seed (destructive)\n", exe)

	fmt.Fprintln(w, "\nConfiguration:")
	fmt.Fprintf(w, "  Non-secret .env (first match): DFMS_CONFIG_DIR, %s, ./config, current directory.\n", config.PlatformConfigDir())
	fmt.Fprintf(w, "  Secrets (Windows): service Environment REG_MULTI_SZ on TIPERDFMS (DFMS_SYMMETRIC_KEY, …).\n")
	fmt.Fprintf(w, "  Fallback: secrets.env in the ProgramData directory. DFMS_* env vars override files.\n")
	fmt.Fprintf(w, "  See docs/windows-deploy.md (HTTP) and docs/windows-deploy-tls.md (HTTPS).\n")
}
