package internals

import (
	"context"
	"database/sql"
	"fmt"
	"group800_web/views"
	"time"
)

// listUsers retrieves a list of distinct user full names ordered alphabetically
func (app *Config) listUsers(db *sql.DB) ([]*views.User, error) {
	query := `
  SELECT DISTINCT user_full_name 
  FROM webhooks 
  ORDER BY user_full_name;
`

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var users []*views.User

	// Iterate over the rows
	for rows.Next() {
		var user views.User
		err := rows.Scan(&user.FullName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		users = append(users, &user)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return users, nil
}

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

func weeklyTimeSheet(db *sql.DB, userFullName string, d time.Time) ([]views.CheckInData, error) {
	query := `
SELECT 
    ci.location_name,
    strftime('%H:%M', ci.time_logged) AS check_in,
    CASE 
        WHEN co.time_logged IS NOT NULL THEN strftime('%H:%M', co.time_logged)
        ELSE ''
    END AS check_out_time,
    CASE 
        WHEN co.time_logged IS NOT NULL THEN 
            strftime('%H:%M', (julianday(co.time_logged) - julianday(ci.time_logged)) * 86400, 'unixepoch')
        ELSE 
          ''
    END AS duration
FROM
    webhooks ci
LEFT JOIN 
    webhooks co 
ON 
    ci.user_id = co.user_id 
    AND ci.location_name = co.location_name 
    AND ci.clocking_type = 'In' 
    AND co.clocking_type = 'Out' 
    AND date(datetime(ci.time_logged)) = date(datetime(co.time_logged)) 
    AND datetime(co.time_logged) > datetime(ci.time_logged)
WHERE 
    ci.clocking_type = 'In'
    AND date(datetime(ci.time_logged)) = ?
    AND (co.time_logged IS NULL OR date(datetime(co.time_logged)) = ? )
    AND ci.user_full_name = ?
GROUP BY 
    ci.user_id, ci.time_logged
HAVING check_out_time = '' OR duration > '00:00'
ORDER BY 
    ci.user_full_name, check_in;
`
	// Prepare the query
	rows, err := db.QueryContext(context.Background(), query, d.Format("2006-01-02"), d.Format("2006-01-02"), userFullName)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var data []views.CheckInData

	// Iterate over the rows
	for rows.Next() {
		var row views.CheckInData
		err := rows.Scan(&row.Location, &row.CheckIn, &row.CheckOut, &row.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		row.Date = d.Format("2006-01-02")
		row.Name = userFullName
		data = append(data, row)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(data) == 0 {
		fmt.Println("No data")
		return nil, nil
	}

	// Convert data to JSON for OpenAI
	// dataJSON, err := json.Marshal(data)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
	// }

	// Convert JSON data to a string for easy submission
	return data, nil
}
