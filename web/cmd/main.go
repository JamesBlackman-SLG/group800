package main

import (
	"database/sql"
	"fmt"
	"group800_web/internals"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	var db *sql.DB

	tursoUrl := os.Getenv("TURSO_800_GROUP_URL")
	tursoToken := os.Getenv("TURSO_SLG_TOKEN")

	if tursoUrl == "" {
		log.Fatal("TURSO_800_GROUP_URL env var not set")
	}

	if tursoToken == "" {
		log.Fatal("TURSO_SLG_TOKEN env var not set")
	}

	url := fmt.Sprintf("%s?authToken=%s", tursoUrl, tursoToken)
	log.Println(url)
	db, err := sql.Open("libsql", url)
	if err != nil {
		log.Fatal("failed to open db")
	}
	defer db.Close()

	_, err = db.Exec(internals.CreateWebhooksTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(internals.CreateWorkersTable)
	if err != nil {
		log.Fatal(err)
	}

	err = internals.ImportWorkersFromCSV(db, "../workers.csv")
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	// initialize config
	app := internals.Config{Router: router, DB: db}

	// routes
	app.Routes()

	log.Println("Group 800 Web API running on :8080")
	err = router.Run("127.0.0.1:8080")
	if err != nil {
		log.Fatal("failed to run server")
	}
}
