package main

import (
	"context"
	"log"
	"os"

	"kinstack/config"
	"kinstack/internal/platform/database"
)

func main() {
	cfg := config.LoadDatabase()
	db, err := database.Open(cfg.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	ctx := context.Background()
	switch command {
	case "up":
		err = db.Migrate(ctx)
	case "down":
		err = db.Down(ctx)
	case "status":
		err = db.Status(ctx)
	default:
		log.Fatalf(
			"unknown migration command %q (use up, down, or status)",
			command,
		)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("migrations %s complete", command)
}
