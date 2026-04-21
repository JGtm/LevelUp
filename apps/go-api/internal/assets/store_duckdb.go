package assets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/platform/duckdb"
)

// DuckDBIndexStore implémente IndexStore via une table unifiée asset_index
// dans metadata.duckdb.
//
// Table : asset_index (kind, title_id, id, variant, lang, url, local_path, content_hash, fetched_at, raw_json)
// PK    : (kind, title_id, id, variant, lang)
//
// Toutes les écritures passent par le WriteQueue (voir write_queue.go).
// Cette struct ne doit jamais écrire directement ; elle expose PersistIndex
// comme interface pour que le WriteQueue l'appelle depuis sa goroutine dédiée.
type DuckDBIndexStore struct {
	dbPath string
	mu     sync.Mutex
	db     *duckdb.DB
}

// NewDuckDBIndexStore crée un DuckDBIndexStore pour le chemin de DB donné.
// La connexion est ouverte de façon lazy à la première utilisation.
func NewDuckDBIndexStore(dbPath string) *DuckDBIndexStore {
	return &DuckDBIndexStore{dbPath: dbPath}
}

// Available retourne true si la DB est accessible.
// Utilise le cache de connexion de duckdb.OpenReadWrite (non bloquant si déjà ouvert).
func (s *DuckDBIndexStore) Available(ctx context.Context) bool {
	db, err := s.openDB()
	if err != nil {
		return false
	}
	// Ping léger via la connexion existante.
	row := db.QueryRow(ctx, `SELECT 1`)
	var v int
	return row.Scan(&v) == nil
}

// LookupIndex retourne l'entrée d'index pour la ref donnée, nil si absente.
func (s *DuckDBIndexStore) LookupIndex(ctx context.Context, ref Ref) (*IndexEntry, error) {
	db, err := s.openDB()
	if err != nil {
		return nil, fmt.Errorf("duckdb index: open: %w", err)
	}

	row := db.QueryRow(ctx, `
		SELECT kind, title_id, id, variant, lang, url, local_path, content_hash, fetched_at, raw_json
		FROM asset_index
		WHERE kind = ? AND title_id = ? AND id = ? AND variant = ? AND lang = ?
		LIMIT 1`,
		string(ref.Kind), ref.TitleID, ref.ID, ref.Variant, ref.Lang,
	)

	var kind, titleID, id, variant, lang, url, localPath, hash string
	var fetchedAt time.Time
	var rawJSON sql.NullString
	err = row.Scan(&kind, &titleID, &id, &variant, &lang, &url, &localPath, &hash, &fetchedAt, &rawJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("duckdb index: lookup: %w", err)
	}

	entry := &IndexEntry{
		Ref: Ref{
			Kind:    Kind(kind),
			TitleID: titleID,
			ID:      id,
			Variant: variant,
			Lang:    lang,
		},
		URL:         url,
		LocalPath:   localPath,
		ContentHash: hash,
		FetchedAt:   fetchedAt,
	}
	if rawJSON.Valid {
		entry.RawJSON = []byte(rawJSON.String)
	}
	return entry, nil
}

// PersistIndex insère ou met à jour une entrée dans asset_index.
// Appelé exclusivement par le WriteQueue (1 goroutine writer).
func (s *DuckDBIndexStore) PersistIndex(ctx context.Context, ref Ref, e IndexEntry) error {
	db, err := s.openDB()
	if err != nil {
		return fmt.Errorf("duckdb index: open for persist: %w", err)
	}

	var rawJSON any
	if len(e.RawJSON) > 0 {
		rawJSON = string(e.RawJSON)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO asset_index
			(kind, title_id, id, variant, lang, url, local_path, content_hash, fetched_at, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (kind, title_id, id, variant, lang) DO UPDATE SET
			url          = excluded.url,
			local_path   = excluded.local_path,
			content_hash = excluded.content_hash,
			fetched_at   = excluded.fetched_at,
			raw_json     = excluded.raw_json`,
		string(ref.Kind), ref.TitleID, ref.ID, ref.Variant, ref.Lang,
		e.URL, e.LocalPath, e.ContentHash, e.FetchedAt, rawJSON,
	)
	if err != nil {
		return fmt.Errorf("duckdb index: persist: %w", err)
	}
	return nil
}

// EnsureTable crée la table asset_index si elle n'existe pas.
// Idempotent — appelé au démarrage via la couche migration.
func (s *DuckDBIndexStore) EnsureTable(ctx context.Context) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS asset_index (
			kind         VARCHAR NOT NULL,
			title_id     VARCHAR NOT NULL DEFAULT '',
			id           VARCHAR NOT NULL DEFAULT '',
			variant      VARCHAR NOT NULL DEFAULT '',
			lang         VARCHAR NOT NULL DEFAULT '',
			url          VARCHAR NOT NULL DEFAULT '',
			local_path   VARCHAR NOT NULL DEFAULT '',
			content_hash VARCHAR NOT NULL DEFAULT '',
			fetched_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
			raw_json     TEXT,
			PRIMARY KEY (kind, title_id, id, variant, lang)
		)`)
	if err != nil {
		return fmt.Errorf("duckdb index: ensure table: %w", err)
	}
	return nil
}

// Close libère la connexion DuckDB.
func (s *DuckDBIndexStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
}

// openDB ouvre (ou réutilise) la connexion RW à metadata.duckdb.
func (s *DuckDBIndexStore) openDB() (*duckdb.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db, nil
	}
	db, err := duckdb.OpenReadWrite(s.dbPath)
	if err != nil {
		return nil, err
	}
	s.db = db
	return db, nil
}

// isLockError retourne true si l'erreur correspond à un verrou DuckDB.
func isLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "lock") || strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "could not set lock")
}
