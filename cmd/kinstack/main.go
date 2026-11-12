package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"kinstack/internal/bootstrap"
	"kinstack/internal/pkg/assert"
)

func main() {
	if runSubcommand(os.Args[1:]) {
		return
	}
	assert.NoError(bootstrap.Run())
}

// runSubcommand dispatches the operator commands. It reports whether it ran one,
// so the caller knows whether to boot the server instead.
func runSubcommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "healthcheck":
		healthcheck()
		return true
	case "prune-audit":
		pruneAudit(args[1:])
		return true
	case "backfill-2fa":
		if err := bootstrap.BackfillTwoFactorSecrets(); err != nil {
			fmt.Fprintln(os.Stderr, "backfill-2fa failed:", err)
			os.Exit(1)
		}
		return true
	}
	return false
}

// pruneAudit deletes audit rows older than the retention window. The default is
// a year; an operator may shorten it, and an unparseable value is a usage error.
func pruneAudit(args []string) {
	retention := 8760 * time.Hour
	if len(args) > 0 {
		parsed, err := time.ParseDuration(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "prune-audit: invalid retention:", err)
			os.Exit(2)
		}
		retention = parsed
	}
	if err := bootstrap.PruneAudit(retention); err != nil {
		fmt.Fprintln(os.Stderr, "prune-audit failed:", err)
		os.Exit(1)
	}
}

// healthcheck is the container probe entrypoint. Distroless images have no
// shell or curl, so the application binary itself checks the local health route.
func healthcheck() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck failed:", err)
		os.Exit(1)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck status:", response.StatusCode)
		os.Exit(1)
	}
}
