package api

import (
    "encoding/json"
    "net/http"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/magnusohle/openanki-backend/internal/auth"
    "github.com/magnusohle/openanki-backend/internal/database"
    "github.com/magnusohle/openanki-backend/internal/media"
)

type GroupsHandler struct{}

func RegisterGroupsRoutes(r chi.Router) {
    handler := &GroupsHandler{}
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware)
        r.Post("/", handler.CreateGroup)
        r.Get("/", handler.ListGroups)
        r.Post("/{id}/join", handler.JoinGroup)
        r.Post("/join", handler.JoinWithCode) // New endpoint for code-based join
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

func (h *GroupsHandler) JoinWithCode(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    
    var req struct {
        Code string `json:"code"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
         http.Error(w, "Invalid request", http.StatusBadRequest)
         return
    }
    
    if req.Code == "" {
        http.Error(w, "Code required", http.StatusBadRequest)
        return
    }
    
    group, err := database.JoinGroupByCode(req.Code, userID)
    if err != nil {
        http.Error(w, "Failed to join group (invalid code or error)", http.StatusBadRequest)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(group)
}

// UploadDeck - share a deck to the group (Step 1: Get Presigned URL)
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
        // DeckData removed - client uploads directly to R2
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Generate unique R2 key
    // Format: groups/{groupID}/decks/{uuid}.apkg
    uuid := database.GenerateRandomString(12) // Reusing existing helper or just simple random
    r2Key := "groups/" + strconv.Itoa(groupID) + "/decks/" + uuid + ".apkg"
    
    // Generate Presigned PUT URL
    s3Service, err := media.NewS3Service()
    if err != nil {
        http.Error(w, "Storage unavailable", http.StatusInternalServerError)
        return
    }
    
    uploadURL, err := s3Service.GeneratePresignedPutURL(r2Key, "application/octet-stream", 15*time.Minute) // 15 min expiry
    if err != nil {
        http.Error(w, "Failed to generate upload URL", http.StatusInternalServerError)
        return
    }
    
    // Create DB Record (optimistic creation, or we could do it after - but optimisitc is simpler for client flow)
    deck, err := database.CreateGroupDeck(groupID, userID, req.Name, req.CardCount, r2Key)
    if err != nil {
        http.Error(w, "Failed to create deck record", http.StatusInternalServerError)
        return
    }
    
    // Return URL + Deck Info
    resp := struct {
        UploadURL string              `json:"upload_url"`
        Deck      *database.GroupDeck `json:"deck"`
    }{
        UploadURL: uploadURL,
        Deck:      deck,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
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

// DownloadDeck - get download URL for a deck
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

    // Generate Presigned GET URL
    s3Service, err := media.NewS3Service()
    if err != nil {
        http.Error(w, "Storage unavailable", http.StatusInternalServerError)
        return
    }
    
    downloadURL, err := s3Service.GeneratePresignedGetURL(deck.R2Key, 1*time.Hour) // 1 hour expiry
    if err != nil {
        http.Error(w, "Failed to generate download URL", http.StatusInternalServerError)
        return
    }
    
    // Return URL
    resp := struct {
        DownloadURL string `json:"download_url"`
        Name        string `json:"name"`
        CardCount   int    `json:"card_count"`
    }{
        DownloadURL: downloadURL,
        Name:        deck.Name,
        CardCount:   deck.CardCount,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
