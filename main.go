package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
  "sync"

	_ "github.com/tursodatabase/go-libsql"
)

type Webhook struct {
	ID           string `json:"id"`
	Event        string `json:"event"`
	Data         Data   `json:"data"`
	Sequence     int    `json:"sequence"`
	DispatchedAt int64  `json:"dispatchedAt"`
}

type Data struct {
	DepartmentName       *string `json:"departmentName"`
	Comment              *string `json:"comment"`
	Color                *string `json:"color"`
	AbsenceTypeName      *string `json:"absenceTypeName"`
	AbsenceTypeID        *string `json:"absenceTypeId"`
	ProjectCode          *string `json:"projectCode"`
	ProjectName          *string `json:"projectName"`
	ProjectID            *string `json:"projectId"`
	UserEmployeeNumber   *string `json:"userEmployeeNumber"`
	DepartmentID         *string `json:"departmentId"`
	TimeZone             string  `json:"timeZone"`
	TimeInserted         string  `json:"timeInserted"`
	LocationName         string  `json:"locationName"`
	UserFullName         string  `json:"userFullName"`
	UserLastName         string  `json:"userLastName"`
	UserFirstName        string  `json:"userFirstName"`
	ClockingType         string  `json:"clockingType"`
	TimeLogged           string  `json:"timeLogged"`
	TimeLoggedRounded    string  `json:"timeLoggedRounded"`
	UserID               string  `json:"userId"`
	PayPeriodName        string  `json:"payPeriodName"`
	LocationID           string  `json:"locationId"`
	PayPeriodID          string  `json:"payPeriodId"`
	DeviceSerialNumber   string  `json:"deviceSerialNumber"`
	ClockingPairID       string  `json:"clockingPairId"`
	ClockingSequenceID   string  `json:"clockingSequenceId"`
	ClientID             string  `json:"clientId"`
	UserCardNumber       string  `json:"userCardNumber"`
	VerificationModeID   int     `json:"verificationModeId"`
	PlanningID           int     `json:"planningId"`
	RecordHash           int64   `json:"recordHash"`
	ClockingActionTypeID int     `json:"clockingActionTypeId"`
	RecordIgnored        bool    `json:"recordIgnored"`
}

const createTableQuery = `
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

const insertQuery = `
	INSERT OR REPLACE INTO webhooks (
		id, event, sequence, dispatched_at, clocking_type, device_serial_number, client_id, user_id, user_card_number, user_first_name,
		user_last_name, user_full_name, user_employee_number, department_id, department_name, location_id, location_name, project_id,
		project_name, project_code, time_zone, time_logged, time_logged_rounded, time_inserted, clocking_action_type_id, verification_mode_id,
		record_hash, record_ignored, clocking_pair_id, clocking_sequence_id, planning_id, absence_type_id, absence_type_name, color, comment,
		pay_period_id, pay_period_name
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

type WebhookPayload struct {
	Data map[string]interface{} `json:"data"`
}

// var webhookData []WebhookPayload

// const webhookSecret = "tmkey_gwCFPntotcNaH3454dEpayxn2uoPYvWU" // replace with the actual secret
var dbMutex sync.Mutex
func webhookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlePost(db, w, r)
		default:
			http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		}
	}
}

func simulateWebhook(db *sql.DB) {
	body := `{
		"id": "tm_0f9ab7fedb7e4cd7a4570e13f2526fed",
		"event": "attendance.inserted",
		"sequence": 42,
		"dispatchedAt": 1730912637,
		"data": {
			"clockingType": "Out",
			"deviceSerialNumber": "125058917500042",
			"clientId": "a1a1f9f2-e6b1-4c2f-94ec-a17c53731a8c",
			"userId": "1a595e44-8199-44de-ae9e-dae8c2c60cda",
			"userCardNumber": "5335288",
			"userFirstName": "Nikki",
			"userLastName": "Panayiotou",
			"userFullName": "Nikki Panayiotou",
			"userEmployeeNumber": null,
			"departmentId": null,
			"departmentName": null,
			"locationId": "e2a2da2d-2888-47e2-a231-f37088607aff",
			"locationName": "800 GROUP LTD",
			"projectId": null,
			"projectName": null,
			"projectCode": null,
			"timeZone": "Europe/London",
			"timeLogged": "2024-11-06T17:03:53",
			"timeLoggedRounded": "2024-11-06T17:03:00",
			"timeInserted": "2024-11-06T17:03:56.7124278Z",
			"clockingActionTypeId": 1,
			"verificationModeId": 4,
			"recordHash": -8652157066490000484,
			"recordIgnored": false,
			"clockingPairId": "Pairs_Sequences_a1a1f9f2-e6b1-4c2f-94ec-a17c53731a8c_1a595e44-8199-44de-ae9e-dae8c2c60cda_2024_11_06_7b6170fb-82b5-4af3-867c-1a8482a7d744",
			"clockingSequenceId": "Sequences_a1a1f9f2-e6b1-4c2f-94ec-a17c53731a8c_1a595e44-8199-44de-ae9e-dae8c2c60cda_2024_11_06",
			"planningId": 0,
			"absenceTypeId": null,
			"absenceTypeName": null,
			"color": null,
			"comment": null,
			"payPeriodId": "3b0124c5-6c1e-4242-89b3-2f8d5c953620",
			"payPeriodName": "45"
		}
	}`

	var webhook Webhook
	err := json.Unmarshal([]byte(body), &webhook)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}
	// log.Println("Webhook:", webhook)
	handleInsert(db, webhook)
}

func handleInsert(db *sql.DB, webhook Webhook) {
	_, err := db.Exec(insertQuery,
		webhook.ID,
		webhook.Event,
		webhook.Sequence,
		webhook.DispatchedAt,
		webhook.Data.ClockingType,
		webhook.Data.DeviceSerialNumber,
		webhook.Data.ClientID,
		webhook.Data.UserID,
		webhook.Data.UserCardNumber,
		webhook.Data.UserFirstName,
		webhook.Data.UserLastName,
		webhook.Data.UserFullName,
		nullString(webhook.Data.UserEmployeeNumber),
		nullString(webhook.Data.DepartmentID),
		nullString(webhook.Data.DepartmentName),
		webhook.Data.LocationID,
		webhook.Data.LocationName,
		nullString(webhook.Data.ProjectID),
		nullString(webhook.Data.ProjectName),
		nullString(webhook.Data.ProjectCode),
		webhook.Data.TimeZone,
		webhook.Data.TimeLogged,
		webhook.Data.TimeLoggedRounded,
		webhook.Data.TimeInserted,
		webhook.Data.ClockingActionTypeID,
		webhook.Data.VerificationModeID,
		webhook.Data.RecordHash,
		webhook.Data.RecordIgnored,
		webhook.Data.ClockingPairID,
		webhook.Data.ClockingSequenceID,
		webhook.Data.PlanningID,
		nullString(webhook.Data.AbsenceTypeID),
		nullString(webhook.Data.AbsenceTypeName),
		nullString(webhook.Data.Color),
		nullString(webhook.Data.Comment),
		webhook.Data.PayPeriodID,
		webhook.Data.PayPeriodName,
	)
	if err != nil {
		fmt.Println("ERROR WRITING SQL")
		log.Fatal(err)
	}
	fmt.Println("Record inserted successfully")
}

// Helper function to convert nil pointers to a sql.NullString
func nullString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func handlePost(db *sql.DB, w http.ResponseWriter, r *http.Request) {
  dbMutex.Lock()
  defer dbMutex.Unlock()
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
    log.Println("Failed to insert data")
    fmt.Println(err)
    http.Error(w, "Failed to insert data", http.StatusInternalServerError)
    return
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

// func verifySignature(body []byte, signature string) bool {
// 	expectedSignature := calculateExpectedSignature(body)
//
// 	// Strip "sha256=" prefix if present for comparison
// 	signatureNoPrefix := strings.TrimPrefix(signature, "sha256=")
// 	expectedSignatureNoPrefix := strings.TrimPrefix(expectedSignature, "sha256=")
//
// 	// Log the stripped values to help debug
// 	log.Printf("Expected Signature (no prefix): %s", expectedSignatureNoPrefix)
// 	log.Printf("Received Signature (no prefix): %s", signatureNoPrefix)
//
// 	// Compare all possible variants
// 	return hmac.Equal([]byte(signature), []byte(expectedSignature)) ||
// 		hmac.Equal([]byte(signatureNoPrefix), []byte(expectedSignature)) ||
// 		hmac.Equal([]byte(signature), []byte(expectedSignatureNoPrefix)) ||
// 		hmac.Equal([]byte(signatureNoPrefix), []byte(expectedSignatureNoPrefix))
// }

// func calculateExpectedSignature(body []byte) string {
// 	mac := hmac.New(sha256.New, []byte(webhookSecret))
// 	mac.Write(body)
// 	expectedMAC := mac.Sum(nil)
// 	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)
// 	return expectedSignature
// }

func main() {
	var db *sql.DB

	dbName := "file:.webhook.db?_journal_mode=WAL"

	db, err := sql.Open("libsql", dbName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s", err)
		return
	}
	defer db.Close()
	log.Printf("Creating table")
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal(err)
	}

	// simulateWebhook(db)

	http.HandleFunc("/webhook", webhookHandler(db))
	log.Println("Starting server on :443 with HTTPS...")
	log.Fatal(http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/group800.silverlininggroup.co.uk/fullchain.pem", "/etc/letsencrypt/live/group800.silverlininggroup.co.uk/privkey.pem", nil))
}
