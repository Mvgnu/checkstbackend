package api

import (
    "encoding/json"
    "net/http"
    "crypto/sha256"
    "encoding/hex"

    "github.com/go-chi/chi/v5"
    "github.com/magnusohle/openanki-backend/internal/auth"
    "github.com/magnusohle/openanki-backend/internal/database"
)

// Simple hash implementation (replace with bcrypt in production if dependencies allow)
func hashPassword(password string) string {
    hash := sha256.Sum256([]byte(password))
    return hex.EncodeToString(hash[:])
}

type AuthHandler struct{}

func RegisterAuthRoutes(r chi.Router) {
    handler := &AuthHandler{}
    r.Post("/register", handler.Register)
    r.Post("/login", handler.Login)
}

type registerRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    Username string `json:"username"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req registerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if req.Email == "" || req.Password == "" || req.Username == "" {
        http.Error(w, "Missing fields", http.StatusBadRequest)
        return
    }

    hashedPwd := hashPassword(req.Password)
    user, err := database.CreateUser(req.Email, hashedPwd, req.Username)
    if err != nil {
        // Simplified error handling
        http.Error(w, "Failed to create user (email/username might be taken)", http.StatusConflict)
        return
    }

    token, _ := auth.GenerateToken(user.ID, user.Email)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "user": user,
        "token": token,
    })
}

type loginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req loginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    user, err := database.GetUserByEmail(req.Email)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    if user == nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    if hashPassword(req.Password) != user.PasswordHash {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    token, _ := auth.GenerateToken(user.ID, user.Email)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "user": user,
        "token": token,
    })
}
