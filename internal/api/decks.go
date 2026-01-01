package api

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/magnusohle/openanki-backend/internal/auth"
    "github.com/magnusohle/openanki-backend/internal/database"
)

type DecksHandler struct{}

func RegisterDecksRoutes(r chi.Router) {
    handler := &DecksHandler{}
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware)
        r.Post("/upload", handler.UploadDeck)
        r.Get("/group/{groupID}", handler.ListGroupDecks)
        r.Get("/{id}/download", handler.DownloadDeck)
    })
}


func (h *DecksHandler) UploadDeck(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(int)
    
    // Parse multipart form
    err := r.ParseMultipartForm(10 << 20) // 10MB limit
    if err != nil {
        http.Error(w, "File too large", http.StatusBadRequest)
        return
    }

    file, handler, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Error retrieving file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    title := r.FormValue("title")
    description := r.FormValue("description")
    groupIDStr := r.FormValue("group_id")
    
    if title == "" || groupIDStr == "" {
        http.Error(w, "Title and Group ID required", http.StatusBadRequest)
        return
    }

    groupID, _ := strconv.Atoi(groupIDStr)
    // Verify membership
    isMember, _ := database.IsMember(groupID, userID)
    if !isMember {
         http.Error(w, "Not a member of this group", http.StatusForbidden)
         return
    }

    // Save File
    uploadDir := "./data/uploads"
    os.MkdirAll(uploadDir, 0755)
    
    filename := fmt.Sprintf("%d_%d_%s", userID, time.Now().Unix(), handler.Filename)
    filePath := filepath.Join(uploadDir, filename)
    
    dst, err := os.Create(filePath)
    if err != nil {
        http.Error(w, "Failed to save file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()
    io.Copy(dst, file)

    // Save Metadata
    deck, err := database.CreateSharedDeck(title, description, filePath, userID, &groupID)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(deck)
}

func (h *DecksHandler) ListGroupDecks(w http.ResponseWriter, r *http.Request) {
    groupIDStr := chi.URLParam(r, "groupID")
    groupID, _ := strconv.Atoi(groupIDStr)

    decks, err := database.ListSharedDecks(groupID)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(decks)
}

func (h *DecksHandler) DownloadDeck(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, _ := strconv.Atoi(idStr)

    deck, err := database.GetSharedDeck(id)
    if err != nil || deck == nil {
        http.Error(w, "Deck not found", http.StatusNotFound)
        return
    }

    // Check permissions? Assuming if you have link or are in group you can download.
    // Ideally check if user is in deck.GroupID
    
    f, err := os.Open(deck.FilePath)
    if err != nil {
        http.Error(w, "File not found on server", http.StatusInternalServerError)
        return
    }
    defer f.Close()

    database.IncrementDownloads(id)
    
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.apkg\"", deck.Title))
    w.Header().Set("Content-Type", "application/octet-stream")
    io.Copy(w, f)
}
