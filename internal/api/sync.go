package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/magnusohle/openanki-backend/internal/auth"
	"github.com/magnusohle/openanki-backend/internal/database"
)

type SyncHandler struct{
    Repo *database.Repository
}

// Ensure SyncHandler implements ServerInterface
var _ ServerInterface = (*SyncHandler)(nil)

func RegisterSyncRoutes(r chi.Router, repo *database.Repository) {
	handler := &SyncHandler{Repo: repo}
    
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware)
        HandlerFromMux(handler, r)
    })
}

// GetSyncMeta returns user's current sync status
func (h *SyncHandler) GetSyncMeta(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	meta, err := h.Repo.GetSyncMeta(userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

    resp := SyncMeta{
        UserId:   &meta.UserID,
        Usn:      &meta.USN,
        LastSync: &meta.LastSync,
    }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PushSync receives client changes and applies them to server
func (h *SyncHandler) PushSync(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	user, err := database.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user.SubscriptionStatus == "free" {
		http.Error(w, "Sync requires a subscription", http.StatusForbidden)
		return
	}

	var req SyncPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
    
    // Convert API types to Database types
    payload := &database.SyncPayload{}
    
    if req.Decks != nil {
        for _, d := range *req.Decks {
            payload.Decks = append(payload.Decks, database.SyncDeck{
                ID:          d.Id,
                Name:        d.Name,
                Description: safeString(d.Description),
                ConfigID:    d.ConfigId,
                CreatedAt:   d.CreatedAt,
                ModifiedAt:  d.ModifiedAt,
                USN:         d.Usn,
            })
        }
    }
    
    if req.Notes != nil {
        for _, n := range *req.Notes {
            payload.Notes = append(payload.Notes, database.SyncNote{
                ID:    n.Id,
                GUID:  n.Guid,
                MID:   n.Mid,
                Mod:   n.Mod,
                USN:   n.Usn,
                Tags:  n.Tags,
                Flds:  n.Flds,
                Sfld:  n.Sfld,
                Csum:  n.Csum,
                Flags: n.Flags,
                Data:  n.Data,
            })
        }
    }
    
    if req.Cards != nil {
        for _, c := range *req.Cards {
            payload.Cards = append(payload.Cards, database.SyncCard{
                ID:             c.Id,
                NoteID:         c.NoteId,
                DeckID:         c.DeckId,
                Ordinal:        c.Ordinal,
                ModifiedAt:     c.ModifiedAt,
                USN:            c.Usn,
                State:          c.State,
                Queue:          c.Queue,
                Due:            c.Due,
                Interval:       c.Interval,
                EaseFactor:     c.EaseFactor,
                Reps:           c.Reps,
                Lapses:         c.Lapses,
                LeftCount:      c.LeftCount,
                OriginalDue:    c.OriginalDue,
                OriginalDeckID: c.OriginalDeckId,
                Flags:          c.Flags,
                Data:           c.Data,
            })
        }
    }
    
    if req.Graves != nil {
        for _, g := range *req.Graves {
            payload.Graves = append(payload.Graves, database.SyncGrave{
                OID:  g.Oid,
                Type: g.Type,
            })
        }
    }

	// Execute transactional sync
	newUSN, err := h.Repo.PushSyncSafe(userID, payload)
	if err != nil {
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return new USN
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(USNResponse{ServerUsn: &newUSN})
}

// PullSync returns changes since client's last USN
func (h *SyncHandler) PullSync(w http.ResponseWriter, r *http.Request, params PullSyncParams) {
	userID := r.Context().Value("user_id").(int)

	user, err := database.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user.SubscriptionStatus == "free" {
		http.Error(w, "Sync requires a subscription", http.StatusForbidden)
		return
	}

    since := 0
    if params.Since != nil {
        since = *params.Since
    }

	meta, err := h.Repo.GetSyncMeta(userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dbDecks, err := h.Repo.GetDecksSince(userID, since)
	if err != nil {
		http.Error(w, "Failed to get decks", http.StatusInternalServerError)
		return
	}
    
    decks := make([]SyncDeck, len(dbDecks))
    for i, d := range dbDecks {
        decks[i] = SyncDeck{
            Id:          d.ID,
            Name:        d.Name,
            Description: &d.Description,
            ConfigId:    d.ConfigID,
            CreatedAt:   d.CreatedAt,
            ModifiedAt:  d.ModifiedAt,
            Usn:         d.USN,
        }
    }

	dbNotes, err := h.Repo.GetNotesSince(userID, since)
	if err != nil {
		http.Error(w, "Failed to get notes", http.StatusInternalServerError)
		return
	}
    notes := make([]SyncNote, len(dbNotes))
    for i, n := range dbNotes {
        notes[i] = SyncNote{
            Id:    n.ID,
            Guid:  n.GUID,
            Mid:   n.MID,
            Mod:   n.Mod,
            Usn:   n.USN,
            Tags:  n.Tags,
            Flds:  n.Flds,
            Sfld:  n.Sfld,
            Csum:  n.Csum,
            Flags: n.Flags,
            Data:  n.Data,
        }
    }

	dbCards, err := h.Repo.GetCardsSince(userID, since)
	if err != nil {
		http.Error(w, "Failed to get cards", http.StatusInternalServerError)
		return
	}
    cards := make([]SyncCard, len(dbCards))
    for i, c := range dbCards {
        cards[i] = SyncCard{
            Id:             c.ID,
            NoteId:         c.NoteID,
            DeckId:         c.DeckID,
            Ordinal:        c.Ordinal,
            ModifiedAt:     c.ModifiedAt,
            Usn:            c.USN,
            State:          c.State,
            Queue:          c.Queue,
            Due:            c.Due,
            Interval:       c.Interval,
            EaseFactor:     c.EaseFactor,
            Reps:           c.Reps,
            Lapses:         c.Lapses,
            LeftCount:      c.LeftCount,
            OriginalDue:    c.OriginalDue,
            OriginalDeckId: c.OriginalDeckID,
            Flags:          c.Flags,
            Data:           c.Data,
        }
    }


	dbGraves, err := h.Repo.GetGravesSince(userID, since)
	if err != nil {
		http.Error(w, "Failed to get graves", http.StatusInternalServerError)
		return
	}
    graves := make([]SyncGrave, len(dbGraves))
    for i, g := range dbGraves {
        graves[i] = SyncGrave{
            Oid:  g.OID,
            Type: g.Type,
        }
    }

	resp := SyncPullResponse{
		ServerUsn: &meta.USN,
		Decks:     &decks,
		Notes:     &notes,
		Cards:     &cards,
		Graves:    &graves,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// FullSync handles initial sync or reset - client sends all data
func (h *SyncHandler) FullSync(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	user, err := database.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	if user.SubscriptionStatus == "free" {
		http.Error(w, "Sync requires a subscription", http.StatusForbidden)
		return
	}

	var req SyncPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// This is NOT using generated safe push because FullSync logic is slightly different (Delete data first).
    // But we can just use PushSyncSafe after delete!
    
	// Clear all existing data for this user
	if err := h.Repo.DeleteUserData(userID); err != nil {
		http.Error(w, "Failed to reset user data", http.StatusInternalServerError)
		return
	}

    // Convert and push
    payload := &database.SyncPayload{}
    if req.Decks != nil {
        for _, d := range *req.Decks {
            payload.Decks = append(payload.Decks, database.SyncDeck{
                ID:          d.Id,
                Name:        d.Name,
                Description: safeString(d.Description),
                ConfigID:    d.ConfigId,
                CreatedAt:   d.CreatedAt,
                ModifiedAt:  d.ModifiedAt,
                USN:         d.Usn,
            })
        }
    }
    // ... (Mapping others omitted for brevity, but needed in real code)
    // Actually we need to map all of them. reusing mapping logic would be smart.
    if req.Notes != nil {
        for _, n := range *req.Notes {
            payload.Notes = append(payload.Notes, database.SyncNote{
                ID: n.Id, GUID: n.Guid, MID: n.Mid, Mod: n.Mod, USN: n.Usn, Tags: n.Tags, Flds: n.Flds, Sfld: n.Sfld, Csum: n.Csum, Flags: n.Flags, Data: n.Data,
            })
        }
    }
    if req.Cards != nil {
        for _, c := range *req.Cards {
            payload.Cards = append(payload.Cards, database.SyncCard{
                ID: c.Id, NoteID: c.NoteId, DeckID: c.DeckId, Ordinal: c.Ordinal, ModifiedAt: c.ModifiedAt, USN: c.Usn, State: c.State, Queue: c.Queue, Due: c.Due, Interval: c.Interval, EaseFactor: c.EaseFactor, Reps: c.Reps, Lapses: c.Lapses, LeftCount: c.LeftCount, OriginalDue: c.OriginalDue, OriginalDeckID: c.OriginalDeckId, Flags: c.Flags, Data: c.Data,
            })
        }
    }
    
	newUSN, err := h.Repo.PushSyncSafe(userID, payload)
	if err != nil {
		http.Error(w, "Failed to push data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(USNResponse{ServerUsn: &newUSN})
}

// ListMedia returns all media hashes for a user
func (h *SyncHandler) ListMedia(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	rows, err := h.Repo.DB.Query(
		"SELECT hash, filename FROM user_media WHERE user_id = ?",
		userID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	media := []MediaItem{}
	for rows.Next() {
		var hash, filename string
		if err := rows.Scan(&hash, &filename); err == nil {
			media = append(media, MediaItem{Hash: &hash, Filename: &filename})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MediaListResponse{Media: &media})
}

// UploadMedia handles media file uploads
func (h *SyncHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	err := r.ParseMultipartForm(50 << 20) // 50MB limit
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

	hash := r.FormValue("hash")
	if hash == "" {
		http.Error(w, "Hash required", http.StatusBadRequest)
		return
	}

	// Create user media directory
	mediaDir := filepath.Join("./data/media", strconv.Itoa(userID))
	os.MkdirAll(mediaDir, 0755)

	// Save file with hash as filename
	filePath := filepath.Join(mediaDir, hash)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	// Record in database
	usn, _ := h.Repo.IncrementUSN(userID)
	h.Repo.DB.Exec(`
		INSERT OR REPLACE INTO user_media (user_id, filename, hash, size, usn)
		VALUES (?, ?, ?, ?, ?)
	`, userID, handler.Filename, hash, handler.Size, usn)

    status := "ok"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MediaUploadResponse{Status: &status, Hash: &hash})
}

// DownloadMedia serves a media file by hash
func (h *SyncHandler) DownloadMedia(w http.ResponseWriter, r *http.Request, hash string) {
	userID := r.Context().Value("user_id").(int)
    // hash passed as argument

	filePath := filepath.Join("./data/media", strconv.Itoa(userID), hash)

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get filename from database
	var filename string
	err = h.Repo.DB.QueryRow(
		"SELECT filename FROM user_media WHERE user_id = ? AND hash = ?",
		userID, hash,
	).Scan(&filename)
	if err != nil {
		filename = hash // fallback
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, file)
}

func safeString(s *string) string {
    if s == nil { return "" }
    return *s
}
