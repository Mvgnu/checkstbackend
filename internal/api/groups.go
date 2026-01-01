package api

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
    "github.com/magnusohle/openanki-backend/internal/auth"
    "github.com/magnusohle/openanki-backend/internal/database"
)

type GroupsHandler struct{}

func RegisterGroupsRoutes(r chi.Router) {
    handler := &GroupsHandler{}
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware)
        r.Post("/", handler.CreateGroup)
        r.Get("/", handler.ListGroups)
        r.Post("/{id}/join", handler.JoinGroup)
        // Deck sharing
        r.Post("/{id}/decks", handler.UploadDeck)
        r.Get("/{id}/decks", handler.ListGroupDecks)
        r.Get("/{id}/decks/{deckId}", handler.DownloadDeck)
    })
}

type createGroupRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    University  string `json:"university"`
    Degree      string `json:"degree"`
}

func (h *GroupsHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    var req createGroupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    if req.Name == "" {
        http.Error(w, "Group name required", http.StatusBadRequest)
        return
    }

    group, err := database.CreateGroup(req.Name, req.Description, req.University, req.Degree, userID)
    if err != nil {
        http.Error(w, "Failed to create group", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(group)
}

func (h *GroupsHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
    uni := r.URL.Query().Get("university")
    degree := r.URL.Query().Get("degree")
    
    // Get user ID if authenticated (optional for listing)
    userID := 0
    if uid := r.Context().Value("user_id"); uid != nil {
        userID = uid.(int)
    }

    groups, err := database.ListGroupsWithMembership(uni, degree, userID)
    if err != nil {
        http.Error(w, "Failed to list groups", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(groups)
}

func (h *GroupsHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    groupIDStr := chi.URLParam(r, "id")
    groupID, err := strconv.Atoi(groupIDStr)
    if err != nil {
        http.Error(w, "Invalid Group ID", http.StatusBadRequest)
        return
    }

    // Check if already member
    isMember, _ := database.IsMember(groupID, userID)
    if isMember {
        http.Error(w, "Already a member", http.StatusConflict) // Or just 200 OK
        return
    }

    if err := database.JoinGroup(groupID, userID); err != nil {
         http.Error(w, "Failed to join group", http.StatusInternalServerError)
         return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"joined"}`))
}

// UploadDeck - share a deck to the group
func (h *GroupsHandler) UploadDeck(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    groupIDStr := chi.URLParam(r, "id")
    groupID, _ := strconv.Atoi(groupIDStr)
    
    // Check membership
    isMember, _ := database.IsMember(groupID, userID)
    if !isMember {
        http.Error(w, "Not a member of this group", http.StatusForbidden)
        return
    }

	// Check subscription (Owner Pays)
	user, err := database.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user.SubscriptionStatus == "free" {
		http.Error(w, "Group uploads require a subscription", http.StatusForbidden)
		return
	}
    
    var req struct {
        Name      string `json:"name"`
        CardCount int    `json:"card_count"`
        DeckData  string `json:"deck_data"` // JSON blob of deck content
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    deck, err := database.CreateGroupDeck(groupID, userID, req.Name, req.CardCount, req.DeckData)
    if err != nil {
        http.Error(w, "Failed to upload deck", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(deck)
}

// ListGroupDecks - get all shared decks in a group
func (h *GroupsHandler) ListGroupDecks(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    groupIDStr := chi.URLParam(r, "id")
    groupID, _ := strconv.Atoi(groupIDStr)
    
    isMember, _ := database.IsMember(groupID, userID)
    if !isMember {
        http.Error(w, "Not a member of this group", http.StatusForbidden)
        return
    }
    
    decks, err := database.ListGroupDecks(groupID)
    if err != nil {
        http.Error(w, "Failed to list decks", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(decks)
}

// DownloadDeck - get a specific deck's content
func (h *GroupsHandler) DownloadDeck(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    groupIDStr := chi.URLParam(r, "id")
    groupID, _ := strconv.Atoi(groupIDStr)
    deckIDStr := chi.URLParam(r, "deckId")
    deckID, _ := strconv.Atoi(deckIDStr)
    
    isMember, _ := database.IsMember(groupID, userID)
    if !isMember {
        http.Error(w, "Not a member of this group", http.StatusForbidden)
        return
    }
    
    deck, err := database.GetGroupDeck(deckID)
    if err != nil {
        http.Error(w, "Deck not found", http.StatusNotFound)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(deck)
}
