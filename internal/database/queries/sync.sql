-- name: GetSyncMeta :one
SELECT usn, last_sync FROM user_collections 
WHERE user_id = ? LIMIT 1;

-- name: CreateSyncMeta :exec
INSERT OR IGNORE INTO user_collections (user_id, usn, last_sync) 
VALUES (?, 0, NULL);

-- name: UpdateUSN :one
UPDATE user_collections 
SET usn = usn + 1, last_sync = CURRENT_TIMESTAMP 
WHERE user_id = ? 
RETURNING usn;

-- name: GetUSN :one
SELECT usn FROM user_collections WHERE user_id = ?;

-- name: UpsertDeck :exec
INSERT INTO user_decks (id, user_id, name, description, config_id, created_at, modified_at, usn)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    config_id = excluded.config_id,
    modified_at = excluded.modified_at,
    usn = excluded.usn;

-- name: UpsertNote :exec
INSERT INTO user_notes (id, user_id, guid, mid, mod, usn, tags, flds, sfld, csum, flags, data)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    guid = excluded.guid,
    mid = excluded.mid,
    mod = excluded.mod,
    usn = excluded.usn,
    tags = excluded.tags,
    flds = excluded.flds,
    sfld = excluded.sfld,
    csum = excluded.csum,
    flags = excluded.flags,
    data = excluded.data;

-- name: UpsertCard :exec
INSERT INTO user_cards (id, user_id, note_id, deck_id, ordinal, modified_at, usn, 
    state, queue, due, interval, ease_factor, reps, lapses, left_count,
    original_due, original_deck_id, flags, data, stability, difficulty)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    note_id = excluded.note_id,
    deck_id = excluded.deck_id,
    ordinal = excluded.ordinal,
    modified_at = excluded.modified_at,
    usn = excluded.usn,
    state = excluded.state,
    queue = excluded.queue,
    due = excluded.due,
    interval = excluded.interval,
    ease_factor = excluded.ease_factor,
    reps = excluded.reps,
    lapses = excluded.lapses,
    left_count = excluded.left_count,
    original_due = excluded.original_due,
    original_deck_id = excluded.original_deck_id,
    flags = excluded.flags,
    data = excluded.data,
    stability = excluded.stability,
    difficulty = excluded.difficulty;

-- name: RecordGrave :exec
INSERT INTO user_graves (user_id, usn, oid, type) 
VALUES (?, ?, ?, ?);

-- name: GetDecksSince :many
SELECT id, name, description, config_id, created_at, modified_at, usn
FROM user_decks
WHERE user_id = ? AND usn > ?
ORDER BY usn;

-- name: GetNotesSince :many
SELECT id, guid, mid, mod, usn, tags, flds, sfld, csum, flags, data
FROM user_notes
WHERE user_id = ? AND usn > ?
ORDER BY usn;

-- name: GetCardsSince :many
SELECT id, note_id, deck_id, ordinal, modified_at, usn, state, queue, due, 
    interval, ease_factor, reps, lapses, left_count, original_due, original_deck_id, flags, data,
    stability, difficulty
FROM user_cards
WHERE user_id = ? AND usn > ?
ORDER BY usn;

-- name: GetGravesSince :many
SELECT oid, type FROM user_graves
WHERE user_id = ? AND usn > ?
ORDER BY usn;

-- name: DeleteUserCards :exec
DELETE FROM user_cards WHERE user_id = ?;

-- name: DeleteUserNotes :exec
DELETE FROM user_notes WHERE user_id = ?;

-- name: DeleteUserDecks :exec
DELETE FROM user_decks WHERE user_id = ?;

-- name: DeleteUserGraves :exec
DELETE FROM user_graves WHERE user_id = ?;

-- name: DeleteUserMedia :exec
DELETE FROM user_media WHERE user_id = ?;

-- name: ResetUserUSN :exec
UPDATE user_collections SET usn = 0, last_sync = NULL WHERE user_id = ?;

-- name: DeleteSpecificCard :exec
DELETE FROM user_cards WHERE id = ? AND user_id = ?;

-- name: DeleteSpecificNote :exec
DELETE FROM user_notes WHERE id = ? AND user_id = ?;

-- name: DeleteSpecificDeck :exec
DELETE FROM user_decks WHERE id = ? AND user_id = ?;
