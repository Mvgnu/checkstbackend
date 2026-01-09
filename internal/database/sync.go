package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"
)

// SyncMeta holds user sync status
type SyncMeta struct {
	UserID   int       `json:"user_id"`
	USN      int       `json:"usn"`
	LastSync time.Time `json:"last_sync"`
}

// SyncDeck represents a synced deck
type SyncDeck struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ConfigID    int    `json:"config_id"`
	CreatedAt   int64  `json:"created_at"`
	ModifiedAt  int64  `json:"modified_at"`
	USN         int    `json:"usn"`
}

// SyncNote represents a synced note
type SyncNote struct {
	ID    int64  `json:"id"`
	GUID  string `json:"guid"`
	MID   int64  `json:"mid"`
	Mod   int64  `json:"mod"`
	USN   int    `json:"usn"`
	Tags  string `json:"tags"`
	Flds  string `json:"flds"`
	Sfld  string `json:"sfld"`
	Csum  int64  `json:"csum"`
	Flags int    `json:"flags"`
	Data  string `json:"data"`
}

// SyncPayload represents a collection of data to be synced
type SyncPayload struct {
    Decks  []SyncDeck
    Notes  []SyncNote
    Cards  []SyncCard
    Graves []SyncGrave
}

// PushSyncSafe performs a transactional sync push
func (r *Repository) PushSyncSafe(userID int, payload *SyncPayload) (int, error) {
    ctx := context.Background()
    tx, err := r.DB.BeginTx(ctx, nil)
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    qtx := r.Q.WithTx(tx)
    
    // Ensure meta exists
    if err := qtx.CreateSyncMeta(ctx, int64(userID)); err != nil {
        return 0, err
    }
    
    // Increment USN
    usnInt64, err := qtx.UpdateUSN(ctx, int64(userID))
    if err != nil {
        return 0, err
    }
    usn := int64(usnInt64)
    
    // Apply Decks
    for _, deck := range payload.Decks {
        err := qtx.UpsertDeck(ctx, UpsertDeckParams{
            ID:          deck.ID,
            UserID:      int64(userID),
            Name:        deck.Name,
            Description: sql.NullString{String: deck.Description, Valid: deck.Description != ""},
            ConfigID:    sql.NullInt64{Int64: int64(deck.ConfigID), Valid: true},
            CreatedAt:   deck.CreatedAt,
            ModifiedAt:  deck.ModifiedAt,
            Usn:         usn,
        })
        if err != nil {
            return 0, fmt.Errorf("failed to upsert deck %d: %w", deck.ID, err)
        }
    }
    
    // Apply Notes
    for _, note := range payload.Notes {
        err := qtx.UpsertNote(ctx, UpsertNoteParams{
            ID:     note.ID,
            UserID: int64(userID),
            Guid:   note.GUID,
            Mid:    note.MID,
            Mod:    note.Mod,
            Usn:    usn,
            Tags:   note.Tags,
            Flds:   note.Flds,
            Sfld:   note.Sfld,
            Csum:   note.Csum,
            Flags:  int64(note.Flags),
            Data:   note.Data,
        })
        if err != nil {
             return 0, fmt.Errorf("failed to upsert note %d: %w", note.ID, err)
        }
    }
    
    // Apply Cards
    for _, card := range payload.Cards {
        err := qtx.UpsertCard(ctx, UpsertCardParams{
            ID:             card.ID,
            UserID:         int64(userID),
            NoteID:         card.NoteID,
            DeckID:         card.DeckID,
            Ordinal:        int64(card.Ordinal),
            ModifiedAt:     card.ModifiedAt,
            Usn:            usn,
            State:          int64(card.State),
            Queue:          int64(card.Queue),
            Due:            card.Due,
            Interval:       int64(card.Interval),
            EaseFactor:     int64(card.EaseFactor),
            Reps:           int64(card.Reps),
            Lapses:         int64(card.Lapses),
            LeftCount:      int64(card.LeftCount),
            OriginalDue:    card.OriginalDue,
            OriginalDeckID: card.OriginalDeckID,
            Flags:          int64(card.Flags),
            Data:           card.Data,
        })
        if err != nil {
            return 0, fmt.Errorf("failed to upsert card %d: %w", card.ID, err)
        }
    }
    
    // Apply Graves
    for _, grave := range payload.Graves {
        // Record
        err := qtx.RecordGrave(ctx, RecordGraveParams{
            UserID: int64(userID),
            Usn:    usn,
            Oid:    grave.OID,
            Type:   int64(grave.Type),
        })
        if err != nil {
            return 0, fmt.Errorf("failed to record grave %d: %w", grave.OID, err)
        }
        
        // Execute deletion
        switch grave.Type {
        case 0: // Card
            err = qtx.DeleteSpecificCard(ctx, DeleteSpecificCardParams{ID: grave.OID, UserID: int64(userID)})
        case 1: // Note
            err = qtx.DeleteSpecificNote(ctx, DeleteSpecificNoteParams{ID: grave.OID, UserID: int64(userID)})
        case 2: // Deck
            err = qtx.DeleteSpecificDeck(ctx, DeleteSpecificDeckParams{ID: grave.OID, UserID: int64(userID)})
        }
        if err != nil {
            // We ignore errors on delete (idempotency), but logging would be good.
            // For now, continue.
        }
    }
    
    if err := tx.Commit(); err != nil {
        return 0, err
    }
    
    return int(usn), nil
}

// SyncCard represents a synced card
type SyncCard struct {
	ID             int64  `json:"id"`
	NoteID         int64  `json:"note_id"`
	DeckID         int64  `json:"deck_id"`
	Ordinal        int    `json:"ordinal"`
	ModifiedAt     int64  `json:"modified_at"`
	USN            int    `json:"usn"`
	State          int    `json:"state"`
	Queue          int    `json:"queue"`
	Due            int64  `json:"due"`
	Interval       int    `json:"interval"`
	EaseFactor     int    `json:"ease_factor"`
	Reps           int    `json:"reps"`
	Lapses         int    `json:"lapses"`
	LeftCount      int    `json:"left_count"`
	OriginalDue    int64  `json:"original_due"`
	OriginalDeckID int64  `json:"original_deck_id"`
	Flags          int    `json:"flags"`
	Data           string `json:"data"`
}

// SyncGrave represents a deleted item
type SyncGrave struct {
	OID  int64 `json:"oid"`
	Type int   `json:"type"` // 0=card, 1=note, 2=deck
}

// InitSyncSchema applies sync-specific tables
func (r *Repository) InitSyncSchema() error {
	schema, err := os.ReadFile("internal/database/sync_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read sync_schema.sql: %w", err)
	}

	_, err = r.DB.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to apply sync schema: %w", err)
	}

	// Manual Migration for FSRS columns (idempotent-ish)
	// We ignore "duplicate column" errors
	r.DB.Exec(`ALTER TABLE user_cards ADD COLUMN stability REAL DEFAULT 0`)
	r.DB.Exec(`ALTER TABLE user_cards ADD COLUMN difficulty REAL DEFAULT 0`)

	return nil
}

// GetSyncMeta returns user's current sync status
func (r *Repository) GetSyncMeta(userID int) (*SyncMeta, error) {
    ctx := context.Background()
    // Attempt to get existing
    meta, err := r.Q.GetSyncMeta(ctx, int64(userID))
    if errors.Is(err, sql.ErrNoRows) {
        // Create initial
        if err := r.Q.CreateSyncMeta(ctx, int64(userID)); err != nil {
            return nil, err
        }
        return &SyncMeta{UserID: userID, USN: 0}, nil
    }
    if err != nil {
        return nil, err
    }
    
    res := &SyncMeta{
        UserID: userID,
        USN:    int(meta.Usn),
    }
    if meta.LastSync.Valid {
        res.LastSync = meta.LastSync.Time
    }
    return res, nil
}

// IncrementUSN atomically increments user's USN and returns new value
func (r *Repository) IncrementUSN(userID int) (int, error) {
    ctx := context.Background()
    if err := r.Q.CreateSyncMeta(ctx, int64(userID)); err != nil {
        return 0, err
    }
    
    usn, err := r.Q.UpdateUSN(ctx, int64(userID))
    if err != nil {
        return 0, err
    }
    return int(usn), nil
}

// UpsertDeck inserts or updates a deck for a user
func (r *Repository) UpsertDeck(userID int, deck *SyncDeck, usn int) error {
    return r.Q.UpsertDeck(context.Background(), UpsertDeckParams{
        ID:          deck.ID,
        UserID:      int64(userID),
        Name:        deck.Name,
        Description: sql.NullString{String: deck.Description, Valid: deck.Description != ""},
        ConfigID:    sql.NullInt64{Int64: int64(deck.ConfigID), Valid: true},
        CreatedAt:   deck.CreatedAt,
        ModifiedAt:  deck.ModifiedAt,
        Usn:         int64(usn),
    })
}

// UpsertNote inserts or updates a note for a user
func (r *Repository) UpsertNote(userID int, note *SyncNote, usn int) error {
    return r.Q.UpsertNote(context.Background(), UpsertNoteParams{
        ID:     note.ID,
        UserID: int64(userID),
        Guid:   note.GUID,
        Mid:    note.MID,
        Mod:    note.Mod,
        Usn:    int64(usn),
        Tags:   note.Tags,
        Flds:   note.Flds,
        Sfld:   note.Sfld,
        Csum:   note.Csum,
        Flags:  int64(note.Flags),
        Data:   note.Data,
    })
}

// UpsertCard inserts or updates a card for a user
func (r *Repository) UpsertCard(userID int, card *SyncCard, usn int) error {
    return r.Q.UpsertCard(context.Background(), UpsertCardParams{
        ID:             card.ID,
        UserID:         int64(userID),
        NoteID:         card.NoteID,
        DeckID:         card.DeckID,
        Ordinal:        int64(card.Ordinal),
        ModifiedAt:     card.ModifiedAt,
        Usn:            int64(usn),
        State:          int64(card.State),
        Queue:          int64(card.Queue),
        Due:            card.Due,
        Interval:       int64(card.Interval),
        EaseFactor:     int64(card.EaseFactor),
        Reps:           int64(card.Reps),
        Lapses:         int64(card.Lapses),
        LeftCount:      int64(card.LeftCount),
        OriginalDue:    card.OriginalDue,
        OriginalDeckID: card.OriginalDeckID,
        Flags:          int64(card.Flags),
        Data:           card.Data,
    })
}

// RecordGrave records a deleted item
func (r *Repository) RecordGrave(userID int, oid int64, itemType int, usn int) error {
    return r.Q.RecordGrave(context.Background(), RecordGraveParams{
        UserID: int64(userID),
        Usn:    int64(usn),
        Oid:    oid,
        Type:   int64(itemType),
    })
}


// GetDecksSince returns decks modified since the given USN
func (r *Repository) GetDecksSince(userID, sinceUSN int) ([]SyncDeck, error) {
    dbDecks, err := r.Q.GetDecksSince(context.Background(), GetDecksSinceParams{
        UserID: int64(userID),
        Usn:    int64(sinceUSN),
    })
    if err != nil {
        return nil, err
    }
    
    decks := make([]SyncDeck, len(dbDecks))
    for i, d := range dbDecks {
        decks[i] = SyncDeck{
            ID:          d.ID,
            Name:        d.Name,
            Description: d.Description.String,
            ConfigID:    int(d.ConfigID.Int64),
            CreatedAt:   d.CreatedAt,
            ModifiedAt:  d.ModifiedAt,
            USN:         int(d.Usn),
        }
    }
    return decks, nil
}

// GetNotesSince returns notes modified since the given USN
func (r *Repository) GetNotesSince(userID, sinceUSN int) ([]SyncNote, error) {
    dbNotes, err := r.Q.GetNotesSince(context.Background(), GetNotesSinceParams{
        UserID: int64(userID),
        Usn:    int64(sinceUSN),
    })
    if err != nil {
        return nil, err
    }
    
    notes := make([]SyncNote, len(dbNotes))
    for i, n := range dbNotes {
        notes[i] = SyncNote{
            ID:    n.ID,
            GUID:  n.Guid,
            MID:   n.Mid,
            Mod:   n.Mod,
            USN:   int(n.Usn),
            Tags:  n.Tags,
            Flds:  n.Flds,
            Sfld:  n.Sfld,
            Csum:  n.Csum,
            Flags: int(n.Flags),
            Data:  n.Data,
        }
    }
    return notes, nil
}

// GetCardsSince returns cards modified since the given USN
func (r *Repository) GetCardsSince(userID, sinceUSN int) ([]SyncCard, error) {
    dbCards, err := r.Q.GetCardsSince(context.Background(), GetCardsSinceParams{
        UserID: int64(userID),
        Usn:    int64(sinceUSN),
    })
    if err != nil {
        return nil, err
    }
    
    cards := make([]SyncCard, len(dbCards))
    for i, c := range dbCards {
        cards[i] = SyncCard{
            ID:             c.ID,
            NoteID:         c.NoteID,
            DeckID:         c.DeckID,
            Ordinal:        int(c.Ordinal),
            ModifiedAt:     c.ModifiedAt,
            USN:            int(c.Usn),
            State:          int(c.State),
            Queue:          int(c.Queue),
            Due:            c.Due,
            Interval:       int(c.Interval),
            EaseFactor:     int(c.EaseFactor),
            Reps:           int(c.Reps),
            Lapses:         int(c.Lapses),
            LeftCount:      int(c.LeftCount),
            OriginalDue:    c.OriginalDue,
            OriginalDeckID: c.OriginalDeckID,
            Flags:          int(c.Flags),
            Data:           c.Data,
        }
    }
    return cards, nil
}

// GetGravesSince returns deleted items since the given USN
func (r *Repository) GetGravesSince(userID, sinceUSN int) ([]SyncGrave, error) {
    dbGraves, err := r.Q.GetGravesSince(context.Background(), GetGravesSinceParams{
        UserID: int64(userID),
        Usn:    int64(sinceUSN),
    })
    if err != nil {
        return nil, err
    }
    
    graves := make([]SyncGrave, len(dbGraves))
    for i, g := range dbGraves {
        graves[i] = SyncGrave{
            OID:  g.Oid,
            Type: int(g.Type),
        }
    }
    return graves, nil
}

// ApplyGrave deletes an entity based on grave record
func (r *Repository) ApplyGrave(userID int, grave SyncGrave) error {
    ctx := context.Background()
    uid := int64(userID)
    
	switch grave.Type {
	case 0: // Card
        return r.Q.DeleteSpecificCard(ctx, DeleteSpecificCardParams{ID: grave.OID, UserID: uid})
	case 1: // Note
        return r.Q.DeleteSpecificNote(ctx, DeleteSpecificNoteParams{ID: grave.OID, UserID: uid})
	case 2: // Deck
        return r.Q.DeleteSpecificDeck(ctx, DeleteSpecificDeckParams{ID: grave.OID, UserID: uid})
	default:
		return fmt.Errorf("unknown grave type: %d", grave.Type)
	}
}

// DeleteUserData clears all sync data for a user (for full sync reset)
func (r *Repository) DeleteUserData(userID int) error {
    ctx := context.Background()
    uid := int64(userID)
    
    if err := r.Q.DeleteUserCards(ctx, uid); err != nil { return err }
    if err := r.Q.DeleteUserNotes(ctx, uid); err != nil { return err }
    if err := r.Q.DeleteUserDecks(ctx, uid); err != nil { return err }
    if err := r.Q.DeleteUserGraves(ctx, uid); err != nil { return err }
    if err := r.Q.DeleteUserMedia(ctx, uid); err != nil { return err }
    
	return r.Q.ResetUserUSN(ctx, uid)
}
