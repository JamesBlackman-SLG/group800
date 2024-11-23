package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
)

func export(db *sql.DB) {
	// Query the data
	query := "SELECT * FROM webhooks;" // Replace with your table name
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Fatalf("Failed to get column names: %v", err)
	}

	// Open the CSV file for writing
	file, err := os.Create("output.csv")
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	if err := writer.Write(columns); err != nil {
		log.Fatalf("Failed to write headers: %v", err)
	}

	// Prepare a slice to hold values and a slice of pointers for Scan
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Write rows
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}

		record := make([]string, len(columns))
		for i, val := range values {
			if val != nil {
				record[i] = fmt.Sprintf("%v", val)
			} else {
				record[i] = ""
			}
		}

		// Properly quote string fields
		for i, field := range record {
			if strings.ContainsAny(field, ",\n\"") {
				record[i] = `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
			}
		}

		if err := writer.Write(record); err != nil {
			log.Fatalf("Failed to write record: %v", err)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error occurred during row iteration: %v", err)
	}

	log.Println("Data successfully exported to output.csv")
}
