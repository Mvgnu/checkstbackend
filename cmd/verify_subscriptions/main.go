package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const baseURL = "http://localhost:8080/api/v1"

type AuthResponse struct {
	Token string `json:"token"`
}

func main() {
	// 1. Connect to DB to manipulate subscription status
	home, _ := os.UserHomeDir()
	dbPath := home + "/.checkst/openanki.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// 1.5 Ensure Schema Migration (Quick Fix for Dev)
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN subscription_status TEXT DEFAULT 'free'")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN subscription_expiry DATETIME")

	tests := []struct {
		username   string
		email      string
		password   string
		subStatus  string
		expectCode int
	}{
		{"test_free", "free@checkst.app", "password123", "free", 403},
		{"test_pro", "pro@checkst.app", "password123", "pro", 200},
		{"test_host", "host@checkst.app", "password123", "group_host", 200},
	}

	fmt.Println("Starting Subscription Verification...")

	for _, t := range tests {
		fmt.Printf("\nTesting User: %s (%s)\n", t.username, t.subStatus)

		// A. Register (ignore error if exists)
		apiCall("POST", "/auth/register", map[string]string{
			"email":    t.email,
			"password": t.password,
			"username": t.username,
		}, "")

		// B. Login
		respBody, code := apiCall("POST", "/auth/login", map[string]string{
			"email":    t.email,
			"password": t.password,
		}, "")

		if code != 200 {
			log.Fatalf("Login failed for %s: %d", t.username, code)
		}

		var authResp AuthResponse
		json.Unmarshal([]byte(respBody), &authResp)
		token := authResp.Token

		// C. Update Subscription in DB
		_, err := db.Exec("UPDATE users SET subscription_status = ? WHERE email = ?", t.subStatus, t.email)
		if err != nil {
			log.Fatalf("Failed to update DB: %v", err)
		}
		fmt.Printf("  -> Updated DB status to '%s'\n", t.subStatus)

		// D. Verify Access to Gated Endpoint (Sync Push)
		// We send an empty push request
		pushReq := map[string]interface{}{
			"client_usn": 0,
			"decks":      []interface{}{},
			"notes":      []interface{}{},
			"cards":      []interface{}{},
		}
		_, pushCode := apiCall("POST", "/sync/push", pushReq, token)

		if pushCode == t.expectCode {
			fmt.Printf("  ✅ Success: Got expected status %d\n", pushCode)
		} else {
			fmt.Printf("  ❌ FAILURE: Expected %d, got %d\n", t.expectCode, pushCode)
			// Don't exit, try others
		}
	}
}

func apiCall(method, endpoint string, body interface{}, token string) (string, int) {
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, baseURL+endpoint, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	return string(respBytes), resp.StatusCode
}
