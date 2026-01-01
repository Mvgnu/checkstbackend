-- User sync collection tracking
CREATE TABLE IF NOT EXISTS user_collections (
    user_id INTEGER PRIMARY KEY,
    usn INTEGER NOT NULL DEFAULT 0,
    last_sync DATETIME,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- User's synced decks
CREATE TABLE IF NOT EXISTS user_decks (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    config_id INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    usn INTEGER NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- User's synced notes (card content)
CREATE TABLE IF NOT EXISTS user_notes (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    guid TEXT NOT NULL,
    mid INTEGER NOT NULL,
    mod INTEGER NOT NULL,
    usn INTEGER NOT NULL,
    tags TEXT NOT NULL DEFAULT '',
    flds TEXT NOT NULL,
    sfld TEXT NOT NULL DEFAULT '',
    csum INTEGER NOT NULL DEFAULT 0,
    flags INTEGER NOT NULL DEFAULT 0,
    data TEXT NOT NULL DEFAULT '',
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- User's synced cards (scheduling data)
CREATE TABLE IF NOT EXISTS user_cards (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL,
    note_id INTEGER NOT NULL,
    deck_id INTEGER NOT NULL,
    ordinal INTEGER NOT NULL DEFAULT 0,
    modified_at INTEGER NOT NULL,
    usn INTEGER NOT NULL,
    state INTEGER NOT NULL DEFAULT 0,
    queue INTEGER NOT NULL DEFAULT 0,
    due INTEGER NOT NULL DEFAULT 0,
    interval INTEGER NOT NULL DEFAULT 0,
    ease_factor INTEGER NOT NULL DEFAULT 2500,
    reps INTEGER NOT NULL DEFAULT 0,
    lapses INTEGER NOT NULL DEFAULT 0,
    left_count INTEGER NOT NULL DEFAULT 0,
    original_due INTEGER NOT NULL DEFAULT 0,
    original_deck_id INTEGER NOT NULL DEFAULT 0,
    flags INTEGER NOT NULL DEFAULT 0,
    data TEXT NOT NULL DEFAULT '',
    stability REAL DEFAULT 0,
    difficulty REAL DEFAULT 0,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- Deleted items pending sync (tombstones)
CREATE TABLE IF NOT EXISTS user_graves (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    usn INTEGER NOT NULL,
    oid INTEGER NOT NULL,
    type INTEGER NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- Media file tracking
CREATE TABLE IF NOT EXISTS user_media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    filename TEXT NOT NULL,
    hash TEXT NOT NULL,
    size INTEGER NOT NULL,
    usn INTEGER NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id),
    UNIQUE(user_id, hash)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_user_decks_user ON user_decks(user_id);
CREATE INDEX IF NOT EXISTS idx_user_decks_usn ON user_decks(user_id, usn);
CREATE INDEX IF NOT EXISTS idx_user_notes_user ON user_notes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_notes_usn ON user_notes(user_id, usn);
CREATE INDEX IF NOT EXISTS idx_user_cards_user ON user_cards(user_id);
CREATE INDEX IF NOT EXISTS idx_user_cards_usn ON user_cards(user_id, usn);
CREATE INDEX IF NOT EXISTS idx_user_graves_user ON user_graves(user_id, usn);
CREATE INDEX IF NOT EXISTS idx_user_media_hash ON user_media(user_id, hash);
