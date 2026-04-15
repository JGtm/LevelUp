// Package duckdb — PlayerPool : pool de connexions DuckDB par joueur.
//
// Architecture :
//   - Une connexion read-only par base (player stats.duckdb + shared_matches_v2.duckdb).
//   - ATTACH shared_matches_v2 est réalisé une seule fois à l'init du pool.
//   - Le pool est global au processus et threadsafe (sync.Map).
//   - Aucun appel à duckdb.Connect() hors de ce package.
package duckdb

import (
	"context"
	"fmt"
	"sync"
)

// PlayerDB regroupe les connexions DB nécessaires pour un joueur.
type PlayerDB struct {
	Player   *DB // stats.duckdb du joueur (avec shared attaché)
	Shared   *DB // shared_matches_v2.duckdb
	Metadata *DB // metadata.duckdb
	XUID     string
	Gamertag string
}

// GlobalPool est le registre process-level des PlayerDB par gamertag (slug).
var globalPool sync.Map // map[string]*PlayerDB

// PlayerPoolConfig contient les chemins résolus pour un joueur.
type PlayerPoolConfig struct {
	Gamertag    string
	XUID        string
	PlayerDBPath string // abs path vers stats.duckdb
	SharedDBPath string // abs path vers shared_matches_v2.duckdb
	MetaDBPath   string // abs path vers metadata.duckdb
}

// GetOrOpen retourne le PlayerDB existant pour ce joueur, ou l'ouvre.
// Thread-safe : deux goroutines concurrentes obtiennent la même instance.
func GetOrOpen(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	if pdb, ok := globalPool.Load(cfg.Gamertag); ok {
		return pdb.(*PlayerDB), nil
	}

	// Double-checked locking via sync.Map.LoadOrStore
	pdb, err := openPlayerDB(ctx, cfg)
	if err != nil {
		return nil, err
	}

	actual, loaded := globalPool.LoadOrStore(cfg.Gamertag, pdb)
	if loaded {
		// Une autre goroutine a déjà ouvert — fermer notre doublon.
		_ = pdb.Player.Close()
		return actual.(*PlayerDB), nil
	}
	return pdb, nil
}

// CloseAll ferme toutes les connexions du pool. À appeler au shutdown.
func CloseAll() {
	globalPool.Range(func(key, value any) bool {
		pdb := value.(*PlayerDB)
		_ = pdb.Player.Close()
		globalPool.Delete(key)
		return true
	})
}

// openPlayerDB ouvre et initialise un PlayerDB complet.
func openPlayerDB(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	playerDB, err := OpenReadOnly(cfg.PlayerDBPath)
	if err != nil {
		return nil, fmt.Errorf("pool: open player db %s: %w", cfg.Gamertag, err)
	}

	sharedDB, err := OpenReadOnly(cfg.SharedDBPath)
	if err != nil {
		_ = playerDB.Close()
		return nil, fmt.Errorf("pool: open shared db: %w", err)
	}

	metaDB, err := OpenReadOnly(cfg.MetaDBPath)
	if err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		return nil, fmt.Errorf("pool: open metadata db: %w", err)
	}

	// ATTACH shared sur la connexion player pour les requêtes join
	if err := attachShared(ctx, playerDB, cfg.SharedDBPath); err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		_ = metaDB.Close()
		return nil, fmt.Errorf("pool: attach shared on player db: %w", err)
	}

	return &PlayerDB{
		Player:   playerDB,
		Shared:   sharedDB,
		Metadata: metaDB,
		XUID:     cfg.XUID,
		Gamertag: cfg.Gamertag,
	}, nil
}

// attachShared attache shared_matches_v2.duckdb sur une connexion player.
// Idempotent : ignore l'erreur si déjà attachée.
func attachShared(ctx context.Context, db *DB, sharedPath string) error {
	_, err := db.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS shared (READ_ONLY)", sharedPath),
	)
	if err != nil {
		// Déjà attachée ou autre erreur transitoire — vérifier si la vue est accessible.
		var count int
		pingErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM shared.match_registry").Scan(&count)
		if pingErr != nil {
			return fmt.Errorf("attach shared: %w (attach err: %v)", pingErr, err)
		}
		// Accessible malgré l'erreur d'ATTACH → déjà attachée, OK.
	}
	return nil
}

// XUID du joueur depuis sync_meta (fallback si cfg.XUID vide).
func ResolveXUID(ctx context.Context, db *DB) (string, error) {
	var xuid string
	err := db.QueryRow(ctx, "SELECT value FROM sync_meta WHERE key = 'xuid'").Scan(&xuid)
	if err != nil {
		return "", fmt.Errorf("resolve xuid: %w", err)
	}
	return xuid, nil
}
