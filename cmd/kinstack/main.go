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
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			healthcheck()
			return
		case "backfill-2fa":
			if err := bootstrap.BackfillTwoFactorSecrets(); err != nil {
				fmt.Fprintln(os.Stderr, "backfill-2fa failed:", err)
				os.Exit(1)
			}
			return
		}
	}
	assert.NoError(bootstrap.Run())
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
