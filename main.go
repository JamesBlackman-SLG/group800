package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"bytes"
	"strings"

  _ "github.com/mattn/go-sqlite3"
)

type Webhook struct {
	ID           string `json:"id"`
	Event        string `json:"event"`
	Sequence     int    `json:"sequence"`
	DispatchedAt int64  `json:"dispatchedAt"`
	Data         Data   `json:"data"`
}

type Data struct {
	ClockingType         string  `json:"clockingType"`
	DeviceSerialNumber   string  `json:"deviceSerialNumber"`
	ClientID             string  `json:"clientId"`
	UserID               string  `json:"userId"`
	UserCardNumber       string  `json:"userCardNumber"`
	UserFirstName        string  `json:"userFirstName"`
	UserLastName         string  `json:"userLastName"`
	UserFullName         string  `json:"userFullName"`
	UserEmployeeNumber   *string `json:"userEmployeeNumber"`
	DepartmentID         *string `json:"departmentId"`
	DepartmentName       *string `json:"departmentName"`
	LocationID           string  `json:"locationId"`
	LocationName         string  `json:"locationName"`
	ProjectID            *string `json:"projectId"`
	ProjectName          *string `json:"projectName"`
	ProjectCode          *string `json:"projectCode"`
	TimeZone             string  `json:"timeZone"`
	TimeLogged           string  `json:"timeLogged"`
	TimeLoggedRounded    string  `json:"timeLoggedRounded"`
	TimeInserted         string  `json:"timeInserted"`
	ClockingActionTypeID int     `json:"clockingActionTypeId"`
	VerificationModeID   int     `json:"verificationModeId"`
	RecordHash           int64   `json:"recordHash"`
	RecordIgnored        bool    `json:"recordIgnored"`
	ClockingPairID       string  `json:"clockingPairId"`
	ClockingSequenceID   string  `json:"clockingSequenceId"`
	PlanningID           int     `json:"planningId"`
	AbsenceTypeID        *string `json:"absenceTypeId"`
	AbsenceTypeName      *string `json:"absenceTypeName"`
	Color                *string `json:"color"`
	Comment              *string `json:"comment"`
	PayPeriodID          string  `json:"payPeriodId"`
	PayPeriodName        string  `json:"payPeriodName"`
}

type WebhookPayload struct {
	Data map[string]interface{} `json:"data"`
}

var webhookData []WebhookPayload

const webhookSecret = "tmkey_gwCFPntotcNaH3454dEpayxn2uoPYvWU" // replace with the actual secret

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handlePost(w, r)
	case http.MethodGet:
		handleGet(w, r)
	case http.MethodPut:
		handlePut(w, r)
	case http.MethodDelete:
		handleDelete(w, r)
	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	// log.Println("POST - Request received")

	// Print all headers
	// for key, values := range r.Header {
	// 	for _, value := range values {
	// 		log.Printf("Header: %s=%s", key, value)
	// 	}
	// }

	// Retrieve the signature from the header
	signature := strings.TrimSpace(r.Header.Get("Timemoto-Signature"))
	if signature == "" {
		http.Error(w, "Missing signature", http.StatusForbidden)
		return
	}

	// Read and keep a copy of the body for reuse
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read request body", http.StatusInternalServerError)
		return
	}

	// Log the raw body to help understand what's being hashed
	log.Printf("%s", string(body))

  var webhook Webhook
	err = json.Unmarshal([]byte(body), &webhook)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

  log.Printf("id = %s", webhook.ID)


	_, err = db.Exec(insertQuery,
		webhook.ID, webhook.Event, webhook.Sequence, webhook.DispatchedAt, webhook.Data.ClockingType, webhook.Data.DeviceSerialNumber,
		webhook.Data.ClientID, webhook.Data.UserID, webhook.Data.UserCardNumber, webhook.Data.UserFirstName, webhook.Data.UserLastName,
		webhook.Data.UserFullName, webhook.Data.UserEmployeeNumber, webhook.Data.DepartmentID, webhook.Data.DepartmentName,
		webhook.Data.LocationID, webhook.Data.LocationName, webhook.Data.ProjectID, webhook.Data.ProjectName, webhook.Data.ProjectCode,
		webhook.Data.TimeZone, webhook.Data.TimeLogged, webhook.Data.TimeLoggedRounded, webhook.Data.TimeInserted, webhook.Data.ClockingActionTypeID,
		webhook.Data.VerificationModeID, webhook.Data.RecordHash, webhook.Data.RecordIgnored, webhook.Data.ClockingPairID,
		webhook.Data.ClockingSequenceID, webhook.Data.PlanningID, webhook.Data.AbsenceTypeID, webhook.Data.AbsenceTypeName, webhook.Data.Color,
		webhook.Data.Comment, webhook.Data.PayPeriodID, webhook.Data.PayPeriodName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Record inserted successfully")

	// Verify the signature
	// if !verifySignature(body, signature) {
	// 	log.Printf("Signature verification failed")
	// 	// log.Printf("Expected Signature: %s", calculateExpectedSignature(body))
	// 	// log.Printf("Received Signature: %s", signature)
	// 	// http.Error(w, "Invalid signature", http.StatusForbidden)
	// 	return
	// }

	// Reset the request body so it can be read again
	// r.Body = io.NopCloser(bytes.NewBuffer(body))

	// Decode the payload
	// var payload WebhookPayload
	// err = json.NewDecoder(r.Body).Decode(&payload)
	// if err != nil {
		// http.Error(w, "Invalid payload", http.StatusBadRequest)
		// return
	// }

	// Store the payload in memory
	// webhookData = append(webhookData, payload)
	// fmt.Fprintln(w, "POST webhook received successfully")
	// log.Printf("POST - Received webhook data: %+v\n", payload)
}

func verifySignature(body []byte, signature string) bool {
	expectedSignature := calculateExpectedSignature(body)

	// Strip "sha256=" prefix if present for comparison
	signatureNoPrefix := strings.TrimPrefix(signature, "sha256=")
	expectedSignatureNoPrefix := strings.TrimPrefix(expectedSignature, "sha256=")

	// Log the stripped values to help debug
	log.Printf("Expected Signature (no prefix): %s", expectedSignatureNoPrefix)
	log.Printf("Received Signature (no prefix): %s", signatureNoPrefix)

	// Compare all possible variants
	return hmac.Equal([]byte(signature), []byte(expectedSignature)) ||
		hmac.Equal([]byte(signatureNoPrefix), []byte(expectedSignature)) ||
		hmac.Equal([]byte(signature), []byte(expectedSignatureNoPrefix)) ||
		hmac.Equal([]byte(signatureNoPrefix), []byte(expectedSignatureNoPrefix))
}

func calculateExpectedSignature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)
	return expectedSignature
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET - Request received with URL: %s\n", r.URL.String())
	fmt.Fprintf(w, "GET request received at URL: %s\n", r.URL.String())
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	log.Printf("PUT - Received data: %s\n", string(body))
	fmt.Fprintln(w, "PUT request received successfully")
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	log.Printf("DELETE - Request received with URL: %s\n", r.URL.String())
	fmt.Fprintf(w, "DELETE request received at URL: %s\n", r.URL.String())
}

func main() {
  db, err := sql.Open("sqlite3", "./webhooks.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS webhooks (
		id TEXT,
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

	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal(err)
	}


	insertQuery := `
	INSERT INTO webhooks (
		id, event, sequence, dispatched_at, clocking_type, device_serial_number, client_id, user_id, user_card_number, user_first_name,
		user_last_name, user_full_name, user_employee_number, department_id, department_name, location_id, location_name, project_id,
		project_name, project_code, time_zone, time_logged, time_logged_rounded, time_inserted, clocking_action_type_id, verification_mode_id,
		record_hash, record_ignored, clocking_pair_id, clocking_sequence_id, planning_id, absence_type_id, absence_type_name, color, comment,
		pay_period_id, pay_period_name
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	http.HandleFunc("/webhook", webhookHandler)
	log.Println("Starting server on :443 with HTTPS...")
	log.Fatal(http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/group800.silverlininggroup.co.uk/fullchain.pem", "/etc/letsencrypt/live/group800.silverlininggroup.co.uk/privkey.pem", nil))
}

