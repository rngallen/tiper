// The dfms binary runs TIPER Depot Fuel Management System: Fiber HTTP API,
// workflow approvals, stock ledger, billing engines, and background workers.
// Command-line handling is delegated to pkg/winsvc (run/console, Windows
// service, and "migrate up|reset|status").
package main

import (
	"fmt"
	"os"

	"dfms/pkg/winsvc"
)

func main() {
	if err := winsvc.RunCLI(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
