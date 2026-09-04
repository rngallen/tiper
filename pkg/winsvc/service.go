// Package winsvc integrates TIPER DFMS with the OS service manager
// (Windows SCM via kardianos/service) and provides the command-line entrypoint.
package winsvc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"dfms/internal/app"
	"dfms/pkg/logs"

	"github.com/kardianos/service"
)

const (
	serviceName        = "TIPERDFMS"
	serviceDisplay     = "TIPER Depot Fuel Management System"
	serviceDescription = "TIPER DFMS API and background workers (stock, billing, approvals, EWURA)."
)

type program struct {
	cancel context.CancelFunc
}

func (p *program) Start(s service.Service) error {
	// SCM does not honour WorkingDirectory on Windows; without this, cwd is
	// C:\Windows\System32 and logs/uploads land in the wrong place.
	if err := chdirToExecutable(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	logs.GoSafe("winsvc.run", func() {
		if err := app.Run(ctx); err != nil {
			logs.Errorf("TIPER DFMS stopped with error: %v", err)
			fmt.Fprintf(os.Stderr, "TIPER DFMS stopped with error: %v\n", err)
			os.Exit(1)
		}
	})
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func newService() (service.Service, error) {
	return service.New(&program{}, &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplay,
		Description: serviceDescription,
		Option: service.KeyValue{
			"StartType": "automatic",
			"OnFailure": "restart",
		},
	})
}

// chdirToExecutable sets cwd to the folder that contains dfms.exe so relative
// paths (logs/, uploads/) resolve next to the binary. Windows services start
// in System32 unless we do this here (WorkingDirectory is ignored on Windows).
func chdirToExecutable() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("chdir %s: %w", dir, err)
	}
	return nil
}

// runDefault runs as a managed service under the SCM, or in the foreground when
// launched interactively.
func runDefault() error {
	s, err := newService()
	if err != nil {
		return err
	}
	if service.Interactive() {
		return runForeground()
	}
	return s.Run()
}

func runForeground() error {
	return app.Run(context.Background())
}

func runServiceCommand(cmd string) error {
	s, err := newService()
	if err != nil {
		return err
	}
	switch cmd {
	case cmdInstall:
		return s.Install()
	case cmdUninstall:
		return s.Uninstall()
	case cmdStart:
		return s.Start()
	case cmdStop:
		return s.Stop()
	case cmdRestart:
		if err := s.Stop(); err != nil {
			return err
		}
		return s.Start()
	case cmdStatus:
		status, err := s.Status()
		if err != nil {
			return err
		}
		fmt.Printf("%s: %v\n", serviceName, status)
		return nil
	default:
		return fmt.Errorf("internal: unknown service command %q", cmd)
	}
}
