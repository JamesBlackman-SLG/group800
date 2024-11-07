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
)

type CheckInData struct {
	Name string `json:"name"`
	// LocationName      string `json:"locationName"`
	Type string `json:"type"`
	Time string `json:"time"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
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

func analyzeTrends(db *sql.DB, locationName string) (string, error) {
	// Fetch today's data in JSON format for OpenAI
	data, err := fetchTodaysData(db, locationName)
	if err != nil {
		log.Fatalf("failed to fetch today's data: %v", err)
	}

	// Output the data (for testing purposes, here it just prints to console)
	fmt.Println("Data for OpenAI:", data)

	apiKey := os.Getenv("OPENAI_API_KEY")
	apiURL := "https://api.openai.com/v1/chat/completions"

	// Create the prompt
	prompt := fmt.Sprintf(`
    Analyze the following check-in/check-out data for trends and lateness.
    Any check in after 8am or check out before 5pm should be highlighted in the output, which should be easy to read and not make too many observations.
    \n%v
    `, data)

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

	log.Println("OpenAI response:", result.Choices[0].Message.Content)
	return result.Choices[0].Message.Content, nil
}
