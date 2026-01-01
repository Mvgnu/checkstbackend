package database

import (
    "time"
)

type SharedDeck struct {
    ID          int       `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description,omitempty"`
    FilePath    string    `json:"-"`
    AuthorID    int       `json:"author_id"`
    GroupID     *int      `json:"group_id,omitempty"`
    Downloads   int       `json:"downloads"`
    CreatedAt   time.Time `json:"created_at"`
}

func CreateSharedDeck(title, description, filePath string, authorID int, groupID *int) (*SharedDeck, error) {
    query := `INSERT INTO shared_decks (title, description, file_path, author_id, group_id) VALUES (?, ?, ?, ?, ?)`
    result, err := DB.Exec(query, title, description, filePath, authorID, groupID)
    if err != nil {
        return nil, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return nil, err
    }

    return &SharedDeck{
        ID:          int(id),
        Title:       title,
        Description: description,
        FilePath:    filePath,
        AuthorID:    authorID,
        GroupID:     groupID,
        CreatedAt:   time.Now(),
        Downloads:   0,
    }, nil
}

func ListSharedDecks(groupID int) ([]SharedDeck, error) {
    query := `SELECT id, title, description, author_id, downloads, created_at FROM shared_decks WHERE group_id = ? ORDER BY created_at DESC`
    rows, err := DB.Query(query, groupID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var decks []SharedDeck
    for rows.Next() {
        var d SharedDeck
        if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.AuthorID, &d.Downloads, &d.CreatedAt); err != nil {
            return nil, err
        }
        d.GroupID = &groupID // set explicitly since we filtered by it
        decks = append(decks, d)
    }
    return decks, nil
}

func GetSharedDeck(id int) (*SharedDeck, error) {
    query := `SELECT id, title, description, file_path, author_id, group_id, downloads, created_at FROM shared_decks WHERE id = ?`
    row := DB.QueryRow(query, id)

    var d SharedDeck
    err := row.Scan(&d.ID, &d.Title, &d.Description, &d.FilePath, &d.AuthorID, &d.GroupID, &d.Downloads, &d.CreatedAt)
    if err != nil {
        return nil, err
    }
    return &d, nil
}

func IncrementDownloads(id int) {
    DB.Exec(`UPDATE shared_decks SET downloads = downloads + 1 WHERE id = ?`, id)
}
