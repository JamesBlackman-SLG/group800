package internals

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"group800_web/views"
	"os"
	"time"
)

const CreateWebhooksTable = `
	CREATE TABLE IF NOT EXISTS webhooks (
		id TEXT PRIMARY KEY,
		event TEXT,
		sequence INTEGER,
		dispatched_at INTEGER,
		clocking_type TEXT,
		device_serial_number TEXT,
		client_id TEXT,
		user_id TEXT,
		user_card_number TEXT,
		user_first_name TEXT,
		user_last_name TEXT,
		user_full_name TEXT,
		user_employee_number TEXT,
		department_id TEXT,
		department_name TEXT,
		location_id TEXT,
		location_name TEXT,
		project_id TEXT,
		project_name TEXT,
		project_code TEXT,
		time_zone TEXT,
		time_logged TEXT,
		time_logged_rounded TEXT,
		time_inserted TEXT,
		clocking_action_type_id INTEGER,
		verification_mode_id INTEGER,
		record_hash INTEGER,
		record_ignored BOOLEAN,
		clocking_pair_id TEXT,
		clocking_sequence_id TEXT,
		planning_id INTEGER,
		absence_type_id TEXT,
		absence_type_name TEXT,
		color TEXT,
		comment TEXT,
		pay_period_id TEXT,
		pay_period_name TEXT
	);
	`

const CreateWorkersTable = `
	CREATE TABLE IF NOT EXISTS workers(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
    time_moto_user_id TEXT UNIQUE,
		first_name TEXT,
		last_name TEXT,
		trade TEXT,
		employment_type TEXT,
		invoice_from TEXT
	  -- FOREIGN KEY (time_moto_user_id) REFERENCES webhooks(user_id)
	);
	`

// listUsers retrieves a list of distinct user full names ordered alphabetically
func (app *Config) listUsers(db *sql.DB) ([]*views.User, error) {
	query := `
  SELECT DISTINCT ww.user_id, ww.user_full_name, ww.user_first_name, ww.user_last_name, IFNULL(w.trade, "?") AS trade
  FROM webhooks ww
  LEFT JOIN workers w ON ww.user_first_name = w.first_name AND ww.user_last_name = w.last_name
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
		err := rows.Scan(&user.UserID, &user.FullName, &user.FirstName, &user.LastName, &user.Trade)
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

func weeklyTimeSheet(db *sql.DB, userID string, d time.Time) ([]views.CheckInData, error) {
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
    END AS duration,
    IFNULL(w.trade, '?') AS trade
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
LEFT JOIN workers w ON ci.user_id = w.time_moto_user_id
WHERE 
    ci.clocking_type = 'In'
    AND date(datetime(ci.time_logged)) = ?
    AND (co.time_logged IS NULL OR date(datetime(co.time_logged)) = ? )
    AND ci.user_id = ?
GROUP BY 
    ci.user_id, ci.time_logged
HAVING check_out_time = '' OR duration > '00:00'
ORDER BY 
    ci.user_full_name, check_in;
`
	// Prepare the query
	rows, err := db.QueryContext(context.Background(), query, d.Format("2006-01-02"), d.Format("2006-01-02"), userID)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var data []views.CheckInData

	// Iterate over the rows
	for rows.Next() {
		var row views.CheckInData
		err := rows.Scan(&row.Location, &row.CheckIn, &row.CheckOut, &row.Duration, &row.Trade)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		row.Date = d.Format("2006-01-02")
		row.Name = userID
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

func (app *Config) ImportWorkersFromCSV(db *sql.DB, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare("INSERT INTO workers (time_moto_user_id, first_name, last_name, trade, employment_type, invoice_from) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	timeMotoUsers, err := app.listUsers(db)
	if err != nil {
		return err
	}

	for _, record := range records {
		// find the user in timeMotoUsers
		for _, user := range timeMotoUsers {
			if user.FirstName == record[1] && user.LastName == record[2] {
				fmt.Println("Found user: ", user.FullName)
				fmt.Println("Time Moto User ID: ", user.UserID)
				fmt.Printf("Inserting record: %s", record[0])
				fmt.Println()
				_, err = stmt.Exec(user.UserID, record[1], record[2], record[3], record[4], record[5])
				if err != nil {
					err = tx.Rollback()
					return fmt.Errorf("failed to execute statement: %w", err)
				}
			}
		}

		// fmt.Printf("Inserting record: %s", record[0])
		// fmt.Println()
		// _, err = stmt.Exec(record[0], record[1], record[2], record[3], record[4], record[5])
		// if err != nil {
		// 	tx.Rollback()
		// 	return fmt.Errorf("failed to execute statement: %w", err)
		// }
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
