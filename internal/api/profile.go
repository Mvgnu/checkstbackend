package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/magnusohle/openanki-backend/internal/auth"
	"github.com/magnusohle/openanki-backend/internal/database"
)

type ProfileHandler struct{}

func RegisterProfileRoutes(r chi.Router) {
	handler := &ProfileHandler{}
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Get("/me", handler.GetMyProfile)
		r.Put("/me", handler.UpdateMyProfile)
		r.Delete("/me", handler.DeleteMyAccount)
		r.Post("/upgrade-dev", handler.DevUpgrade) // Temporary
	})
}

// DevUpgrade simulates a successful purchase verification
func (h *ProfileHandler) DevUpgrade(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	query := `UPDATE users SET subscription_status = 'pro' WHERE id = ?`
	_, err := database.DB.Exec(query, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success", "message":"Upgraded to PRO"}`))
}

func (h *ProfileHandler) DeleteMyAccount(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	err := database.DeleteUser(userID)
	if err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Account deleted successfully"})
}

func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	user, err := database.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

type updateProfileRequest struct {
	AvatarURL  string `json:"avatar_url"`
	University string `json:"university"`
	Degree     string `json:"degree"`
}

func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := database.UpdateUser(userID, req.AvatarURL, req.University, req.Degree)
	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

    // Return updated user
    user, _ := database.GetUserByID(userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
