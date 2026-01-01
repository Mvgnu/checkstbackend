package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/magnusohle/openanki-backend/internal/database"
)

type ProgressHandler struct{}

func RegisterProgressRoutes(r chi.Router) {
	handler := &ProgressHandler{}
	r.Get("/", handler.GetProgress)
	r.Put("/", handler.UpdateProgress)
	r.Post("/sync", handler.SyncProgress)
}

type UserProgress struct {
	XP                    int      `json:"xp"`
	Level                 int      `json:"level"`
	Streak                int      `json:"streak"`
	TotalReviews          int      `json:"total_reviews"`
	CardsLearned          int      `json:"cards_learned"`
	UnlockedAchievements  []string `json:"unlocked_achievements"`
	LastReviewAt          int64    `json:"last_review_at"`
}

func (h *ProgressHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("user_id").(int)
	
	progress, err := database.GetUserProgress(userId)
	if err != nil {
		http.Error(w, "Failed to get progress", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

func (h *ProgressHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("user_id").(int)
	
	var progress UserProgress
	if err := json.NewDecoder(r.Body).Decode(&progress); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if err := database.UpdateUserProgress(userId, progress.XP, progress.Level, progress.Streak); err != nil {
		http.Error(w, "Failed to update progress", http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *ProgressHandler) SyncProgress(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("user_id").(int)
	
	var clientProgress UserProgress
	if err := json.NewDecoder(r.Body).Decode(&clientProgress); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Get server progress
	serverProgress, err := database.GetUserProgress(userId)
	if err != nil {
		// First sync - just save client data
		database.UpdateUserProgress(userId, clientProgress.XP, clientProgress.Level, clientProgress.Streak)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clientProgress)
		return
	}
	
	// Merge: take highest values
	merged := UserProgress{
		XP:           max(clientProgress.XP, serverProgress.XP),
		Level:        max(clientProgress.Level, serverProgress.Level),
		Streak:       max(clientProgress.Streak, serverProgress.Streak),
		TotalReviews: max(clientProgress.TotalReviews, serverProgress.TotalReviews),
		CardsLearned: max(clientProgress.CardsLearned, serverProgress.CardsLearned),
	}
	
	// Merge achievements (union)
	achievementSet := make(map[string]bool)
	for _, a := range clientProgress.UnlockedAchievements {
		achievementSet[a] = true
	}
	for _, a := range serverProgress.UnlockedAchievements {
		achievementSet[a] = true
	}
	for a := range achievementSet {
		merged.UnlockedAchievements = append(merged.UnlockedAchievements, a)
	}
	
	// Save merged progress
	database.UpdateUserProgress(userId, merged.XP, merged.Level, merged.Streak)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(merged)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
