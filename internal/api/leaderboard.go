package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/magnusohle/openanki-backend/internal/auth"
	"github.com/magnusohle/openanki-backend/internal/database"
)

type LeaderboardHandler struct{}

func RegisterLeaderboardRoutes(r chi.Router) {
	handler := &LeaderboardHandler{}
	r.Get("/global", handler.GetGlobalLeaderboard)
	r.Get("/group/{groupId}", handler.GetGroupLeaderboard)
	r.With(auth.Middleware).Post("/update", handler.UpdateStats)
}

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	XP       int    `json:"xp"`
	Level    int    `json:"level"`
	Streak   int    `json:"streak"`
}

func (h *LeaderboardHandler) GetGlobalLeaderboard(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period") // "week", "month", "all"
	if period == "" {
		period = "week"
	}
	
	limit := 10
	
	entries, err := database.GetLeaderboard(period, limit)
	if err != nil {
		http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *LeaderboardHandler) GetGroupLeaderboard(w http.ResponseWriter, r *http.Request) {
	groupIdStr := chi.URLParam(r, "groupId")
	groupId, _ := strconv.Atoi(groupIdStr)
	
	// Get user from token to verify membership
	userId := r.Context().Value("user_id").(int)
	
	isMember, _ := database.IsMember(groupId, userId)
	if !isMember {
		http.Error(w, "Not a member of this group", http.StatusForbidden)
		return
	}
	
	entries, err := database.GetGroupLeaderboard(groupIdStr, 10)
	if err != nil {
		http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// UpdateStats receives XP updates from the client after review sessions
func (h *LeaderboardHandler) UpdateStats(w http.ResponseWriter, r *http.Request) {
	userId := r.Context().Value("user_id").(int)
	
	var req struct {
		CardsReviewed    int `json:"cards_reviewed"`
		XPEarned         int `json:"xp_earned"`
		TimeSpentSeconds int `json:"time_spent_seconds"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Update user XP in database
	err := database.AddUserXP(userId, req.XPEarned)
	if err != nil {
		http.Error(w, "Failed to update stats", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"xp_added": req.XPEarned,
	})
}
