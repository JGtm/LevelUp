// Package duckdb — PlayerPool : pool de connexions DuckDB par joueur.
//
// Architecture :
//   - Player : stats.duckdb en lecture/écriture (sync engine et handlers HTTP) avec shared attaché.
//   - ReadDB() retourne Player (connexion unique — DuckDB interdit RW+RO sur le même fichier).
//   - Shared : shared_matches_v2.duckdb en lecture seule.
//   - Metadata : metadata.duckdb en lecture seule (PersistSink ouvre sa propre connexion RW).
//   - SharedSocial : shared_social.duckdb en lecture/écriture (médias, likes, favoris).
//   - Les migrations shared sont exécutées par runMigrations() dans main.go, pas ici.
//   - Le pool est global au processus et threadsafe (sync.Map + singleflight).
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
	Gamertag           string
	XUID               string
	TitleSlug          string // Sprint 44 : namespace titre (ex: "halo_infinite")
	PlayerDBPath       string
	SharedDBPath       string
	MetaDBPath         string
	SharedSocialDBPath string // shared_social.duckdb (médias, likes, favoris)
	UserTimezone       string // timezone IANA pour la lecture des TIMESTAMP (ex: "Europe/Paris")
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
	Player       *DB // stats.duckdb du joueur (RW, avec shared attaché) — sync engine et handlers HTTP
	Shared       *DB // shared_matches_v2.duckdb
	SharedSocial *DB // shared_social.duckdb (médias, likes, favoris de matchs)
	Metadata     *DB // metadata.duckdb (RO)
	XUID         string
	Gamertag     string
	TitleSlug    string // Sprint 44 : titre associé
}

// ReadDB retourne la connexion de lecture (Player RW — connexion unique par joueur).
func (pdb *PlayerDB) ReadDB() *DB {
	return pdb.Player
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

// IteratePool parcourt tous les PlayerDB ouverts dans le pool.
// La fonction f reçoit chaque PlayerDB ; retourner false arrête l'itération.
func IteratePool(f func(*PlayerDB) bool) {
	globalPool.Range(func(_, value any) bool {
		return f(value.(*PlayerDB))
	})
}

// CloseAll ferme toutes les connexions du pool. À appeler au shutdown.
func CloseAll() {
	globalPool.Range(func(key, value any) bool {
		pdb := value.(*PlayerDB)
		_ = pdb.Player.Close()

		_ = pdb.Shared.Close()
		if pdb.SharedSocial != nil {
			_ = pdb.SharedSocial.Close()
		}
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

	playerDB, err := OpenReadWrite(cfg.PlayerDBPath, cfg.UserTimezone)
	if err != nil {
		return nil, fmt.Errorf("pool: open player db %s: %w", cfg.Gamertag, err)
	}

	sharedDB, err := OpenReadOnly(cfg.SharedDBPath, cfg.UserTimezone)
	if err != nil {
		_ = playerDB.Close()
		return nil, fmt.Errorf("pool: open shared db: %w", err)
	}
	// Les migrations shared sont gérées par runMigrations() dans main.go.

	// metadata.duckdb est ouverte en RW partagé : le DuckDBIndexStore (assets) a besoin d'y
	// écrire (table asset_index). Les deux utilisent la clé cache "rw:path", donc une seule
	// sql.DB est créée. OpenReadWriteShared garantit maxOpenConns=4 pour les lectures concurrentes.
	metaDB, err := OpenReadWriteShared(cfg.MetaDBPath, cfg.UserTimezone)
	if err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		return nil, fmt.Errorf("pool: open metadata db: %w", err)
	}

	// SharedSocial est optionnel : absent si le fichier n'existe pas encore.
	// Ouvert en read-write pour permettre les écritures (favoris, likes).
	var socialDB *DB
	if cfg.SharedSocialDBPath != "" {
		socialDB, err = OpenReadWrite(cfg.SharedSocialDBPath, cfg.UserTimezone)
		if err != nil {
			// Non bloquant : la DB sera créée lors de la prochaine migration.
			socialDB = nil
		} else {
			// Appliquer les migrations shared_social (idempotentes).
			_ = migration.RunForDB(socialDB.SQLDb(), migration.TargetSharedSocial)
		}
	}

	// ATTACH shared sur la connexion player RW pour les requêtes join (sync engine et handlers HTTP)
	if err := attachShared(ctx, playerDB, cfg.SharedDBPath); err != nil {
		_ = playerDB.Close()
		_ = sharedDB.Close()
		_ = metaDB.Close()
		return nil, fmt.Errorf("pool: attach shared on player db: %w", err)
	}

	// ATTACH shared_matches_v2 sur SharedSocial pour les JOIN match_registry dans Q37.
	if socialDB != nil && cfg.SharedDBPath != "" {
		if err := attachShared(ctx, socialDB, cfg.SharedDBPath); err != nil {
			// Non bloquant : les colonnes map/mode seront NULL.
			_ = err
		}
	}

	return &PlayerDB{
		Player:       playerDB,
		Shared:       sharedDB,
		SharedSocial: socialDB,
		Metadata:     metaDB,
		XUID:         cfg.XUID,
		Gamertag:     cfg.Gamertag,
		TitleSlug:    cfg.TitleSlug,
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

// XUID du joueur depuis sync_meta (fallback si cfg.XUID vide).
func ResolveXUID(ctx context.Context, db *DB) (string, error) {
	var xuid string
	err := db.QueryRow(ctx, "SELECT value FROM sync_meta WHERE key = 'xuid'").Scan(&xuid)
	if err != nil {
		return "", fmt.Errorf("resolve xuid: %w", err)
	}
	return xuid, nil
}
