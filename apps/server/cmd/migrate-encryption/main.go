package main

import (
	"flag"
	"log"

	"mcpist/server/internal/db"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "preview without writing changes")
	flag.Parse()

	database := db.Open()
	db.InitEncryptionKey()

	if !db.EncryptionEnabled() {
		log.Fatal("CREDENTIAL_ENCRYPTION_KEY is not set")
	}

	mode := "LIVE"
	if *dryRun {
		mode = "DRY-RUN"
	}
	log.Printf("[%s] Starting encryption migration...", mode)

	creds, oauth, err := db.MigrateEncryption(database, *dryRun)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Printf("[%s] Done: %d user_credentials, %d oauth_apps migrated", mode, creds, oauth)
}
