// const todaysData = `
//   select id, event, user_full_name, location_name, clocking_type, time(datetime(time_logged)) from webhooks WHERE clocking_type='In' and date(datetime(time_logged)) = date('now') AND time(datetime(time_logged)) >= '08:00:00' order by dispatched_at;
// `

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

type CheckInData struct {
	Name string `json:"name"`
	// LocationName      string `json:"locationName"`
	Type     string `json:"type"`
	Time     string `json:"time"`
	Location string `json:"location"`
	CheckIn  string `json:"check_in"`
	CheckOut string `json:"check_out"`
	Duration string `json:"duration"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func dailyCheckInAnalysis(db *sql.DB, locationName string) (string, error) {
	query := `
SELECT 
    ci.user_full_name,
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
    AND date(datetime(ci.time_logged)) = date('now', '-0 day')
    AND (co.time_logged IS NULL OR date(datetime(co.time_logged)) = date('now', '-0 day'))
    AND ci.location_name = ?
    -- AND (co.location_name IS NULL OR co.location_name = ci.location_name)
    -- AND (co.time_logged IS NULL OR (time(datetime(ci.time_logged)) >= '08:05:00' OR time(datetime(co.time_logged)) <= '16:55:00'))
GROUP BY 
    ci.user_id, ci.time_logged
ORDER BY 
    ci.user_full_name, check_in;
`
	// Prepare the query
	rows, err := db.QueryContext(context.Background(), query, locationName)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var data []CheckInData

	// Iterate over the rows
	for rows.Next() {
		var row CheckInData
		err := rows.Scan(&row.Name, &row.CheckIn, &row.CheckOut, &row.Duration)
		if err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		data = append(data, row)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("row iteration error: %w", err)
	}

	if len(data) == 0 {
		return "No data for today", nil
	}

	// Create a new tab writer
	var buf bytes.Buffer
	writer := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

	// Write the header
	fmt.Fprintln(writer, "Name\tCheck In\tCheck Out\tDuration")

	// Write the data
	for _, row := range data {
		name := row.Name
		if len(name) > 24 {
			name = name[:24] // Truncate if longer than 20 characters
		} else {
			name = fmt.Sprintf("%-24s", name) // Left-align and pad with spaces to 20 characters
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", name, row.CheckIn, row.CheckOut, row.Duration)
	}

	// Flush the writer
	writer.Flush()

	// Print the tabulated data
	fmt.Println(buf.String())

	// Convert data to JSON for OpenAI
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Convert JSON data to a string for easy submission
	return string(dataJSON), nil
}

// FetchDistinctLocations retrieves a list of distinct location names for the current day
func fetchDistinctLocations(db *sql.DB) ([]string, error) {
	query := `
  select distinct location_name 
  from webhooks 
  WHERE date(datetime(time_logged)) = date('now', '-0 day')
  ORDER BY location_name;
`

	rows, err := db.QueryContext(context.Background(), query)
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

// AnalyzeTrendsForAllLocations queries distinct locations and analyzes trends for each
func analyzeTrendsForAllLocations(db *sql.DB) error {
	log.Printf("Analyzing trends for all locations for date: %s", time.Now().Format("2 January 2006"))
	locations, err := fetchDistinctLocations(db)
	if err != nil {
		return fmt.Errorf("failed to fetch distinct locations: %w", err)
	}

	for _, location := range locations {
		fmt.Println(location)
		_, err := dailyCheckInAnalysis(db, location)
		if err != nil {
			log.Printf("failed to analyze trends for location %s: %v", location, err)
			continue
		}
		// fmt.Printf("\n\n%s\n", result)
		// _, _ = analyzeTrends(result)
	}

	return nil
}

// FetchTodaysData retrieves today's data from the database and formats it as JSON for OpenAI
func fetchTodaysData(db *sql.DB, locationName string) (string, error) {
	todaysData := `
  select user_full_name, clocking_type, time_logged_rounded  
  from webhooks 
  WHERE date(datetime(time_logged)) = date('now')
   and location_name = ?
  order by time_logged_rounded;
`

	// Prepare the query
	rows, err := db.QueryContext(context.Background(), todaysData, locationName)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	var data []CheckInData

	// Iterate over the rows
	for rows.Next() {
		var row CheckInData
		err := rows.Scan(&row.Name, &row.Type, &row.Time)
		if err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}
		data = append(data, row)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("row iteration error: %w", err)
	}

	if len(data) == 0 {
		return "No data for today", nil
	}

	// Convert data to JSON for OpenAI
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Convert JSON data to a string for easy submission
	return string(dataJSON), nil
}

func analyzeTrends(jsonData string) (string, error) {
	// // Fetch today's data in JSON format for OpenAI
	// data, err := fetchTodaysData(db, locationName)
	// if err != nil {
	// 	log.Fatalf("failed to fetch today's data: %v", err)
	// }

	// Output the data (for testing purposes, here it just prints to console)
	// fmt.Println("Data for OpenAI:", data)

	apiKey := os.Getenv("OPENAI_API_KEY")
	apiURL := "https://api.openai.com/v1/chat/completions"

	// Create the prompt
	prompt := fmt.Sprintf(`
    Analyze the following check-in/check-out data
    Any check in after 8am or check out before 5pm should be highlighted in the output, which should be easy to read and not make too many observations.
    \n%v
    `, jsonData)

	requestBody, _ := json.Marshal(map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result OpenAIResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "No response from OpenAI", nil
	}

	log.Println(result.Choices[0].Message.Content)
	return result.Choices[0].Message.Content, nil
}
