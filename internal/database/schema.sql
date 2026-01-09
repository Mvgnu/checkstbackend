CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    username TEXT UNIQUE NOT NULL,
    avatar_url TEXT,
    university TEXT,
    degree TEXT,
    xp INTEGER DEFAULT 0,
    level INTEGER DEFAULT 1,
    streak INTEGER DEFAULT 0,
    unlocked_achievements TEXT DEFAULT '[]',
    subscription_status TEXT DEFAULT 'free', -- free, pro, group_host
    subscription_expiry DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    university TEXT, -- Optional: limit group to a uni
    degree TEXT,    -- Optional: limit group to a degree
    creator_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(creator_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role TEXT DEFAULT 'member', -- member, admin
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY(group_id) REFERENCES groups(id),
    FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS shared_decks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL, -- Path to storage
    author_id INTEGER NOT NULL,
    group_id INTEGER, -- Optional: if null, public? Or strictly group-based? Let's say null = public.
    downloads INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(author_id) REFERENCES users(id),
    FOREIGN KEY(group_id) REFERENCES groups(id)
);

CREATE TABLE IF NOT EXISTS group_decks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    uploader_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    card_count INTEGER DEFAULT 0,
    r2_key TEXT, -- Path to .apkg file in R2
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(group_id) REFERENCES groups(id),
    FOREIGN KEY(uploader_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS deck_access (
    deck_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role TEXT DEFAULT 'viewer', -- owner, editor, viewer
    granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (deck_id, user_id),
    FOREIGN KEY(user_id) REFERENCES users(id)
    -- Note: deck_id refers to shared_decks or group_decks? 
    -- For now assuming separate logic for paid hosting of specific deck IDs
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_shared_decks_group ON shared_decks(group_id);
CREATE INDEX IF NOT EXISTS idx_group_decks_group ON group_decks(group_id);

-- Subscriptions for IAP tracking
CREATE TABLE IF NOT EXISTS subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    product_id TEXT NOT NULL,
    transaction_id TEXT UNIQUE NOT NULL,
    expires_at DATETIME NOT NULL,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_transaction ON subscriptions(transaction_id);
