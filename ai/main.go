package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	var db *sql.DB

	tursoUrl := os.Getenv("GROUP_800_TURSO_URL")
	tursoToken := os.Getenv("GROUP_800_TURSO_TOKEN")

	if tursoUrl == "" {
		log.Fatal("No turso url found in .bashrc")
	}
	url := fmt.Sprintf("%s?authToken=%s", tursoUrl, tursoToken)

	db, err := sql.Open("libsql", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s", err)
		return
	}
	defer db.Close()

	err = analyzeTrendsForAllLocations(db)
	if err != nil {
		log.Fatalf("failed to analyze trends: %v", err)
	}
}
