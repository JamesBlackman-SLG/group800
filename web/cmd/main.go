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

	router := gin.Default()

	// initialize config
	app := internals.Config{Router: router, DB: db}

	// routes
	app.Routes()

	err = router.Run(":8080")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run server %s", err)
		return
	}
}
