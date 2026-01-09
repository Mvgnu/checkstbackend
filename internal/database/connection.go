package database

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"time"
	
	_ "github.com/mattn/go-sqlite3"
)

type Repository struct {
    DB *sql.DB
    Q  *Queries
}

var (
    // Deprecated: Use Repository injection instead
    DB *sql.DB
    // Deprecated: Use Repository injection instead
    Q  *Queries
)

func InitDB(dataSourceName string) (*Repository, error) {
	var err error
	DB, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	
	if err = DB.Ping(); err != nil {
		return nil, err
	}
	
    // Initialize sqlc queries
    Q = New(DB)

    // Apply main schema
    if err := applySchema(); err != nil {
        return nil, err
    }
    
    // Auto-Migrate: Add invite_code if missing
    // We ignore error because if it exists, it fails safely (usually)
    // Or we verify if column exists.
    DB.Exec(`ALTER TABLE groups ADD COLUMN invite_code TEXT UNIQUE`)
    DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_invite_code ON groups(invite_code)`)
    
    // Backfill empty invite codes
    // We need to do this for any group that has NULL invite_code
    rows, _ := DB.Query("SELECT id FROM groups WHERE invite_code IS NULL OR invite_code = ''")
    if rows != nil {
        var ids []int
        for rows.Next() {
            var id int
            if err := rows.Scan(&id); err == nil {
                ids = append(ids, id)
            }
        }
        rows.Close()
        
        for _, id := range ids {
            code := generateRandomCode(8)
            DB.Exec("UPDATE groups SET invite_code = ? WHERE id = ?", code, id)
        }
    }

    // Create password_resets table if not exists
    DB.Exec(`CREATE TABLE IF NOT EXISTS password_resets (
        email TEXT NOT NULL PRIMARY KEY,
        code TEXT NOT NULL,
        expires_at DATETIME NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )`)
    
	return &Repository{
        DB: DB,
        Q:  Q,
    }, nil
}

func applySchema() error {
    schema, err := os.ReadFile("internal/database/schema.sql")
    if err != nil {
        return fmt.Errorf("failed to read schema.sql: %w", err)
    }
    
    _, err = DB.Exec(string(schema))
    if err != nil {
        return fmt.Errorf("failed to apply schema: %w", err)
    }
    return nil
}

// ReadSchemaFile is a helper to read files - likely simpler to just embed or read directly as before
// However, since we are in the same package, we can just copy the logic from previous InitSyncSchema or similar
// For now, let's keep it simple. Assuming main.go handles the path correctly relative to cwd.
// But wait, schema.sql reading was in sync.go's InitSyncSchema. 
// I should move ReadSchemaFile or similar helper here if needed, or just inline it.

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func generateRandomCode(length int) string {
    seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[seededRand.Intn(len(charset))]
    }
    return string(b)
}
