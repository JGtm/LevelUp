// Package duckdb — PlayerPool : pool de connexions DuckDB par joueur.
//
// Architecture :
//   - Une connexion read-only par base (player stats.duckdb + shared_matches_v2.duckdb).
//   - ATTACH shared_matches_v2 est réalisé une seule fois à l'init du pool.
//   - Le pool est global au processus et threadsafe (sync.Map + singleflight).
//   - Aucun appel à duckdb.Connect() hors de ce package.
package duckdb

import (
	"context"
	"fmt"
	"sync"

	"levelup/go-api/internal/migration"

	"golang.org/x/sync/singleflight"
)

// PlayerPoolConfig contient les chemins nécessaires pour ouvrir un PlayerDB.
type PlayerPoolConfig struct {
	Gamertag     string
	XUID         string
	TitleSlug    string // Sprint 44 : namespace titre (ex: "halo_infinite")
	PlayerDBPath string
	SharedDBPath string
	MetaDBPath   string
}

// PoolKey retourne la clé unique du pool pour ce joueur.
// Format : "{title_slug}:{gamertag}" ou "{gamertag}" si TitleSlug vide (legacy).
func (c PlayerPoolConfig) PoolKey() string {
	if c.TitleSlug != "" {
		return c.TitleSlug + ":" + c.Gamertag
	}
	return c.Gamertag
}

// PlayerDB regroupe les connexions DB nécessaires pour un joueur.
type PlayerDB struct {
	Player    *DB // stats.duckdb du joueur (avec shared attaché)
	Shared    *DB // shared_matches_v2.duckdb
	Metadata  *DB // metadata.duckdb
	XUID      string
	Gamertag  string
	TitleSlug string // Sprint 44 : titre associé
}

// GlobalPool est le registre process-level des PlayerDB par gamertag (slug).
var globalPool sync.Map // map[string]*PlayerDB

// sfGroup empêche les créations concurrentes de pools pour un même joueur.
var sfGroup singleflight.Group

// GetOrOpen retourne le PlayerDB existant pour ce joueur, ou l'ouvre.
// Thread-safe : singleflight garantit qu'un seul openPlayerDB par clé de pool.
// Clé de pool : "{title_slug}:{gamertag}" (Sprint 44) ou "{gamertag}" (legacy).
func GetOrOpen(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	key := cfg.PoolKey()
	if pdb, ok := globalPool.Load(key); ok {
		return pdb.(*PlayerDB), nil
	}

	// singleflight : une seule goroutine ouvre; les autres attendent le résultat.
	result, err, _ := sfGroup.Do(key, func() (interface{}, error) {
		// Vérifier à nouveau après avoir gagné le lock singleflight.
		if pdb, ok := globalPool.Load(key); ok {
			return pdb.(*PlayerDB), nil
		}
		pdb, openErr := openPlayerDB(ctx, cfg)
		if openErr != nil {
			return nil, openErr
		}
		globalPool.Store(key, pdb)
		return pdb, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*PlayerDB), nil
}

// CloseAll ferme toutes les connexions du pool. À appeler au shutdown.
func CloseAll() {
	globalPool.Range(func(key, value any) bool {
		pdb := value.(*PlayerDB)
		_ = pdb.Player.Close()
		_ = pdb.Shared.Close()
		_ = pdb.Metadata.Close()
		globalPool.Delete(key)
		return true
	})
}

// openPlayerDB ouvre et initialise un PlayerDB complet.
func openPlayerDB(ctx context.Context, cfg PlayerPoolConfig) (*PlayerDB, error) {
	if err := ensurePlayerDBMigrations(cfg.PlayerDBPath); err != nil {
		return nil, fmt.Errorf("pool: migrate player db %s: %w", cfg.Gamertag, err)
	}

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

	// ATTACH metadata sur la connexion player pour les JOIN career_ranks etc.
	if err := attachMeta(ctx, playerDB, cfg.MetaDBPath); err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		_ = metaDB.Close()
		return nil, fmt.Errorf("pool: attach meta on player db: %w", err)
	}

	return &PlayerDB{
		Player:    playerDB,
		Shared:    sharedDB,
		Metadata:  metaDB,
		XUID:      cfg.XUID,
		Gamertag:  cfg.Gamertag,
		TitleSlug: cfg.TitleSlug,
	}, nil
}

func ensurePlayerDBMigrations(path string) error {
	_ = migration.All()

	rwDB, err := OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("open rw: %w", err)
	}
	defer rwDB.Close()

	if err := migration.RunForDB(rwDB.SQLDb(), migration.TargetPlayer); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
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

// attachMeta attache metadata.duckdb sous l'alias "meta" sur une connexion player.
// Permet les JOIN sur meta.career_ranks, meta.weapon_labels, etc.
// Idempotent : ignore l'erreur si déjà attachée.
func attachMeta(ctx context.Context, db *DB, metaPath string) error {
	_, err := db.Exec(ctx,
		fmt.Sprintf("ATTACH '%s' AS meta (READ_ONLY)", metaPath),
	)
	if err != nil {
		// Déjà attachée — vérifier si accessible.
		var count int
		pingErr := db.QueryRow(ctx, "SELECT COUNT(*) FROM meta.career_ranks").Scan(&count)
		if pingErr != nil {
			return fmt.Errorf("attach meta: %w (attach err: %v)", pingErr, err)
		}
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
