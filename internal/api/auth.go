package api

import (
    "encoding/json"
    "net/http"
    "crypto/sha256"
    "encoding/hex"

    "github.com/go-chi/chi/v5"
    "github.com/magnusohle/openanki-backend/internal/auth"
    "github.com/magnusohle/openanki-backend/internal/database"
    "github.com/magnusohle/openanki-backend/internal/mailer"
    "math/rand"
    "time"
)

// Simple hash implementation (replace with bcrypt in production if dependencies allow)
func hashPassword(password string) string {
    hash := sha256.Sum256([]byte(password))
    return hex.EncodeToString(hash[:])
}

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude confusing chars 0,O,1,I

func generateResetCode() string {
    seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
    b := make([]byte, 6)
    for i := range b {
        b[i] = charset[seededRand.Intn(len(charset))]
    }
    return string(b)
}

type AuthHandler struct{}

func RegisterAuthRoutes(r chi.Router) {
    handler := &AuthHandler{}
    r.Post("/register", handler.Register)
    r.Post("/login", handler.Login)
    r.Post("/forgot-password", handler.ForgotPassword)
    r.Post("/reset-password", handler.ResetPassword)
}

// ... existing Register/Login ...

// ForgotPassword handles requesting a reset code
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email string `json:"email"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    if req.Email == "" {
        http.Error(w, "Email required", http.StatusBadRequest)
        return
    }

    // Verify user exists
    user, err := database.GetUserByEmail(req.Email)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    if user == nil {
        // Return OK to prevent email enumeration? 
        // User asked for "easiest", but security best practice is to lie.
        // But for UX, knowing it failed is helpful if they typod.
        // I'll return OK but not send email.
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"message": "If account exists, email sent"})
        return
    }

    code := generateResetCode()
    if err := database.SaveResetCode(req.Email, code); err != nil {
        http.Error(w, "Failed to save code", http.StatusInternalServerError)
        return
    }

    // Send Email
    if err := mailer.SendResetEmail(req.Email, code); err != nil {
        // Log error but don't fail request to client?
        // Or fail so they can retry.
        http.Error(w, "Failed to send email: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Email sent"})
}

// ResetPassword handles verifying code and setting new password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email       string `json:"email"`
        Code        string `json:"code"`
        NewPassword string `json:"newPassword"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    if req.Email == "" || req.Code == "" || req.NewPassword == "" {
        http.Error(w, "Missing fields", http.StatusBadRequest)
        return
    }

    if !database.VerifyAndConsumeResetCode(req.Email, req.Code) {
        http.Error(w, "Invalid or expired code", http.StatusBadRequest)
        return
    }

    hashedPwd := hashPassword(req.NewPassword)
    if err := database.UpdateUserPassword(req.Email, hashedPwd); err != nil {
        http.Error(w, "Failed to update password", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
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
