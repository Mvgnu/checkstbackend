package database

import "time"

// GroupDeck represents a deck shared within a group
type GroupDeck struct {
    ID        int       `json:"id"`
    GroupID   int       `json:"group_id"`
    UploaderID int      `json:"uploader_id"`
    Name      string    `json:"name"`
    CardCount int       `json:"card_count"`
    DeckData  string    `json:"deck_data,omitempty"` // Only included in download
    CreatedAt time.Time `json:"created_at"`
}

// CreateGroupDeck adds a deck to a group
func CreateGroupDeck(groupID, uploaderID int, name string, cardCount int, deckData string) (*GroupDeck, error) {
    query := `INSERT INTO group_decks (group_id, uploader_id, name, card_count, deck_data) VALUES (?, ?, ?, ?, ?)`
    result, err := DB.Exec(query, groupID, uploaderID, name, cardCount, deckData)
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
        CreatedAt:  time.Now(),
    }, nil
}

// ListGroupDecks returns all decks shared in a group (without deck_data for efficiency)
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

// GetGroupDeck returns a specific deck with its full content
func GetGroupDeck(deckID int) (*GroupDeck, error) {
    query := `SELECT id, group_id, uploader_id, name, card_count, deck_data, created_at FROM group_decks WHERE id = ?`
    
    var d GroupDeck
    err := DB.QueryRow(query, deckID).Scan(&d.ID, &d.GroupID, &d.UploaderID, &d.Name, &d.CardCount, &d.DeckData, &d.CreatedAt)
    if err != nil {
        return nil, err
    }
    
    return &d, nil
}
