package database

import "time"

// GroupDeck represents a deck shared within a group
type GroupDeck struct {
    ID        int       `json:"id"`
    GroupID   int       `json:"group_id"`
    UploaderID int      `json:"uploader_id"`
    Name      string    `json:"name"`
    CardCount int       `json:"card_count"`
    R2Key     string    `json:"r2_key,omitempty"` // Path in R2 bucket
    CreatedAt time.Time `json:"created_at"`
}

// CreateGroupDeck adds a deck to a group (now just metadata + R2 path)
func CreateGroupDeck(groupID, uploaderID int, name string, cardCount int, r2Key string) (*GroupDeck, error) {
    query := `INSERT INTO group_decks (group_id, uploader_id, name, card_count, r2_key) VALUES (?, ?, ?, ?, ?)`
    result, err := DB.Exec(query, groupID, uploaderID, name, cardCount, r2Key)
    if err != nil {
        return nil, err
    }
    
    id, _ := result.LastInsertId()
    return &GroupDeck{
        ID:         int(id),
        GroupID:    groupID,
        UploaderID: uploaderID,
        Name:       name,
        CardCount:  cardCount,
        R2Key:      r2Key,
        CreatedAt:  time.Now(),
    }, nil
}

// ListGroupDecks returns all decks shared in a group
func ListGroupDecks(groupID int) ([]GroupDeck, error) {
    query := `SELECT id, group_id, uploader_id, name, card_count, created_at FROM group_decks WHERE group_id = ? ORDER BY created_at DESC`
    
    rows, err := DB.Query(query, groupID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var decks []GroupDeck
    for rows.Next() {
        var d GroupDeck
        if err := rows.Scan(&d.ID, &d.GroupID, &d.UploaderID, &d.Name, &d.CardCount, &d.CreatedAt); err != nil {
            continue
        }
        decks = append(decks, d)
    }
    
    return decks, nil
}

// GetGroupDeck returns a specific deck metadata (including R2 key for download link generation)
func GetGroupDeck(deckID int) (*GroupDeck, error) {
    query := `SELECT id, group_id, uploader_id, name, card_count, r2_key, created_at FROM group_decks WHERE id = ?`
    
    var d GroupDeck
    err := DB.QueryRow(query, deckID).Scan(&d.ID, &d.GroupID, &d.UploaderID, &d.Name, &d.CardCount, &d.R2Key, &d.CreatedAt)
    if err != nil {
        return nil, err
    }
    
    return &d, nil
}
