package internals

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// FetchDistinctLocations retrieves a list of distinct location names for the current day
func (app *Config) fetchDistinctLocations(db *sql.DB, d time.Time) ([]string, error) {
	query := `
  select distinct location_name 
  from webhooks 
  WHERE date(datetime(time_logged)) = ?
  ORDER BY location_name;
`

	rows, err := db.QueryContext(context.Background(), query, d.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var locations []string

	// Iterate over the rows
	for rows.Next() {
		var location string
		err := rows.Scan(&location)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		locations = append(locations, location)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return locations, nil
}
