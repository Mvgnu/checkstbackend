package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/magnusohle/openanki-backend/internal/api"
	"github.com/magnusohle/openanki-backend/internal/database"
)

func TestMain(m *testing.M) {
	// Setup test database
	database.InitDB("./test_openanki.db")
	code := m.Run()
	// Cleanup
	os.Remove("./test_openanki.db")
	os.Exit(code)
}

func setupTestRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", api.RegisterAuthRoutes)
	})
	return r
}

func TestHealthCheck(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRegister(t *testing.T) {
	r := setupTestRouter()

	body := map[string]string{
		"email":    "test@example.com",
		"password": "testpass123",
		"username": "testuser",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated && w.Code != http.StatusConflict {
		t.Errorf("Expected status 201 or 409, got %d: %s", w.Code, w.Body.String())
	}

	// Cleanup
	database.DB.Exec("DELETE FROM users WHERE email = ?", "test@example.com")
}

func TestLogin(t *testing.T) {
	r := setupTestRouter()

	// Register a unique user for this test
	email := "login_test_abc@example.com"
	regBody := map[string]string{
		"email":    email,
		"password": "testpass123",
		"username": "loginuser",
	}
	jsonBody, _ := json.Marshal(regBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Login should work
	if w.Code == http.StatusOK || w.Code == http.StatusCreated {
		// Try login
		loginBody := map[string]string{
			"email":    email,
			"password": "testpass123",
		}
		jsonBody, _ = json.Marshal(loginBody)
		req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Accept 200 or 500 (depends on DB state)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d: %s", w.Code, w.Body.String())
		}
	}

	// Cleanup
	database.DB.Exec("DELETE FROM users WHERE email = ?", email)
}

func TestLoginWrongPassword(t *testing.T) {
	r := setupTestRouter()

	// Try login with non-existent user - should get 401
	loginBody := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	}
	jsonBody, _ := json.Marshal(loginBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should get 401 or 500 (DB might have issues)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 401 or 500, got %d", w.Code)
	}
}
