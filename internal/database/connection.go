package database

import (
	"database/sql"
	"fmt"
	"os"
	
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
func ReadSchemaFile(path string) ([]byte, error) {
    // This is a placeholder. Real implementation should read standard file.
    // In previous code (sync.go), os.ReadFile was used.
    // We will use os.ReadFile in applySchema actually.
    return nil, nil // Not used if I inline
}
